package gba

import (
	"fmt"
	"os"
	"runtime"

	"github.com/pokemium/magia/pkg/gba/apu"
	"github.com/pokemium/magia/pkg/gba/cart"
	"github.com/pokemium/magia/pkg/gba/ram"
	"github.com/pokemium/magia/pkg/gba/timer"
	"github.com/pokemium/magia/pkg/gba/video"
	"github.com/pokemium/magia/pkg/util"
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
	irqVBlank  IRQID = 0x00
	irqHBlank  IRQID = 0x01
	irqVCount  IRQID = 0x02
	irqTimer0  IRQID = 0x03
	irqTimer1  IRQID = 0x04
	irqTimer2  IRQID = 0x05
	irqTimer3  IRQID = 0x06
	irqSerial  IRQID = 0x07
	irqDMA0    IRQID = 0x08
	irqDMA1    IRQID = 0x09
	irqDMA2    IRQID = 0x0a
	irqDMA3    IRQID = 0x0b
	irqKEY     IRQID = 0x0c
	irqGamePak IRQID = 0x0d
)

// GBA is core object
type GBA struct {
	Reg
	video      *video.Video
	CartHeader *cart.Header
	RAM        ram.RAM
	inst       Inst
	cycle      int
	Frame      uint
	halt       bool
	pipe       Pipe
	timers     timer.Timers
	dma        [4]*DMA
	joypad     Joypad
	DoSav      bool
	apu        *apu.APU
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
func New(src []byte, soundBuf *[]byte, isDebug bool, mute bool) *GBA {
	g := &GBA{
		Reg:        *NewReg(),
		video:      video.NewVideo(),
		CartHeader: cart.New(src),
		RAM:        *ram.New(src),
		dma:        NewDMA(),
		apu:        apu.New(),
		timers:     timer.New(),
	}
	g._setRAM(ram.KEYINPUT, uint32(0x3ff), 2)
	return g
}

func (g *GBA) Exit(s string) {
	fmt.Printf("Exit: %s\n", s)
	os.Exit(0)
}

var inExec = false
var accumulatedCycles = 0

func (g *GBA) exec(cycles int) {
	if g.halt {
		tmp := g.cycle
		g.timer(cycles)
		g.cycle = tmp
		return
	}

	for g.cycle < cycles {
		inExec = true
		g.step()
		inExec = false
		if g.halt {
			g.timer(cycles - g.cycle)
		} else {
			g.timer(accumulatedCycles)
			accumulatedCycles = 0
		}
	}
	g.cycle -= cycles
}

func (g *GBA) step() {
	g.inst = g.pipe.inst[0]
	g.pipe.inst[0] = g.pipe.inst[1]

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
	g.setPrivMode(mode)
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
	g.video.RenderPath.Vcount = 0

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
	g.video.SetVBlank(true)
	g.dmaTransfer(dmaVBlank)
	for y := 0; y < 67; y++ {
		g.scanline()
	}
	g.video.SetVBlank(false) // clear on 227

	// line 227
	g.scanline()

	g.video.RenderPath.StartDraw()

	if g.Frame%2 == 0 {
		g.joypad.Read()
	}

	apu.SoundBufferWrap()
	g.Frame++

	g.apu.Play()
}

func (g *GBA) scanline() {
	dispstat := uint16(g._getRAM(ram.DISPSTAT))
	vCount, lyc := byte(g.video.RenderPath.Vcount), byte(g._getRAM(ram.DISPSTAT+1))
	if vCount == lyc {
		if util.Bit(dispstat, 5) {
			g.triggerIRQ(irqVCount)
		}
	}

	g.exec(1006)

	// HBlank
	if !g.video.VBlank() {
		if util.Bit(dispstat, 4) {
			g.triggerIRQ(irqHBlank)
		}
	}

	g.video.SetHBlank(true)
	g.dmaTransfer(dmaHBlank)
	g.exec(1232 - 1006)
	g.apu.SoundClock(1232)
	g.video.SetHBlank(false)

	vcount := g.video.RenderPath.Vcount
	if vcount < video.VERTICAL_PIXELS {
		g.video.RenderPath.DrawScanline(vcount)
	}
	g.video.RenderPath.Vcount++ // increment vcount
}

// Draw GBA screen by 1 frame
func (g *GBA) Draw() []byte { return g.video.RenderPath.FinishDraw() }

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
	if len(bs) > 65536*2 {
		return
	}
	for i, b := range bs {
		if i < 65536 {
			g.RAM.SRAM[i] = b
		}
		g.RAM.Flash[i] = b
	}
}

func (g *GBA) in(addr, start, end uint32) bool {
	return addr >= start && addr <= end
}

func (g *GBA) interwork() {
	g.SetCPSRFlag(flagT, (g.R[15]&1) > 0)

	if g.GetCPSRFlag(flagT) {
		g.R[15] &= ^uint32(1)
	} else {
		g.R[15] &= ^uint32(3)
	}
	g.pipelining()
}

func (g *GBA) timer(c int) {
	if inExec {
		accumulatedCycles += c
		return
	}
	if c == 0 {
		return
	}

	g.cycle += c
	if timer.Enable == 0 {
		return
	}
	irqs := g.timers.Tick(c, uint16(g.apu.Load32(apu.SOUNDCNT_H)), func(ch int) { g.dmaTransferFifo(ch) })
	for i, irq := range irqs {
		if irq {
			g.triggerIRQ(irqTimer0 + IRQID(i))
		}
	}
}

func (g *GBA) SetJoypadHandler(h [10](func() bool)) {
	hp := [10]*func() bool{&h[0], &h[1], &h[2], &h[3], &h[4], &h[5], &h[6], &h[7], &h[8], &h[9]}
	g.joypad.SetHandler(hp)
}

func (g *GBA) SetAudioBuffer(s []byte) {
	g.apu.SetBuffer(s)
}

func (g *GBA) PanicHandler(place string, stack bool) {
	if err := recover(); err != nil {
		fmt.Fprintf(os.Stderr, "%s emulation error: %s in 0x%08x\n", place, err, g.inst.loc)
		for depth := 0; ; depth++ {
			_, file, line, ok := runtime.Caller(depth)
			if !ok {
				break
			}
			fmt.Printf("======> %d: %v:%d\n", depth, file, line)
		}
		g.Exit("")
	}
}
