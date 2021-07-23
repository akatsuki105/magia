package apu

import (
	"math"

	"github.com/pokemium/magia/pkg/util"
)

const (
	CPU_FREQ_HZ              = 16777216
	SND_FREQUENCY            = 32768 // sample rate
	SND_SAMPLES              = 512
	SAMP_CYCLES              = (CPU_FREQ_HZ / SND_FREQUENCY)
	BUFF_SAMPLES             = ((SND_SAMPLES) * 16 * 2)
	BUFF_SAMPLES_MSK         = ((BUFF_SAMPLES) - 1)
	SAMPLE_TIME      float64 = 1.0 / SND_FREQUENCY
	STREAM_LEN               = (2 * 2 * SND_FREQUENCY / 60) - (2*2*SND_FREQUENCY/60)%4
)

const (
	PSG_MAX = 0x7f
	PSG_MIN = -0x80
)

const (
	SAMP_MAX = 0x1ff
	SAMP_MIN = -0x200
)

var waveSamples, wavePosition byte
var WaveRAM [0x20]byte
var resetSoundChanMap = map[uint32]int{0x65 - 0x60: 0, 0x6d - 0x60: 1, 0x75 - 0x60: 2, 0x7d - 0x60: 3}

type APU struct {
	Enable bool
	buffer [72]byte
	stream []byte
	chans  [4]*SoundChan
}

type SoundChan struct {
	phase                                   bool
	lfsr                                    uint16
	samples, lengthTime, sweepTime, envTime float64
}

func (a *APU) Load32(ofs uint32) uint32 {
	if ofs >= WAVE_RAM && ofs <= WAVE_RAM+0xf {
		bank := (a.Load32(SOUND3CNT_L) >> 2) & 0x10
		idx := (bank ^ 0x10) | (ofs & 0xf)
		return util.LE32(WaveRAM[idx:])
	}

	return util.LE32(a.buffer[ofs:])
}

func (a *APU) Store8(ofs uint32, val byte) {
	if ofs >= WAVE_RAM && ofs <= WAVE_RAM+0xf {
		bank := (a.Load32(SOUND3CNT_L) >> 2) & 0x10
		idx := (bank ^ 0x10) | (ofs & 0xf)
		WaveRAM[idx] = val
	}
	if isResetSoundChan(ofs) {
		a.resetSoundChan(ofs, val)
	}

	switch ofs {
	case SOUND1CNT_L:
		val &= 0x7f
	case SOUND1CNT_L + 1:
		val = 0
	case SOUND1CNT_X + 1, SOUND2CNT_H + 1:
		val &= 0xc7
	case SOUND3CNT_L:
		val &= 0xe0
	case SOUND3CNT_L + 1:
		val = 0
	case SOUND3CNT_X + 1:
		val &= 0xc7
	case SOUND4CNT_L:
		val &= 0x3f
	case SOUND4CNT_H + 1:
		val &= 0xc0
	case SOUNDCNT_L:
		val &= 0x77
	case SOUNDCNT_H:
		val &= 0x0f
	case SOUNDCNT_X:
		val &= 0x80
	case SOUNDCNT_X + 1:
		val = 0
	}

	a.buffer[ofs] = byte(val)
}

func (a *APU) Store16(ofs uint32, val uint16) {
	a.Store8(ofs, byte(val))
	a.Store8(ofs+1, byte(val>>8))
	if ofs == SOUNDCNT_H {
		if util.Bit(val, 11) {
			FifoALen = 0
		}
		if util.Bit(val, 15) {
			FifoBLen = 0
		}
	}
}

func (a *APU) Store32(ofs uint32, val uint32) {
	a.Store8(ofs, byte(val))
	a.Store8(ofs+1, byte(val>>8))
	a.Store8(ofs+2, byte(val>>16))
	a.Store8(ofs+3, byte(val>>24))
}

func isResetSoundChan(addr uint32) bool {
	_, ok := resetSoundChanMap[addr]
	return ok
}

func (a *APU) resetSoundChan(addr uint32, b byte) {
	a._resetSoundChan(resetSoundChanMap[addr], util.Bit(b, 7))
}

func New() *APU {
	return &APU{
		chans: [4]*SoundChan{{}, {}, {}, {}},
	}
}

