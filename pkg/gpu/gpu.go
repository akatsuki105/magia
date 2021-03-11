package gpu

import (
	"mettaur/pkg/util"
)

// IO offset
const (
	DISPCNT  = 0x0
	DISPSTAT = 0x4
	VCOUNT   = 0x6
	BG0CNT   = 0x8
	BG1CNT   = 0xa
	BG2CNT   = 0xc
	BG3CNT   = 0xe
	BG0HOFS  = 0x10
	BG0VOFS  = 0x12
	BG1HOFS  = 0x14
	BG1VOFS  = 0x16
	BG2HOFS  = 0x18
	BG2VOFS  = 0x1a
	BG3HOFS  = 0x1c
	BG3VOFS  = 0x1e
	BG2PA    = 0x20
	BG2PB    = 0x22
	BG2PC    = 0x24
	BG2PD    = 0x26
	BG2X     = 0x28
	BG2Y     = 0x2c
	BG3PA    = 0x30
	BG3PB    = 0x32
	BG3PC    = 0x34
	BG3PD    = 0x36
	BG3X     = 0x38
	BG3Y     = 0x3c
	WIN0H    = 0x40
	WIN1H    = 0x42
	WIN0V    = 0x44
	WIN1V    = 0x46
	WININ    = 0x48
	WINOUT   = 0x4a
	MOSAIC   = 0x4c
	BLDCNT   = 0x50
	BLDALPHA = 0x52
	BLDY     = 0x54
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
	return (addr >= 0x0400_0000) && (addr <= 0x0400_0000+0x60)
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
