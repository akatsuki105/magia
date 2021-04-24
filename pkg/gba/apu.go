package gba

import (
	"math"
	"mettaur/pkg/ram"
	"mettaur/pkg/util"
	"time"

	"github.com/hajimehoshi/oto"
)

const (
	CPU_FREQ_HZ              = 16777216
	SND_FREQUENCY            = 32768
	SND_SAMPLES              = 512
	SAMP_CYCLES              = (CPU_FREQ_HZ / SND_FREQUENCY)
	BUFF_SAMPLES             = ((SND_SAMPLES) * 16 * 2)
	BUFF_SAMPLES_MSK         = ((BUFF_SAMPLES) - 1)
	SAMPLE_TIME      float64 = 1.0 / SND_FREQUENCY
	SAMPLE_RATE              = SND_FREQUENCY
	STREAM_LEN               = 2940 // 2 * 2 * sampleRate * (1/60)
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
var waveRAM [0x20]byte
var resetSoundChanMap = map[uint32]int{0x04000065: 0, 0x0400006d: 1, 0x04000075: 2, 0x0400007d: 3}

type APU struct {
	enable  bool
	context *oto.Context
	player  *oto.Player
	chans   [4]*SoundChan
}

type SoundChan struct {
	phase                                   bool
	lfsr                                    uint16
	samples, lengthTime, sweepTime, envTime float64
}

func isWaveRAM(addr uint32) bool {
	return addr >= 0x04000090 && addr <= 0x0400009f
}

func isResetSoundChan(addr uint32) bool {
	_, ok := resetSoundChanMap[addr]
	return ok
}

func (g *GBA) resetSoundChan(addr uint32, b byte) {
	g._resetSoundChan(resetSoundChanMap[addr], util.Bit(b, 7))
}

func newAPU() *APU {
	stream = make([]byte, STREAM_LEN)

	context, err := oto.NewContext(SAMPLE_RATE, 2, 2, STREAM_LEN)
	if err != nil {
		panic(err)
	}

	player := context.NewPlayer()
	return &APU{
		context: context,
		player:  player,
		chans:   [4]*SoundChan{&SoundChan{}, &SoundChan{}, &SoundChan{}, &SoundChan{}},
	}
}

func (a *APU) exit() {
	a.context.Close()
}

func (a *APU) play() {
	a.enable = true
	go func() {
		for range time.Tick(time.Second / 60) {
			soundMix()
			if a.enable {
				a.player.Write(stream)
			}
		}
	}()
}

var dutyLookUp = [4]float64{0.125, 0.25, 0.5, 0.75}
var dutyLookUpi = [4]float64{0.875, 0.75, 0.5, 0.25}

func (g *GBA) squareSample(ch int) int8 {
	if !g.isSoundChanEnable(ch) {
		return 0
	}

	toneAddr := uint32(ram.SOUND1CNT_H)
	if ch == 1 {
		toneAddr = ram.SOUND2CNT_L
	}

	freqHz := g._getRAM(ram.SOUND1CNT_X) & 0b0111_1111_1111
	if ch == 1 {
		freqHz = g._getRAM(ram.SOUND2CNT_H) & 0b0111_1111_1111
	}
	frequency := 131072 / float64(2048-freqHz)

	// Full length of the generated wave (if enabled) in seconds
	soundLen := g._getRAM(toneAddr) & 0b0011_1111
	length := float64(64-soundLen) / 256

	// Envelope volume change interval in seconds
	envStep := g._getRAM(toneAddr) >> 8 & 0b111
	envelopeInterval := float64(envStep) / 64

	cycleSamples := SND_FREQUENCY / frequency // Numbers of samples that a single cycle (wave phase change 1 -> 0) takes at output sample rate

	// Length reached check (if so, just disable the channel and return silence)
	if (ch == 0 && util.Bit(g._getRAM(ram.SOUND1CNT_X), 14)) || (ch == 1 && util.Bit(g._getRAM(ram.SOUND2CNT_H), 14)) {
		g.apu.chans[ch].lengthTime += SAMPLE_TIME
		if g.apu.chans[ch].lengthTime >= length {
			g.enableSoundChan(ch, false)
			return 0
		}
	}

	// Frequency sweep (Square 1 channel only)
	if ch == 0 {
		sweepTime := (g._getRAM(ram.SOUND1CNT_L) >> 4) & 0b111
		sweepInterval := 0.0078 * float64(sweepTime+1) // Frquency sweep change interval in seconds

		g.apu.chans[0].sweepTime += SAMPLE_TIME
		if g.apu.chans[0].sweepTime >= sweepInterval {
			g.apu.chans[0].sweepTime -= sweepInterval

			// A Sweep Shift of 0 means that Sweep is disabled
			sweepShift := byte(g._getRAM(ram.SOUND1CNT_L) & 7)

			if sweepShift != 0 {
				disp := freqHz >> sweepShift

				if util.Bit(g._getRAM(ram.SOUND1CNT_L), 3) {
					freqHz -= disp
				} else {
					freqHz += disp
				}

				if freqHz < 0x7ff {
					// update frequency
					ctrl := uint16(g._getRAM(ram.SOUND1CNT_X))
					ctrl = (ctrl & ^uint16(0x7ff)) | uint16(freqHz)
					g._setRAM(ram.SOUND1CNT_X, uint32(ctrl), 2)
				} else {
					g.enableSoundChan(0, false)
				}
			}
		}
	}

	// Envelope volume
	envelope := uint16((g._getRAM(toneAddr) >> 12) & 0xf)
	if envStep > 0 {
		g.apu.chans[ch].envTime += SAMPLE_TIME

		if g.apu.chans[ch].envTime >= envelopeInterval {
			g.apu.chans[ch].envTime -= envelopeInterval

			tone := uint16(g._getRAM(toneAddr))
			if util.Bit(tone, 11) {
				if envelope < 0xf {
					envelope++
				}
			} else {
				if envelope > 0 {
					envelope--
				}
			}

			tone = (tone & ^uint16(0xf000)) | (envelope << 12)
			g._setRAM(toneAddr, uint32(tone), 2)
		}
	}

	// Phase change (when the wave goes from Low to High or High to Low, the Square Wave pattern)
	duty := (g._getRAM(toneAddr) >> 6) & 0b11
	g.apu.chans[ch].samples++
	if g.apu.chans[ch].phase {
		// 1 -> 0
		phaseChange := cycleSamples * dutyLookUp[duty]
		if g.apu.chans[ch].samples > phaseChange {
			g.apu.chans[ch].samples -= phaseChange
			g.apu.chans[ch].phase = false
		}
	} else {
		// 0 -> 1
		phaseChange := cycleSamples * dutyLookUpi[duty]
		if g.apu.chans[ch].samples > phaseChange {
			g.apu.chans[ch].samples -= phaseChange
			g.apu.chans[ch].phase = true
		}
	}

	if g.apu.chans[ch].phase {
		return int8(float64(envelope) * PSG_MAX / 15)
	}
	return int8(float64(envelope) * PSG_MIN / 15)
}

func (g *GBA) enableSoundChan(ch int, enable bool) {
	cntx := g._getRAM(ram.SOUNDCNT_X)
	cntx = util.SetBit32(cntx, ch, enable)
	g.RAM.IO[ram.IOOffset(ram.SOUNDCNT_X)] = byte(cntx)
}

func (g *GBA) isSoundMasterEnable() bool {
	cntx := byte(g._getRAM(ram.SOUNDCNT_X))
	return util.Bit(cntx, 7)
}

func (g *GBA) isSoundChanEnable(ch int) bool {
	cntx := byte(g._getRAM(ram.SOUNDCNT_X))
	return util.Bit(cntx, ch)
}

// chan3
func (g *GBA) waveSample() int8 {
	wave := uint16(g._getRAM(ram.SOUND3CNT_L))
	if !(g.isSoundChanEnable(2) && util.Bit(wave, 7)) {
		return 0
	}

	// Actual frequency in Hertz
	rate := g._getRAM(ram.SOUND3CNT_X) & 2047
	frequency := 2097152 / (2048 - float64(rate))

	cnth := uint16(g._getRAM(ram.SOUND3CNT_H)) // volume

	// Full length of the generated wave (if enabled) in seconds
	soundLen := cnth & 0xff
	length := (256 - float64(soundLen)) / 256.

	// Numbers of samples that a single "cycle" (all entries on Wave RAM) takes at output sample rate
	cycleSamples := SND_FREQUENCY / frequency

	// Length reached check (if so, just disable the channel and return silence)
	if util.Bit(uint16(g._getRAM(ram.SOUND3CNT_X)), 14) {
		g.apu.chans[2].lengthTime += SAMPLE_TIME
		if g.apu.chans[2].lengthTime >= length {
			g.enableSoundChan(2, false)
			return 0
		}
	}

	g.apu.chans[2].samples++
	if g.apu.chans[2].samples >= cycleSamples {
		g.apu.chans[2].samples -= cycleSamples

		waveSamples--
		if waveSamples != 0 {
			wavePosition = (wavePosition + 1) & 0b0011_1111
		} else {
			g.waveReset()
		}
	}

	wavedata := waveRAM[(uint32(wavePosition)>>1)&0x1f]
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
func (g *GBA) noiseSample() int8 {
	if !g.isSoundChanEnable(3) {
		return 0
	}

	cnth := g._getRAM(ram.SOUND4CNT_H) // ctrl

	// Actual frequency in Hertz (524288 / r / 2^(s+1))
	r, s := float64(cnth&0x7), float64((cnth>>4)&0xf)
	if r == 0 {
		r = 0.5
	}
	frequency := (524288 / r) / math.Pow(2, s+1)

	cntl := g._getRAM(ram.SOUND4CNT_L) // env
	// Full length of the generated wave (if enabled) in seconds
	soundLen := cntl & 0x3f
	length := (64 - float64(soundLen)) / 256

	// Length reached check (if so, just disable the channel and return silence)
	if util.Bit(cnth, 14) {
		g.apu.chans[3].lengthTime += SAMPLE_TIME
		if g.apu.chans[3].lengthTime >= length {
			g.enableSoundChan(3, false)
			return 0
		}
	}

	// Envelope volume change interval in seconds
	envStep := (cntl >> 8) & 0x7
	envelopeInterval := float64(envStep) / 64

	// Envelope volume
	envelope := (cntl >> 12) & 0xf
	if envStep != 0 {
		g.apu.chans[3].envTime += SAMPLE_TIME
		if g.apu.chans[3].envTime >= envelopeInterval {
			g.apu.chans[3].envTime -= envelopeInterval

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
			g._setRAM(ram.SOUND4CNT_L, newCntl, 4)
		}
	}

	// Numbers of samples that a single cycle (pseudo-random noise value) takes at output sample rate
	cycleSamples := SND_FREQUENCY / frequency

	carry := byte(g.apu.chans[3].lfsr & 0b1)
	g.apu.chans[3].samples++
	if g.apu.chans[3].samples >= cycleSamples {
		g.apu.chans[3].samples -= cycleSamples
		g.apu.chans[3].lfsr >>= 1

		if carry > 0 {
			if util.Bit(cnth, 3) { // R/W Counter Step/Width
				g.apu.chans[3].lfsr ^= 0x60 // 1: 7bits
			} else {
				g.apu.chans[3].lfsr ^= 0x6000 // 0: 15bits
			}
		}
	}

	if carry != 0 {
		return int8((float64(envelope) / 15) * PSG_MAX) // Out=HIGH
	}
	return int8((float64(envelope) / 15) * PSG_MIN) // Out=LOW
}

func (g *GBA) waveReset() {
	wave := uint16(g._getRAM(ram.SOUND3CNT_L))
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
func soundBufferWrap() {
	left, right := sndCurPlay/BUFF_SAMPLES, sndCurWrite/BUFF_SAMPLES
	if left == right {
		sndCurPlay &= BUFF_SAMPLES_MSK
		sndCurWrite &= BUFF_SAMPLES_MSK
	}
}

var sndBuffer [BUFF_SAMPLES]int16
var stream []byte

func soundMix() {
	for i := 0; i < STREAM_LEN; i += 4 {
		snd := sndBuffer[sndCurPlay&BUFF_SAMPLES_MSK] << 6
		stream[i+0], stream[i+1] = byte(snd), byte(snd>>8)
		sndCurPlay++
		snd = sndBuffer[sndCurPlay&BUFF_SAMPLES_MSK] << 6
		stream[i+2], stream[i+3] = byte(snd), byte(snd>>8)
		sndCurPlay++
	}

	// Avoid desync between the Play cursor and the Write cursor
	delta := (int32(sndCurWrite-sndCurPlay) >> 8) - (int32(sndCurWrite-sndCurPlay)>>8)%2
	if delta >= 0 {
		sndCurPlay += uint32(delta)
	} else {
		sndCurPlay -= uint32(-delta)
	}
}

var (
	fifoALen, fifoBLen byte
	fifoA, fifoB       [0x20]int8
)

func (g *GBA) fifoACopy(val uint32) {
	if fifoALen > 28 { // FIFO A full
		fifoALen -= 28
	}

	for i := uint32(0); i < 4; i++ {
		fifoA[fifoALen] = int8(val >> (8 * i))
		fifoALen++
	}
}
func (g *GBA) fifoBCopy(val uint32) {
	if fifoBLen > 28 { // FIFO B full
		fifoBLen -= 28
	}

	for i := uint32(0); i < 4; i++ {
		fifoB[fifoBLen] = int8(val >> (8 * i))
		fifoBLen++
	}
}

var (
	fifoASamp, fifoBSamp int8
)

func (g *GBA) fifoALoad() {
	if fifoALen == 0 {
		return
	}

	fifoASamp = fifoA[0]
	fifoALen--

	for i := byte(0); i < fifoALen; i++ {
		fifoA[i] = fifoA[i+1]
	}
}

func (g *GBA) fifoBLoad() {
	if fifoBLen == 0 {
		return
	}

	fifoBSamp = fifoB[0]
	fifoBLen--

	for i := byte(0); i < fifoBLen; i++ {
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

func (g *GBA) soundClock(cycles uint32) {
	defer g.PanicHandler(true)
	sndCycles += cycles

	sampPcmL, sampPcmR := int16(0), int16(0)

	cnth := uint16(g._getRAM(ram.SOUNDCNT_H)) // snd_pcm_vol
	volADiv, volBDiv := int16((cnth>>2)&0b1), int16((cnth>>3)&0b1)
	sampCh4, sampCh5 := fifoASamp>>volADiv, fifoBSamp>>volBDiv
	// sampCh4, sampCh5 = 0, 0

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
		sampCh := [4]int16{int16(g.squareSample(0)), int16(g.squareSample(1)), int16(g.waveSample()), int16(g.noiseSample())}
		sampPsgL, sampPsgR := int32(0), int32(0)

		cntl := uint16(g._getRAM(ram.SOUNDCNT_L)) // snd_psg_vol
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

func (g *GBA) _resetSoundChan(ch int, enable bool) {
	if enable {
		g.apu.chans[ch] = &SoundChan{}

		switch ch {
		case 2:
			g.waveReset()
		case 3:
			if util.Bit(g._getRAM(ram.SOUND4CNT_H), 3) { // R/W Counter Step/Width
				g.apu.chans[3].lfsr = 0x0040 // 7bit
			} else {
				g.apu.chans[3].lfsr = 0x4000 // 15bit
			}
		}

		g.enableSoundChan(ch, true)
	}
}