func (a *APU) SetBuffer(s []byte) {
	a.stream = s
}

func (a *APU) Play() {
	a.Enable = true
	if a.stream == nil {
		return
	}
	if len(a.stream) == 0 {
		return
	}
	a.soundMix()
}

var dutyLookUp = [4]float64{0.125, 0.25, 0.5, 0.75}
var dutyLookUpi = [4]float64{0.875, 0.75, 0.5, 0.25}

func (a *APU) squareSample(ch int) int8 {
	if !a.isSoundChanEnable(ch) {
		return 0
	}

	toneAddr := uint32(SOUND1CNT_H)
	if ch == 1 {
		toneAddr = SOUND2CNT_L
	}

	freqHz := a.Load32(SOUND1CNT_X) & 0b0111_1111_1111
	if ch == 1 {
		freqHz = a.Load32(SOUND2CNT_H) & 0b0111_1111_1111
	}
	frequency := 131072 / float64(2048-freqHz)

	// Full length of the generated wave (if enabled) in seconds
	soundLen := a.Load32(toneAddr) & 0b0011_1111
	length := float64(64-soundLen) / 256

	// Envelope volume change interval in seconds
	envStep := a.Load32(toneAddr) >> 8 & 0b111
	envelopeInterval := float64(envStep) / 64

	cycleSamples := SND_FREQUENCY / frequency // Numbers of samples that a single cycle (wave phase change 1 -> 0) takes at output sample rate

	// Length reached check (if so, just disable the channel and return silence)
	if (ch == 0 && util.Bit(a.Load32(SOUND1CNT_X), 14)) || (ch == 1 && util.Bit(a.Load32(SOUND2CNT_H), 14)) {
		a.chans[ch].lengthTime += SAMPLE_TIME
		if a.chans[ch].lengthTime >= length {
			a.enableSoundChan(ch, false)
			return 0
		}
	}

	// Frequency sweep (Square 1 channel only)
	if ch == 0 {
		sweepTime := (a.Load32(SOUND1CNT_L) >> 4) & 0b111 // 0-7 (0=7.8ms, 7=54.7ms)
		sweepInterval := 0.0078 * float64(sweepTime+1)    // Frquency sweep change interval in seconds

		a.chans[0].sweepTime += SAMPLE_TIME
		if a.chans[0].sweepTime >= sweepInterval {
			a.chans[0].sweepTime -= sweepInterval

			// A Sweep Shift of 0 means that Sweep is disabled
			sweepShift := byte(a.Load32(SOUND1CNT_L) & 7)

			if sweepShift != 0 {
				// X(t) = X(t-1) ± X(t-1)/2^n
				disp := freqHz >> sweepShift // X(t-1)/2^n
				if util.Bit(a.Load32(SOUND1CNT_L), 3) {
					freqHz -= disp
				} else {
					freqHz += disp
				}

				if freqHz < 0x7ff {
					// update frequency
					ctrl := uint16(a.Load32(SOUND1CNT_X))
					ctrl = (ctrl & ^uint16(0x7ff)) | uint16(freqHz)
					a.buffer[SOUND1CNT_X] = byte(ctrl)
					a.buffer[SOUND1CNT_X+1] = byte(ctrl >> 8)
				} else {
					a.enableSoundChan(0, false)
				}
			}
		}
	}

	// Envelope volume
	envelope := uint16((a.Load32(toneAddr) >> 12) & 0xf)
	if envStep > 0 {
		a.chans[ch].envTime += SAMPLE_TIME

		if a.chans[ch].envTime >= envelopeInterval {
			a.chans[ch].envTime -= envelopeInterval

			tone := uint16(a.Load32(toneAddr))
			if util.Bit(tone, 11) {
				if envelope < 0xf {
					envelope++
				}
			} else {
				if envelope > 0 {
					envelope--
				}
			}

			// update envelope
			tone = (tone & ^uint16(0xf000)) | (envelope << 12)
			a.buffer[toneAddr] = byte(tone)
			a.buffer[toneAddr+1] = byte(tone >> 8)
		}
	}

	// Phase change (when the wave goes from Low to High or High to Low, the Square Wave pattern)
	duty := (a.Load32(toneAddr) >> 6) & 0b11
	a.chans[ch].samples++
	if a.chans[ch].phase {
		// 1 -> 0 -_
		phaseChange := cycleSamples * dutyLookUp[duty]
		if a.chans[ch].samples > phaseChange {
			a.chans[ch].samples -= phaseChange
			a.chans[ch].phase = false
		}
	} else {
		// 0 -> 1 _-
		phaseChange := cycleSamples * dutyLookUpi[duty]
		if a.chans[ch].samples > phaseChange {
			a.chans[ch].samples -= phaseChange
			a.chans[ch].phase = true
		}
	}

	if a.chans[ch].phase {
		return int8(float64(envelope) * PSG_MAX / 15)
	}
	return int8(float64(envelope) * PSG_MIN / 15)
}

