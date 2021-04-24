package ram

const base = 0x0400_0000

// LCD IO
const (
	DISPCNT                        = base
	DISPSTAT                       = base + 0x4
	VCOUNT                         = base + 0x6
	BG0CNT, BG1CNT, BG2CNT, BG3CNT = base + 0x8, base + 0xa, base + 0xc, base + 0xe
	BG0HOFS, BG0VOFS               = base + 0x10, base + 0x12
	BG1HOFS, BG1VOFS               = base + 0x14, base + 0x16
	BG2HOFS, BG2VOFS               = base + 0x18, base + 0x1a
	BG3HOFS, BG3VOFS               = base + 0x1c, base + 0x1e
	BG2PA, BG2PB, BG2PC, BG2PD     = base + 0x20, base + 0x22, base + 0x24, base + 0x26
	BG2X, BG2Y                     = base + 0x28, base + 0x2c
	BG3PA, BG3PB, BG3PC, BG3PD     = base + 0x30, base + 0x32, base + 0x34, base + 0x36
	BG3X, BG3Y                     = base + 0x38, base + 0x3c
	WIN0H                          = base + 0x40
	WIN1H                          = base + 0x42
	WIN0V                          = base + 0x44
	WIN1V                          = base + 0x46
	WININ                          = base + 0x48
	WINOUT                         = base + 0x4a
	MOSAIC                         = base + 0x4c
	BLDCNT                         = base + 0x50
	BLDALPHA                       = base + 0x52
	BLDY                           = base + 0x54
)

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

// DMA Transfer
const (
	DMA0SAD, DMA0DAD, DMA0CNT = base + 0xb0, base + 0xb4, base + 0xb8
	DMA1SAD, DMA1DAD, DMA1CNT = base + 0xbc, base + 0xc0, base + 0xc4
	DMA2SAD, DMA2DAD, DMA2CNT = base + 0xc8, base + 0xcc, base + 0xd0
	DMA3SAD, DMA3DAD, DMA3CNT = base + 0xd4, base + 0xd8, base + 0xdc
)

// Timer
const (
	TM0CNT = base + 0x100
	TM1CNT = base + 0x104
	TM2CNT = base + 0x108
	TM3CNT = base + 0x10c
)

// Keypad Input
const (
	KEYINPUT = base + 0x130
	KEYCNT   = base + 0x132
)

// System IO
const (
	IE, IF  = base + 0x200, base + 0x202
	WAITCNT = base + 0x204
	IME     = base + 0x208
	HALTCNT = base + 0x301
)
