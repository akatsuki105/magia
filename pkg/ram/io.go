package ram

type IORegister uint32

const base = 0x0400_0000

const (
	DISPCNT  IORegister = base
	DISPSTAT            = base + 0x4
	VCOUNT              = base + 0x6
	BG0CNT              = base + 0x8
	BG1CNT              = base + 0xa
	BG2CNT              = base + 0xc
	BG3CNT              = base + 0xe
	BG0HOFS             = base + 0x10
	BG0VOFS             = base + 0x12
	BG1HOFS             = base + 0x14
	BG1VOFS             = base + 0x16
	BG2HOFS             = base + 0x18
	BG2VOFS             = base + 0x1a
	BG3HOFS             = base + 0x1c
	BG3VOFS             = base + 0x1e
	WAITCNT             = base + 0x204
)
