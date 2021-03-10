package gba

import (
	"image"
	"mettaur/pkg/cart"
	"mettaur/pkg/gpu"
	"mettaur/pkg/ram"
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
	lastAddr   uint32
	cycle      int
	PC         uint32
	frame      uint
	halt       bool
}

// New GBA
func New(src []byte) *GBA {
	return &GBA{
		Reg:        *NewReg(),
		GPU:        *gpu.New(),
		CartHeader: cart.New(src),
		RAM:        *ram.New(src),
	}
}

func (g *GBA) step() {
	g.checkIRQ()

	if g.halt {
		g.timer(1)
		return
	}

	if g.GetCPSRFlag(flagT) {
		g.thumbStep()
	} else {
		g.armStep()
	}
}

func (g *GBA) exception(addr uint32, mode Mode) {
	nn := uint32(4)
	if g.GetCPSRFlag(flagT) {
		nn = 2
	}
	g.R[14] = g.PC + nn
	g.setOSMode(mode)
	g.SetCPSRFlag(flagT, false)
	g.SetCPSRFlag(flagI, true)
	switch addr & 0xff {
	case resetVec, fiqVec:
		g.SetCPSRFlag(flagF, true)
	}
	g.R[15] = addr
}

// Update GBA by 1 frame
func (g *GBA) Update() {
	for y := 0; y < 160; y++ {
		g.scanline()
	}

	// VBlank
	dispstat := uint16(g.getRAM(ram.DISPSTAT))
	if util.Bit(dispstat, 3) {
		g.triggerIRQ(irqVBlank)
	}
	dispstat = dispstat | 1
	g.setRAM16(ram.DISPSTAT, dispstat)

	for y := 0; y < 68; y++ {
		g.scanline()
	}

	g.frame++
}

func (g *GBA) scanline() {
	g.cycle = 0
	for g.cycle <= 240*4 {
		g.step()
	}

	// HBlank
	dispstat := uint16(g.getRAM(ram.DISPSTAT))
	if !g.GPU.VBlank() {
		if util.Bit(dispstat, 4) {
			g.triggerIRQ(irqHBlank)
		}
		dispstat = dispstat | 2
		g.setRAM16(ram.DISPSTAT, dispstat)
	}

	for g.cycle <= 68*4 {
		g.step()
	}
}

// Draw GBA screen by 1 frame
func (g *GBA) Draw() *image.RGBA {
	return g.GPU.Draw()
}

func (g *GBA) checkIRQ() {
	cond1 := !g.GetCPSRFlag(flagI)
	cond2 := util.ToBool(g.getRAM(ram.IME) & 0b1)
	cond3 := util.ToBool(uint16(g.getRAM(ram.IE)) & uint16(g.getRAM(ram.IF)))
	if cond1 && cond2 && cond3 {
		g.exception(irqVec, IRQ)
	}
}

func (g *GBA) triggerIRQ(irq int) {
	// if |= flag
	iack := uint16(g.getRAM(ram.IF))
	iack = iack | (1 << irq)
	g.setRAM16(ram.IF, iack)

	g.halt = false
	g.checkIRQ()
}