func (a *APU) enableSoundChan(ch int, enable bool) {
	cntx := a.Load32(SOUNDCNT_X)
	cntx = util.SetBit32(cntx, ch, enable)
	a.buffer[SOUNDCNT_X] = byte(cntx)
}

func (a *APU) IsSoundMasterEnable() bool {
	cntx := byte(a.Load32(SOUNDCNT_X))
	return util.Bit(cntx, 7)
}

func (a *APU) isSoundChanEnable(ch int) bool {
	cntx := byte(a.Load32(SOUNDCNT_X))
	return util.Bit(cntx, ch)
}

// chan3
func (a *APU) waveSample() int8 {
	wave := uint16(a.Load32(SOUND3CNT_L))
	if !(a.isSoundChanEnable(2) && util.Bit(wave, 7)) {
		return 0
	}

	// Actual frequency in Hertz
	rate := a.Load32(SOUND3CNT_X) & 2047
	frequency := 2097152 / (2048 - float64(rate))

	cnth := uint16(a.Load32(SOUND3CNT_H)) // volume

	// Full length of the generated wave (if enabled) in seconds
	soundLen := cnth & 0xff
	length := (256 - float64(soundLen)) / 256.

	// Numbers of samples that a single "cycle" (all entries on Wave RAM) takes at output sample rate
	cycleSamples := SND_FREQUENCY / frequency

	// Length reached check (if so, just disable the channel and return silence)
	if util.Bit(uint16(a.Load32(SOUND3CNT_X)), 14) {
		a.chans[2].lengthTime += SAMPLE_TIME
		if a.chans[2].lengthTime >= length {
			a.enableSoundChan(2, false)
			return 0
		}
	}

	a.chans[2].samples++
	if a.chans[2].samples >= cycleSamples {
		a.chans[2].samples -= cycleSamples

		waveSamples--
		if waveSamples != 0 {
			wavePosition = (wavePosition + 1) & 0b0011_1111
		} else {
			a.waveReset()
		}
	}

	wavedata := WaveRAM[(uint32(wavePosition)>>1)&0x1f]
	sample := (float64((wavedata>>((wavePosition&1)<<2))&0xf) - 0x8) / 8

	switch volume := (cnth >> 13) & 0x7; volume {
	case 0:
		sample = 0 // 0%
	case 1: // 100%
	case 2:
		sample /= 2 // 50%
	case 3:
		sample /= 4 // 25%
	default:
		sample *= 3 / 4 // 75%
	}

	if sample >= 0 {
		return int8(sample / 7 * PSG_MAX)
	}
	return int8(sample / (-8) * PSG_MIN)
}

