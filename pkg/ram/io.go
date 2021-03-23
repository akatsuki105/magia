package ram

const base = 0x0400_0000

// LCD IO
const (
	DISPCNT  = base
	DISPSTAT = base + 0x4
	VCOUNT   = base + 0x6
	BG0CNT   = base + 0x8
	BG1CNT   = base + 0xa
	BG2CNT   = base + 0xc
	BG3CNT   = base + 0xe
	BG0HOFS  = base + 0x10
	BG0VOFS  = base + 0x12
	BG1HOFS  = base + 0x14
	BG1VOFS  = base + 0x16
	BG2HOFS  = base + 0x18
	BG2VOFS  = base + 0x1a
	BG3HOFS  = base + 0x1c
	BG3VOFS  = base + 0x1e
	BG2PA    = base + 0x20
	BG2PB    = base + 0x22
	BG2PC    = base + 0x24
	BG2PD    = base + 0x26
	BG2X     = base + 0x28
	BG2Y     = base + 0x2c
	BG3PA    = base + 0x30
	BG3PB    = base + 0x32
	BG3PC    = base + 0x34
	BG3PD    = base + 0x36
	BG3X     = base + 0x38
	BG3Y     = base + 0x3c
	WIN0H    = base + 0x40
	WIN1H    = base + 0x42
	WIN0V    = base + 0x44
	WIN1V    = base + 0x46
	WININ    = base + 0x48
	WINOUT   = base + 0x4a
	MOSAIC   = base + 0x4c
	BLDCNT   = base + 0x50
	BLDALPHA = base + 0x52
	BLDY     = base + 0x54
)

// DMA Transfer
const (
	DMA0SAD = base + 0xb0
	DMA0DAD = base + 0xb4
	DMA0CNT = base + 0xb8
	DMA1SAD = base + 0xbc
	DMA1DAD = base + 0xc0
	DMA1CNT = base + 0xc4
	DMA2SAD = base + 0xc8
	DMA2DAD = base + 0xcc
	DMA2CNT = base + 0xd0
	DMA3SAD = base + 0xd4
	DMA3DAD = base + 0xd8
	DMA3CNT = base + 0xdc
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
	IE      = base + 0x200
	IF      = base + 0x202
	WAITCNT = base + 0x204
	IME     = base + 0x208
	HALTCNT = base + 0x301
)
