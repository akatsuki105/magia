package gba

import (
	"mettaur/pkg/gpu"
	"mettaur/pkg/ram"
	"mettaur/pkg/timer"
	"mettaur/pkg/util"
)

var (
	mask [4]uint32 = [4]uint32{
		0b1111_1111_1111_1111_1111_1111_0000_0000,
		0b1111_1111_1111_1111_0000_0000_1111_1111,
		0b1111_1111_0000_0000_1111_1111_1111_1111,
		0b0000_0000_1111_1111_1111_1111_1111_1111,
	}
)

func (g *GBA) _getRAM(addr uint32) uint32 {
	switch {
	case gpu.IsIO(addr):
		return util.LE32(g.GPU.IO[(addr - 0x0400_0000):])
	case ram.Palette(addr):
		offset := ram.PaletteOffset(addr)
		return util.LE32(g.GPU.Palette[offset:])
	case ram.VRAM(addr):
		offset := ram.VRAMOffset(addr)
		return util.LE32(g.GPU.VRAM[offset:])
	case ram.OAM(addr):
		offset := ram.OAMOffset(addr)
		return util.LE32(g.GPU.OAM[offset:])
	default:
		return g.RAM.Get(addr)
	}
}
func (g *GBA) getRAM32(addr uint32, s bool) uint32 {
	g.timer(g.waitBus(addr, 32, s))
	return g._getRAM(addr)
}

func (g *GBA) getRAM16(addr uint32, s bool) uint16 {
	g.timer(g.waitBus(addr, 16, s))
	return uint16(g._getRAM(addr))
}

func (g *GBA) getRAM8(addr uint32, s bool) byte {
	g.timer(g.waitBus(addr, 8, s))
	return byte(g._getRAM(addr))
}

func (g *GBA) setRAM32(addr, value uint32, s bool) {
	g.timer(g.waitBus(addr, 32, s))
	b0, b1, b2, b3 := value&0xff, (value>>8)&0xff, (value>>16)&0xff, (value>>24)&0xff
	g._setRAM8(addr, byte(b0))
	g._setRAM8(addr+1, byte(b1))
	g._setRAM8(addr+2, byte(b2))
	g._setRAM8(addr+3, byte(b3))
}

func (g *GBA) setRAM16(addr uint32, value uint16, s bool) {
	g.timer(g.waitBus(addr, 16, s))
	g._setRAM16(addr, value)
}

func (g *GBA) _setRAM16(addr uint32, value uint16) {
	b0, b1 := value&0xff, (value>>8)&0xff
	g._setRAM8(addr, byte(b0))
	g._setRAM8(addr+1, byte(b1))
}

func (g *GBA) setRAM8(addr uint32, b byte, s bool) {
	g.timer(g.waitBus(addr, 8, s))
	g._setRAM8(addr, b)
}

func (g *GBA) _setRAM8(addr uint32, b byte) {
	switch {
	case gpu.IsIO(addr):
		g.GPU.IO[addr-0x0400_0000] = b
	case timer.IsIO(addr):
		g.timers.SetIO(addr-0x0400_0100, b)
	case addr == ram.DISPCNT || addr == ram.DISPCNT+1 || addr == ram.IME || addr == ram.IME+1 || addr == ram.IME+2 || addr == ram.IME+3:
		g.RAM.Set8(addr, b)
		g.checkIRQ()
	case addr == ram.IF || addr == ram.IF+1:
		value := byte(g._getRAM(addr))
		g.RAM.Set8(addr, value & ^b)
	case addr == ram.HALTCNT:
		g.halt = true
	case ram.Palette(addr):
		g.GPU.Palette[ram.PaletteOffset(addr)] = b
	case ram.VRAM(addr):
		g.GPU.VRAM[ram.VRAMOffset(addr)] = b
	case ram.OAM(addr):
		g.GPU.OAM[ram.OAMOffset(addr)] = b
	default:
		g.RAM.Set8(addr, b)
	}
}
