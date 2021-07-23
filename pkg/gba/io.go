package gba

import (
	"fmt"
	"strings"

	"github.com/pokemium/magia/pkg/gba/apu"
	"github.com/pokemium/magia/pkg/gba/ram"
	"github.com/pokemium/magia/pkg/gba/timer"
	"github.com/pokemium/magia/pkg/util"
)

var lastBios uint32 = 0xE129F000

func (g *GBA) _getRAM(addr uint32) uint32 {
	switch {
	case (addr >= 0x0400_0000) && (addr < 0x0400_0000+0x60):
		return g.video.Load32(addr)
	case (addr >= 0x0400_0060) && (addr < 0x0400_00A8):
		return g.Sound.Load32(addr - 0x0400_0060)
	case isDMA0IO(addr):
		return g.dma[0].get(addr - ram.DMA0SAD)
	case isDMA1IO(addr):
		return g.dma[1].get(addr - ram.DMA1SAD)
	case isDMA2IO(addr):
		return g.dma[2].get(addr - ram.DMA2SAD)
	case isDMA3IO(addr):
		return g.dma[3].get(addr - ram.DMA3SAD)
	case timer.IsTimerIO(addr):
		return g.timers.GetIO(addr - 0x0400_0100)
	case addr == ram.KEYINPUT || addr == ram.KEYINPUT+1:
		return util.LE32(g.joypad.Input[addr-ram.KEYINPUT:])
	case ram.Palette(addr):
		return g.video.RenderPath.Palette.Load32(addr)
	case ram.VRAM(addr):
		offset := ram.VRAMOffset(addr)
		return g.video.RenderPath.VRAM.LoadU32(offset)
	case ram.OAM(addr):
		offset := ram.OAMOffset(addr)
		return g.video.RenderPath.OAM.LoadU32(offset)
	default:
		value := g.RAM.Get(addr)
		if ram.BIOS(addr) {
			if ram.BIOS(g.R[15]) {
				lastBios = value
			} else {
				value = lastBios
			}
		}
		return value
	}
}
func (g *GBA) getRAM32(addr uint32, s bool) uint32 {
	g.tick(g.waitBus(addr, 32, s))
	val := g._getRAM(addr & ^uint32(3))

	if addr&3 > 0 { // https://github.com/jsmolka/gba-tests/blob/a6447c5404c8fc2898ddc51f438271f832083b7e/thumb/memory.asm#L72
		val = util.ROR(val, 8*(uint(addr)&3))
	}
	return val
}

func (g *GBA) getRAM16(addr uint32, s bool) uint32 {
	g.tick(g.waitBus(addr, 16, s))
	val := g._getRAM(addr)
	return val & 0x0000_ffff
}

func (g *GBA) getRAM8(addr uint32, s bool) byte {
	g.tick(g.waitBus(addr, 8, s))
	return byte(g._getRAM(addr))
}

func (g *GBA) setRAM32(addr, value uint32, s bool) {
	addr = util.Align4(addr)
	g.tick(g.waitBus(addr, 32, s))
	g._setRAM(addr, value, 4)
}

func (g *GBA) setRAM16(addr uint32, value uint16, s bool) {
	addr = util.Align2(addr)
	g.tick(g.waitBus(addr, 16, s))
	g._setRAM(addr, uint32(value), 2)
}

func (g *GBA) setRAM8(addr uint32, b byte, s bool) {
	g.tick(g.waitBus(addr, 8, s))
	g._setRAM(addr, uint32(b), 1)
}

func (g *GBA) _setRAM(addr uint32, val uint32, width int) {
	defer func() {
		if err := recover(); err != nil {
			if strings.Contains(fmt.Sprintln(err), "runtime error: index out of range") {
				panic(fmt.Sprintf("ram error: write to 0x%08x(%v)", addr, err))
			}
			panic(err)
		}
	}()

	switch {
	case (addr >= 0x0400_0000) && (addr < 0x0400_0000+0x60):
		switch width {
		case 1:
			g.video.Set8(addr, byte(val))
		case 2:
			g.video.Set16(addr, uint16(val))
		case 4:
			g.video.Set32(addr, val)
		}

	case g.in(addr, ram.SOUND1CNT_L, ram.SOUNDCNT_H): // sound io
		if util.Bit(byte(g.Sound.Load32(apu.SOUNDCNT_X)), 7) {
			for i := uint32(0); i < uint32(width); i++ {
				g.Sound.Store8(addr+i-ram.SOUND1CNT_L, byte(val>>(8*i)))
			}
		}

	case addr == ram.SOUNDCNT_X:
		old := byte(g.Sound.Load32(apu.SOUNDCNT_X))
		old = (old & 0xf) | (byte(val) & 0xf0)
		g.Sound.Store8(apu.SOUNDCNT_X, old)
		if !util.Bit(byte(val), 7) {
			for i := uint32(0x4000060); i <= 0x4000081; i++ {
				g.Sound.Store8(i-0x4000060, 0)
			}
			g.Sound.Store8(apu.SOUNDCNT_X, 0)
		}

	case g.in(addr, ram.WAVE_RAM, ram.WAVE_RAM+0xf): // wave ram
		if width == 2 {
			g.Sound.Store16(addr-ram.SOUND1CNT_L, uint16(val))
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

	case timer.IsTimerIO(addr):
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
		ofs := ram.PaletteOffset(addr)
		switch width {
		case 2:
			g.video.RenderPath.Palette.Store16(ofs, uint16(val))
		case 4:
			g.video.RenderPath.Palette.Store32(ofs, val)
		}

	case ram.VRAM(addr):
		ofs := ram.VRAMOffset(addr)
		switch width {
		case 2:
			g.video.RenderPath.VRAM.Store16(ofs, uint16(val))
		case 4:
			g.video.RenderPath.VRAM.Store32(ofs, val)
		}

	case ram.OAM(addr):
		ofs := ram.OAMOffset(addr)
		switch width {
		case 2:
			g.video.RenderPath.OAM.Store16(ofs, uint16(val))
		case 4:
			g.video.RenderPath.OAM.Store32(ofs, val)
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