// chan4
func (a *APU) noiseSample() int8 {
	if !a.isSoundChanEnable(3) {
		return 0
	}

	cnth := a.Load32(SOUND4CNT_H) // ctrl

	// Actual frequency in Hertz (524288 / r / 2^(s+1))
	r, s := float64(cnth&0x7), float64((cnth>>4)&0xf)
	if r == 0 {
		r = 0.5
	}
	frequency := (524288 / r) / math.Pow(2, s+1)

	cntl := a.Load32(SOUND4CNT_L) // env
	// Full length of the generated wave (if enabled) in seconds
	soundLen := cntl & 0x3f
	length := (64 - float64(soundLen)) / 256

	// Length reached check (if so, just disable the channel and return silence)
	if util.Bit(cnth, 14) {
		a.chans[3].lengthTime += SAMPLE_TIME
		if a.chans[3].lengthTime >= length {
			a.enableSoundChan(3, false)
			return 0
		}
	}

	// Envelope volume change interval in seconds
	envStep := (cntl >> 8) & 0x7
	envelopeInterval := float64(envStep) / 64

	// Envelope volume
	envelope := (cntl >> 12) & 0xf
	if envStep != 0 {
		a.chans[3].envTime += SAMPLE_TIME
		if a.chans[3].envTime >= envelopeInterval {
			a.chans[3].envTime -= envelopeInterval

			if util.Bit(cntl, 11) {
				if envelope < 0xf {
					envelope++
				}
			} else {
				if envelope > 0x0 {
					envelope--
				}
			}

			newCntl := (cntl & ^uint32(0xf000)) | (envelope << 12)
			a.buffer[SOUND4CNT_L] = byte(newCntl)
			a.buffer[SOUND4CNT_L+1] = byte(newCntl >> 8)
			a.buffer[SOUND4CNT_L+2] = byte(newCntl >> 16)
			a.buffer[SOUND4CNT_L+3] = byte(newCntl >> 24)
		}
	}

	// Numbers of samples that a single cycle (pseudo-random noise value) takes at output sample rate
	cycleSamples := SND_FREQUENCY / frequency

	carry := byte(a.chans[3].lfsr & 0b1)
	a.chans[3].samples++
	if a.chans[3].samples >= cycleSamples {
		a.chans[3].samples -= cycleSamples
		a.chans[3].lfsr >>= 1

		if carry > 0 {
			if util.Bit(cnth, 3) { // R/W Counter Step/Width
				a.chans[3].lfsr ^= 0x60 // 1: 7bits
			} else {
				a.chans[3].lfsr ^= 0x6000 // 0: 15bits
			}
		}
	}

	if carry != 0 {
		return int8((float64(envelope) / 15) * PSG_MAX) // Out=HIGH
	}
	return int8((float64(envelope) / 15) * PSG_MIN) // Out=LOW
}

func (a *APU) waveReset() {
	wave := uint16(a.Load32(SOUND3CNT_L))
	if util.Bit(wave, 5) { // R/W Wave RAM Dimension
		// 64 samples (at 4 bits each, uses both banks so initial position is always 0)
		wavePosition, waveSamples = 0, 64
		return
	}
	// 32 samples (at 4 bits each, bank selectable through Wave Control register)
	wavePosition, waveSamples = byte((wave>>1)&0x20), 32
}

var (
	sndCurPlay  uint32 = 0
	sndCurWrite uint32 = 0x200
)

// This prevents the cursor from overflowing. Call after some time (like per frame, or per second...)
func SoundBufferWrap() {
	left, right := sndCurPlay/BUFF_SAMPLES, sndCurWrite/BUFF_SAMPLES
	if left == right {
		sndCurPlay &= BUFF_SAMPLES_MSK
		sndCurWrite &= BUFF_SAMPLES_MSK
	}
}

var sndBuffer [BUFF_SAMPLES]int16

func (a *APU) soundMix() {
	for i := 0; i < STREAM_LEN; i += 4 {
		snd := sndBuffer[sndCurPlay&BUFF_SAMPLES_MSK] << 6
		a.stream[i+0], a.stream[i+1] = byte(snd), byte(snd>>8)
		sndCurPlay++
		snd = sndBuffer[sndCurPlay&BUFF_SAMPLES_MSK] << 6
		a.stream[i+2], a.stream[i+3] = byte(snd), byte(snd>>8)
		sndCurPlay++
	}

	// Avoid desync between the Play cursor and the Write cursor
	delta := (int32(sndCurWrite-sndCurPlay) >> 8) - (int32(sndCurWrite-sndCurPlay)>>8)%2
	sndCurPlay = util.AddInt32(sndCurPlay, delta)
}

var (
	FifoALen, FifoBLen byte
	fifoA, fifoB       [0x20]int8
)

func FifoACopy(val uint32) {
	if FifoALen > 28 { // FIFO A full
		FifoALen -= 28
	}

	for i := uint32(0); i < 4; i++ {
		fifoA[FifoALen] = int8(val >> (8 * i))
		FifoALen++
	}
}

