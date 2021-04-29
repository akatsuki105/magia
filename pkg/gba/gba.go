package gba

import (
	"fmt"
	"image"
	"mettaur/pkg/cart"
	"mettaur/pkg/gpu"
	"mettaur/pkg/joypad"
	"mettaur/pkg/ram"
	"mettaur/pkg/util"
	"os"
)

const (
	resetVec         uint32 = 0x00
	undVec           uint32 = 0x04
	swiVec           uint32 = 0x08
	prefetchAbortVec uint32 = 0xc
	dataAbortVec     uint32 = 0x10
	addr26BitVec     uint32 = 0x14
	irqVec           uint32 = 0x18
	fiqVec           uint32 = 0x1c
)

type IRQID int

const (
	irqVBlank  = 0x00
	irqHBlank  = 0x01
	irqVCount  = 0x02
	irqTimer0  = 0x03
	irqTimer1  = 0x04
	irqTimer2  = 0x05
	irqTimer3  = 0x06
	irqSerial  = 0x07
	irqDMA0    = 0x08
	irqDMA1    = 0x09
	irqDMA2    = 0x0a
	irqDMA3    = 0x0b
	irqKEY     = 0x0c
	irqGamePak = 0x0d
)

// GBA is core object
type GBA struct {
	Reg
	GPU        gpu.GPU
	CartHeader *cart.Header
	RAM        ram.RAM
	inst       Inst
	cycle      int
	Frame      uint
	line       int
	halt       bool
	pipe       Pipe
	timers     Timers
	dma        [4]*DMA
	joypad     joypad.Joypad
	DoSav      bool
	apu        *APU
}

type Pipe struct {
	inst [2]Inst
	ok   bool
}

type Inst struct {
	inst uint32
	loc  uint32
}

// New GBA
func New(src []byte) *GBA {
	g := &GBA{
		Reg:        *NewReg(),
		GPU:        *gpu.New(),
		CartHeader: cart.New(src),
		RAM:        *ram.New(src),
		dma:        NewDMA(),
		apu:        newAPU(),
		timers:     newTimers(),
	}
	g._setRAM(ram.KEYINPUT, uint32(0x3ff), 2)
	g.apu.play()
	return g
}

func (g *GBA) Exit(s string) {
	fmt.Printf("Exit: %s\n", s)
	if debug {
		g.PrintHistory()
	}
	g.apu.exit()
	os.Exit(0)
}

func (g *GBA) exec(cycles int) {
	if g.halt {
		g.timer(cycles)
		return
	}

	for g.cycle < cycles {
		g.step()
		if g.halt {
			g.timer(cycles - g.cycle)
		}
	}
	g.cycle -= cycles
}

var counter = 0

func (g *GBA) step() {
	g.inst = g.pipe.inst[0]
	g.pipe.inst[0] = g.pipe.inst[1]

	if debug {
		g.pushHistory()
		for _, bk := range breakPoint {
			if g.inst.loc == bk {
				g.breakpoint()
			}
		}
	}

	if g.GetCPSRFlag(flagT) {
		g.thumbStep()
	} else {
		g.armStep()
	}
}

func (g *GBA) Reset() {
	g.R[13] = 0x03007f00
	g.CPSR = 0x1f
	g.pipelining()
}

func (g *GBA) SoftReset() {
	g._setRAM(ram.DISPCNT, uint32(0x80), 2)
	g.exception(swiVec, SWI)
}

func (g *GBA) exception(addr uint32, mode Mode) {
	cpsr := g.CPSR
	g.setOSMode(mode)
	g.setSPSR(cpsr)

	g.R[14] = g.exceptionReturn(addr)
	g.SetCPSRFlag(flagT, false)
	g.SetCPSRFlag(flagI, true)
	switch addr & 0xff {
	case resetVec, fiqVec:
		g.SetCPSRFlag(flagF, true)
	}
	g.R[15] = addr
	g.pipelining()
}

// Update GBA by 1 frame
func (g *GBA) Update() {
	g.line, g.GPU.IO[gpu.VCOUNT] = 0, 0

	// line 0~159
	for y := 0; y < 160; y++ {
		g.scanline()
	}

	// VBlank
	dispstat := uint16(g._getRAM(ram.DISPSTAT))
	if util.Bit(dispstat, 3) {
		g.triggerIRQ(irqVBlank)
	}

	// line 160~226
	g.GPU.SetVBlank(true)
	g.dmaTransfer(dmaVBlank)
	for y := 0; y < 67; y++ {
		g.scanline()
	}
	g.GPU.SetVBlank(false) // clear on 227

	// line 227
	g.scanline()

	if g.Frame%2 == 0 {
		g.joypad.Read()
	}

	soundBufferWrap()
	g.Frame++
}

