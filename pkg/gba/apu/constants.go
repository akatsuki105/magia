package apu

const base = -0x60

// Sound IO
const (
	SOUND1CNT_L, SOUND1CNT_H, SOUND1CNT_X = base + 0x60, base + 0x62, base + 0x64
	SOUND2CNT_L, SOUND2CNT_H              = base + 0x68, base + 0x6c
	SOUND3CNT_L, SOUND3CNT_H, SOUND3CNT_X = base + 0x70, base + 0x72, base + 0x74
	SOUND4CNT_L, SOUND4CNT_H              = base + 0x78, base + 0x7c
	SOUNDCNT_L, SOUNDCNT_H, SOUNDCNT_X    = base + 0x80, base + 0x82, base + 0x84
	SOUNDBIAS                             = base + 0x88
	WAVE_RAM                              = base + 0x90
	FIFO_A, FIFO_B                        = base + 0xa0, base + 0xa4
)
