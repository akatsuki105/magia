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
	case isWaveRAM(addr):
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
	g._setRAM32(addr, value)
}

func (g *GBA) _setRAM32(addr, value uint32) {
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
		g.GPU.IO[addr-0x0400_0000] = b
	case isSoundIO(addr):
		if util.Bit(byte(g._getRAM(ram.SOUNDCNT_X)), 7) {
			g.RAM.Set8(addr, b)
			if isResetSoundChan(addr) {
				g.resetSoundChan(addr, b)
			}
		}
	case addr == ram.SOUNDCNT_H+1:
		if util.Bit(b, 3) {
			fifoA = [32]int8{}
			fifoALen = 0
		}
		g.RAM.Set8(addr, b)
	case addr == ram.SOUNDCNT_H+3:
		if util.Bit(b, 7) {
			fifoB = [32]int8{}
			fifoBLen = 0
		}
		g.RAM.Set8(addr, b)
	case addr == ram.SOUNDCNT_X:
		old := byte(g._getRAM(addr))
		old = (old & 0xf) | (b & 0xf0)
		g.RAM.Set8(addr, old)
		if !util.Bit(b, 7) {
			for i := uint32(0x4000060); i <= 0x4000081; i++ {
				g.RAM.IO[ram.IOOffset(i)] = 0
			}
		}
	case isWaveRAM(addr):
		bank := (g._getRAM(ram.SOUND3CNT_L) >> 2) & 0x10
		idx := (bank ^ 0x10) | (addr & 0xf)
		waveRAM[idx] = b
	case isDMA0IO(addr):
		if g.dma[0].set(addr-0x0400_00b0, b) {
			g.dmaTransfer(dmaImmediate)
		}
	case isDMA1IO(addr):
		if g.dma[1].set(addr-0x0400_00bc, b) {
			g.dmaTransfer(dmaImmediate)
		}
	case isDMA2IO(addr):
		if g.dma[2].set(addr-0x0400_00c8, b) {
			g.dmaTransfer(dmaImmediate)
		}
	case isDMA3IO(addr):
		if g.dma[3].set(addr-0x0400_00d4, b) {
			g.dmaTransfer(dmaImmediate)
		}
	case IsTimerIO(addr):
		g.timers.SetIO(addr-0x0400_0100, b)
	case addr == ram.KEYINPUT || addr == ram.KEYINPUT+1 || addr == ram.KEYCNT || addr == ram.KEYCNT+1:
		g.joypad.Input[addr-ram.KEYINPUT] = b
	case addr == ram.IE || addr == ram.IE+1:
		g.RAM.Set8(addr, b)
		g.checkIRQ()
	case addr == ram.IME || addr == ram.IME+1 || addr == ram.IME+2 || addr == ram.IME+3:
		g.RAM.Set8(addr, b&0b1)
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
	case ram.SRAM(addr):
		g.RAM.Set8(addr, b)
		g.DoSav = true
	default:
		g.RAM.Set8(addr, b)
	}
}