func (g *GBA) scanline() {
	dispstat := uint16(g._getRAM(ram.DISPSTAT))
	g.exec(1006)

	// HBlank
	if !g.GPU.VBlank() {
		if util.Bit(dispstat, 4) {
			g.triggerIRQ(irqHBlank)
		}
	}

	g.GPU.SetHBlank(true)
	g.dmaTransfer(dmaHBlank)
	g.exec(1232 - 1006)
	g.soundClock(1232)
	g.GPU.SetHBlank(false)

	vCount, lyc := g.GPU.IncrementVCount(), byte(g._getRAM(ram.DISPSTAT+1))
	if vCount == lyc {
		if util.Bit(dispstat, 5) {
			g.triggerIRQ(irqVCount)
		}
	}

	g.line++
}

// Draw GBA screen by 1 frame
func (g *GBA) Draw() *image.RGBA {
	return g.GPU.Draw()
}

func (g *GBA) checkIRQ() {
	cond1 := !g.GetCPSRFlag(flagI)
	cond2 := g._getRAM(ram.IME)&0b1 > 0
	cond3 := uint16(g._getRAM(ram.IE))&uint16(g._getRAM(ram.IF)) > 0
	if cond1 && cond2 && cond3 {
		g.exception(irqVec, IRQ)
	}
}

func (g *GBA) triggerIRQ(irq IRQID) {
	// if |= flag
	iack := uint16(g._getRAM(ram.IF))
	iack = iack | (1 << irq)
	g.RAM.IO[ram.IOOffset(ram.IF)], g.RAM.IO[ram.IOOffset(ram.IF+1)] = byte(iack), byte(iack>>8)

	g.halt = false
	if debug {
		g.pushIRQHistory(IRQHistory{
			irq:   irq,
			start: g.inst.loc,
			reg:   g.Reg,
		})
	}
	g.checkIRQ()
}

func (g *GBA) pipelining() {
	t := g.GetCPSRFlag(flagT)
	g.R[15] = util.Align2(g.R[15])
	if t {
		g.pipe.inst[0] = Inst{
			inst: uint32(g.getRAM16(g.R[15], false)),
			loc:  g.R[15],
		}
		g.R[15] += 2
		g.pipe.inst[1] = Inst{
			inst: uint32(g.getRAM16(g.R[15], true)),
			loc:  g.R[15],
		}
		g.R[15] += 2
	} else {
		g.pipe.inst[0] = Inst{
			inst: g.getRAM32(g.R[15], false),
			loc:  g.R[15],
		}
		g.R[15] += 4
		g.pipe.inst[1] = Inst{
			inst: g.getRAM32(g.R[15], true),
			loc:  g.R[15],
		}
		g.R[15] += 4
	}
	g.pipe.ok = true
}

func (g *GBA) cycleS2N() int {
	s, n := 0, 0
	switch {
	case ram.GamePak0(g.R[15]) || ram.GamePak1(g.R[15]) || ram.GamePak2(g.R[15]):
		s, n = g.cycleS(g.R[15]), g.cycleN(g.R[15])
	default:
		return 0
	}

	if !g.GetCPSRFlag(flagT) {
		n += s
		s *= 2
	}
	return n - s
}

func (g *GBA) exceptionReturn(vec uint32) uint32 {
	pc := g.R[15]

	t := g.GetCPSRFlag(flagT)
	switch vec {
	case undVec, swiVec:
		if t {
			pc -= 2
		} else {
			pc -= 4
		}
	case fiqVec, irqVec, prefetchAbortVec:
		if !t {
			pc -= 4
		}
	}
	return pc
}

func (g *GBA) CartInfo() string {
	str := `%s
ROM size: %s`
	return fmt.Sprintf(str, g.CartHeader, util.FormatSize(uint(g.RAM.ROMSize)))
}

func (g *GBA) LoadSav(bs []byte) {
	if len(bs) > 65536 {
		return
	}
	for i, b := range bs {
		g.RAM.SRAM[i] = b
	}
}

func (g *GBA) in(addr, start, end uint32) bool {
	return addr >= start && addr <= end
}