func FifoBCopy(val uint32) {
	if FifoBLen > 28 { // FIFO B full
		FifoBLen -= 28
	}

	for i := uint32(0); i < 4; i++ {
		fifoB[FifoBLen] = int8(val >> (8 * i))
		FifoBLen++
	}
}

var (
	fifoASamp, fifoBSamp int8
)

func FifoALoad() {
	if FifoALen == 0 {
		return
	}

	fifoASamp = fifoA[0]
	FifoALen--

	for i := byte(0); i < FifoALen; i++ {
		fifoA[i] = fifoA[i+1]
	}
}

func FifoBLoad() {
	if FifoBLen == 0 {
		return
	}

	fifoBSamp = fifoB[0]
	FifoBLen--

	for i := byte(0); i < FifoBLen; i++ {
		fifoB[i] = fifoB[i+1]
	}
}

var (
	sndCycles = uint32(0)
	psgVolLut = [8]int32{0x000, 0x024, 0x049, 0x06d, 0x092, 0x0b6, 0x0db, 0x100}
	psgRshLut = [4]int32{0xa, 0x9, 0x8, 0x7}
)

func clip(val int32) int16 {
	if val > SAMP_MAX {
		val = SAMP_MAX
	}
	if val < SAMP_MIN {
		val = SAMP_MIN
	}

	return int16(val)
}

func (a *APU) SoundClock(cycles uint32) {
	sndCycles += cycles

	sampPcmL, sampPcmR := int16(0), int16(0)

	cnth := uint16(a.Load32(SOUNDCNT_H)) // snd_pcm_vol
	volADiv, volBDiv := int16((cnth>>2)&0b1)^1, int16((cnth>>3)&0b1)^1
	sampCh4, sampCh5 := (int16(fifoASamp)<<1)>>volADiv, (int16(fifoBSamp)<<1)>>volBDiv

	// Left
	if util.Bit(cnth, 9) {
		sampPcmL = clip(int32(sampPcmL) + int32(sampCh4))
	}
	if util.Bit(cnth, 13) {
		sampPcmL = clip(int32(sampPcmL) + int32(sampCh5))
	}

	// Right
	if util.Bit(cnth, 8) {
		sampPcmR = clip(int32(sampPcmR) + int32(sampCh4))
	}
	if util.Bit(cnth, 12) {
		sampPcmR = clip(int32(sampPcmR) + int32(sampCh5))
	}

	for sndCycles >= SAMP_CYCLES {
		sampCh := [4]int16{int16(a.squareSample(0)), int16(a.squareSample(1)), int16(a.waveSample()), int16(a.noiseSample())}
		sampPsgL, sampPsgR := int32(0), int32(0)

		cntl := uint16(a.Load32(SOUNDCNT_L)) // snd_psg_vol
		for i := 0; i < 4; i++ {
			if util.Bit(cntl, 12+i) {
				sampPsgL = int32(clip(sampPsgL + int32(sampCh[i])))
			}
		}
		for i := 0; i < 4; i++ {
			if util.Bit(cntl, 8+i) {
				sampPsgR = int32(clip(sampPsgR + int32(sampCh[i])))
			}
		}

		sampPsgL *= psgVolLut[(cntl>>4)&7]
		sampPsgR *= psgVolLut[(cntl>>0)&7]

		sampPsgL >>= psgRshLut[(cnth>>0)&3]
		sampPsgR >>= psgRshLut[(cnth>>0)&3]

		sndBuffer[sndCurWrite&BUFF_SAMPLES_MSK] = clip(sampPsgL + int32(sampPcmL))
		sndCurWrite++
		sndBuffer[sndCurWrite&BUFF_SAMPLES_MSK] = clip(sampPsgR + int32(sampPcmR))
		sndCurWrite++

		sndCycles -= SAMP_CYCLES
	}
}

func (a *APU) _resetSoundChan(ch int, enable bool) {
	if enable {
		a.chans[ch] = &SoundChan{}

		switch ch {
		case 2:
			a.waveReset()
		case 3:
			if util.Bit(a.Load32(SOUND4CNT_H), 3) { // R/W Counter Step/Width
				a.chans[3].lfsr = 0x0040 // 7bit
			} else {
				a.chans[3].lfsr = 0x4000 // 15bit
			}
		}

		a.enableSoundChan(ch, true)
	}
}
