package gba

import (
	"fmt"
	"image"
	"mettaur/pkg/cart"
	"mettaur/pkg/gpu"
	"mettaur/pkg/ram"
	"mettaur/pkg/timer"
	"mettaur/pkg/util"
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
	frame      uint
	line       int
	halt       bool
	pipe       Pipe
	debug      Debug
	timers     timer.Timers
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
		debug:      Debug{},
	}
	g._setRAM16(ram.KEYINPUT, 0x3ff)
	return g
}

func (g *GBA) Exit() {
	g.exitDebug()
}

func (g *GBA) exec(cycles int) {
	if g.halt {
		g.timer(cycles)
		return
	}

	for g.cycle < cycles {
		g.step()
		if g.halt {
			g.cycle = cycles
		}
	}
	g.cycle -= cycles
}

var counter = 0

func (g *GBA) step() {
	g.inst = g.pipe.inst[0]
	g.pipe.inst[0] = g.pipe.inst[1]

	for _, bk := range breakPoint {
		if g.inst.loc == bk {
			fmt.Printf("Breakpoint: 0x%04x\n", g.inst.loc)
			g.printRegister()
			fmt.Println()
			counter++
		}
	}

	if g.GetCPSRFlag(flagT) {
		g.thumbStep()
	} else {
		g.armStep()
	}
}

func (g *GBA) Reset() {
	g.exception(resetVec, SWI)
}

func (g *GBA) SoftReset() {
	g._setRAM16(ram.DISPCNT, 0x80)
	g.exception(swiVec, SWI)
}

func (g *GBA) exception(addr uint32, mode Mode) {
	cpsr := g.CPSR
	g.setOSMode(mode)
	g.setSPSR(cpsr)

	g.R[14] = g.inst.loc + g.exceptionNN(addr)
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
	g.line = 0
	g.GPU.IO[gpu.VCOUNT] = 0

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
	for y := 0; y < 67; y++ {
		g.scanline()
	}
	g.GPU.SetVBlank(false) // clear on 227

	// line 227
	g.scanline()

	g.frame++
}

func (g *GBA) scanline() {
	dispstat := uint16(g._getRAM(ram.DISPSTAT))
	vCount := g.GPU.IncrementVCount()
	if vCount == byte(g._getRAM(ram.DISPSTAT+1)) {
		if util.Bit(dispstat, 5) {
			g.triggerIRQ(irqVCount)
		}
	}

	g.exec(1006)

	// HBlank
	if !g.GPU.VBlank() {
		if util.Bit(dispstat, 4) {
			g.triggerIRQ(irqHBlank)
		}
	}

	g.GPU.SetHBlank(true)
	g.exec(1232 - 1006)
	g.GPU.SetHBlank(false)

	g.line++
}

// Draw GBA screen by 1 frame
func (g *GBA) Draw() *image.RGBA {
	return g.GPU.Draw()
}

func (g *GBA) checkIRQ() {
	cond1 := !g.GetCPSRFlag(flagI)
	cond2 := util.ToBool(g._getRAM(ram.IME) & 0b1)
	cond3 := util.ToBool(uint16(g._getRAM(ram.IE)) & uint16(g._getRAM(ram.IF)))
	if cond1 && cond2 && cond3 {
		g.exception(irqVec, IRQ)
	}
}

func (g *GBA) triggerIRQ(irq int) {
	// if |= flag
	iack := uint16(g._getRAM(ram.IF))
	iack = iack | (1 << irq)
	g.RAM.IO[ram.IOOffset(ram.IF)] = byte(iack)
	g.RAM.IO[ram.IOOffset(ram.IF+1)] = byte(iack >> 8)

	g.halt = false
	g.checkIRQ()
}

func (g *GBA) pipelining() {
	t := g.GetCPSRFlag(flagT)
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

func (g *GBA) exceptionNN(vec uint32) uint32 {
	nn := uint32(0)
	t := g.GetCPSRFlag(flagT)
	switch vec {
	case dataAbortVec:
		nn = 8
	case fiqVec, irqVec, prefetchAbortVec:
		nn = 4
	case undVec, swiVec:
		if t {
			nn = 2
		} else {
			nn = 4
		}
	}
	return nn
}
