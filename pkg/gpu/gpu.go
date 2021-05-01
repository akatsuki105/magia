package gpu

import (
	"mettaur/pkg/util"
)

// IO offset
const (
	DISPCNT                        = 0x0
	DISPSTAT                       = 0x4
	VCOUNT                         = 0x6
	BG0CNT, BG1CNT, BG2CNT, BG3CNT = 0x8, 0xa, 0xc, 0xe
	BG0HOFS, BG0VOFS               = 0x10, 0x12
	BG1HOFS, BG1VOFS               = 0x14, 0x16
	BG2HOFS, BG2VOFS               = 0x18, 0x1a
	BG3HOFS, BG3VOFS               = 0x1c, 0x1e
	BG2PA, BG2PB, BG2PC, BG2PD     = 0x20, 0x22, 0x24, 0x26
	BG2X, BG2Y                     = 0x28, 0x2c
	BG3PA, BG3PB, BG3PC, BG3PD     = 0x30, 0x32, 0x34, 0x36
	BG3X, BG3Y                     = 0x38, 0x3c
	WIN0H                          = 0x40
	WIN1H                          = 0x42
	WIN0V                          = 0x44
	WIN1V                          = 0x46
	WININ                          = 0x48
	WINOUT                         = 0x4a
	MOSAIC                         = 0x4c
	BLDCNT                         = 0x50
	BLDALPHA                       = 0x52
	BLDY                           = 0x54
)

const (
	_       = iota
	kb uint = 1 << (10 * iota)
	mb
	gb
)

// GPU graphic processor unit
type GPU struct {
	RAM
	IO [0x60]byte
}

func New() *GPU {
	return &GPU{}
}

// RAM represents VRAM
type RAM struct {
	Palette [kb]byte
	VRAM    [96 * kb]byte
	OAM     [kb]byte
}

// IsIO returns true if addr is for GPU IO register.
func IsIO(addr uint32) bool {
	return (addr >= 0x0400_0000) && (addr < 0x0400_0000+0x60)
}

func (g *GPU) SetIO(addr uint32, b byte) {
	io := addr - 0x0400_0000
	switch io {
	case BG0CNT + 1, BG1CNT + 1:
		b &= 0xdf
	case BG0HOFS + 1, BG0VOFS + 1:
		b &= 0x01
	}

	g.IO[io] = b
}

// VBlank returns true if in VBlank
func (g *GPU) VBlank() bool {
	return util.Bit(uint16(g.IO[DISPSTAT]), 0)
}

// IncrementVCount increments VCOUNT
func (g *GPU) IncrementVCount() byte {
	g.IO[VCOUNT]++
	if g.IO[VCOUNT] == 228 {
		g.IO[VCOUNT] = 0
	}
	g.SetVCounter(g.IO[VCOUNT] == g.IO[DISPSTAT+1])
	return g.IO[VCOUNT]
}

func (g *GPU) SetVBlank(b bool) {
	if b {
		g.IO[DISPSTAT] = g.IO[DISPSTAT] | 0b0000_0001
		return
	}
	g.IO[DISPSTAT] = g.IO[DISPSTAT] & 0b1111_1110
}
func (g *GPU) SetHBlank(b bool) {
	if b {
		g.IO[DISPSTAT] = g.IO[DISPSTAT] | 0b0000_0010
		return
	}
	g.IO[DISPSTAT] = g.IO[DISPSTAT] & 0b1111_1101
}
func (g *GPU) SetVCounter(b bool) {
	if b {
		g.IO[DISPSTAT] = g.IO[DISPSTAT] | 0b0000_0100
		return
	}
	g.IO[DISPSTAT] = g.IO[DISPSTAT] & 0b1111_1011
}
