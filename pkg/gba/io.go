package gba

import (
	"fmt"
	"mettaur/pkg/gpu"
	"mettaur/pkg/ram"
	"mettaur/pkg/util"
	"strings"
)

func (g *GBA) _getRAM(addr uint32) uint32 {
	switch {
	case gpu.IsIO(addr):
		return util.LE32(g.GPU.IO[(addr - 0x0400_0000):])
	case g.in(addr, ram.WAVE_RAM, ram.WAVE_RAM+0xf):
		bank := (g._getRAM(ram.SOUND3CNT_L) >> 2) & 0x10
		idx := (bank ^ 0x10) | (addr & 0xf)
		return util.LE32(waveRAM[idx:])
	case isDMA0IO(addr):
		return g.dma[0].get(addr - 0x0400_00b0)
	case isDMA1IO(addr):
		return g.dma[1].get(addr - 0x0400_00bc)
	case isDMA2IO(addr):
		return g.dma[2].get(addr - 0x0400_00c8)
	case isDMA3IO(addr):
		return g.dma[3].get(addr - 0x0400_00d4)
	case IsTimerIO(addr):
		return g.timers.GetIO(addr - 0x0400_0100)
	case addr == ram.KEYINPUT || addr == ram.KEYINPUT+1:
		return util.LE32(g.joypad.Input[addr-ram.KEYINPUT:])
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
	g._setRAM(addr, value, 4)
}

func (g *GBA) setRAM16(addr uint32, value uint16, s bool) {
	g.timer(g.waitBus(addr, 16, s))
	g._setRAM(addr, uint32(value), 2)
}

func (g *GBA) setRAM8(addr uint32, b byte, s bool) {
	g.timer(g.waitBus(addr, 8, s))
	g._setRAM(addr, uint32(b), 1)
}

func (g *GBA) _setRAM(addr uint32, val uint32, width int) {
	defer func() {
		if err := recover(); err != nil {
			s := fmt.Sprintln(err)
			if strings.Contains(s, "runtime error: index out of range") {
				msg := fmt.Sprintf("access to 0x%08x(%v)", addr, err)
				panic(msg)
			}
			panic(err)
		}
	}()

	switch {
	case gpu.IsIO(addr):
		for i := uint32(0); i < uint32(width); i++ {
			g.GPU.IO[addr-0x0400_0000+i] = byte(val >> (8 * i))
		}
	case g.in(addr, ram.SOUND1CNT_L, ram.SOUNDCNT_L+1): // sound io
		if util.Bit(byte(g._getRAM(ram.SOUNDCNT_X)), 7) {
			for i := uint32(0); i < uint32(width); i++ {
				g.RAM.Set8(addr+i, byte(val>>(8*i)))
				if isResetSoundChan(addr + i) {
					g.resetSoundChan(addr+i, byte(val>>(8*i)))
				}
			}
		}
	case addr == ram.SOUNDCNT_H:
		for i := uint32(0); i < uint32(width); i++ {
			g.RAM.Set8(addr+i, byte(val>>(8*i)))
		}
		if util.Bit(val, 11) {
			fifoA = [32]int8{}
			fifoALen = 0
		}
		if util.Bit(val, 15) {
			fifoB = [32]int8{}
			fifoBLen = 0
		}
	case addr == ram.SOUNDCNT_X:
		old := byte(g._getRAM(addr))
		old = (old & 0xf) | (byte(val) & 0xf0)
		g.RAM.Set8(addr, old)
		if !util.Bit(byte(val), 7) {
			for i := uint32(0x4000060); i <= 0x4000081; i++ {
				g.RAM.IO[ram.IOOffset(i)] = 0
			}
		}
	case g.in(addr, ram.WAVE_RAM, ram.WAVE_RAM+0xf): // wave ram
		for i := uint32(0); i < uint32(width); i++ {
			bank := (g._getRAM(ram.SOUND3CNT_L) >> 2) & 0x10
			idx := (bank ^ 0x10) | (addr & 0xf)
			waveRAM[idx+i] = byte(val >> (8 * i))
		}
	case isDMA0IO(addr):
		for i := uint32(0); i < uint32(width); i++ {
			if g.dma[0].set(addr-0x0400_00b0+i, byte(val>>(8*i))) {
				g.dmaTransfer(dmaImmediate)
			}
		}
	case isDMA1IO(addr):
		for i := uint32(0); i < uint32(width); i++ {
			if g.dma[1].set(addr-0x0400_00bc+i, byte(val>>(8*i))) {
				g.dmaTransfer(dmaImmediate)
			}
		}
	case isDMA2IO(addr):
		for i := uint32(0); i < uint32(width); i++ {
			if g.dma[2].set(addr-0x0400_00c8+i, byte(val>>(8*i))) {
				g.dmaTransfer(dmaImmediate)
			}
		}
	case isDMA3IO(addr):
		for i := uint32(0); i < uint32(width); i++ {
			if g.dma[3].set(addr-0x0400_00d4+i, byte(val>>(8*i))) {
				g.dmaTransfer(dmaImmediate)
			}
		}
	case IsTimerIO(addr):
		for i := uint32(0); i < uint32(width); i++ {
			g.timers.SetIO(addr-0x0400_0100+i, byte(val>>(8*i)))
		}
	case addr == ram.KEYCNT:
		for i := uint32(0); i < uint32(width); i++ {
			g.joypad.Input[2+i] = byte(val >> (8 * i))
		}
	case addr == ram.IE:
		for i := uint32(0); i < uint32(width); i++ {
			g.RAM.Set8(addr+i, byte(val>>(8*i)))
		}
		g.checkIRQ()
	case addr == ram.IF:
		for i := uint32(0); i < uint32(width); i++ {
			value := byte(g._getRAM(addr + i))
			g.RAM.Set8(addr+i, value & ^byte(val>>(8*i)))
		}
	case addr == ram.IME:
		g.RAM.Set8(addr, byte(val)&0b1)
		g.checkIRQ()
	case addr == ram.HALTCNT:
		g.halt = true
	case ram.Palette(addr):
		for i := uint32(0); i < uint32(width); i++ {
			g.GPU.Palette[ram.PaletteOffset(addr+i)] = byte(val >> (8 * i))
		}
	case ram.VRAM(addr):
		for i := uint32(0); i < uint32(width); i++ {
			g.GPU.VRAM[ram.VRAMOffset(addr+i)] = byte(val >> (8 * i))
		}
	case ram.OAM(addr):
		for i := uint32(0); i < uint32(width); i++ {
			g.GPU.OAM[ram.OAMOffset(addr+i)] = byte(val >> (8 * i))
		}
	default:
		for i := uint32(0); i < uint32(width); i++ {
			g.RAM.Set8(addr+i, byte(val>>(8*i)))
		}
		if ram.SRAM(addr) {
			g.DoSav = true
		}
	}
}
