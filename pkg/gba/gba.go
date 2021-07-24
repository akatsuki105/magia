package gba

import (
	"fmt"
	"os"
	"runtime"

	"github.com/pokemium/magia/pkg/gba/apu"
	"github.com/pokemium/magia/pkg/gba/cart"
	"github.com/pokemium/magia/pkg/gba/joypad"
	"github.com/pokemium/magia/pkg/gba/ram"
	"github.com/pokemium/magia/pkg/gba/scheduler"
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
	Frame      uint
	halt       bool
	pipe       Pipe
	timers     *timer.Timers
	scheduler  *scheduler.Scheduler
	dma        [4]*DMA
	joypad     *joypad.Joypad
	DoSav      bool
	Sound      *apu.APU
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
func New(src []byte, j [10]func() bool, audioStream []byte) *GBA {
	s := scheduler.New()
	g := &GBA{
		Reg:        *NewReg(),
		video:      video.NewVideo(),
		CartHeader: cart.New(src),
		RAM:        *ram.New(src),
		dma:        NewDMA(),
		Sound:      apu.New(true, audioStream),
		joypad:     joypad.New(j),
		scheduler:  s,
	}
	g.timers = timer.New(s, &g.RAM, func(i int) { g.triggerIRQ(IRQID(i)) }, func(ch int) { g.dmaTransferFifo(ch) })
	g._setRAM(ram.KEYINPUT, uint32(0x3ff), 2)
	g.softReset()

	g.video.RenderPath.Vcount = 170
	g.RAM.IO[ram.IOOffset(ram.VCOUNT)] = 176
	g.RAM.IO[ram.IOOffset(ram.VCOUNT)+1] = 0
	g.scheduler.ScheduleEvent(scheduler.StartHBlank, g.startHBlank, 170)
	return g
}

func (g *GBA) Exit(s string) {
	fmt.Printf("Exit: %s\n", s)
	os.Exit(0)
}

func (g *GBA) step() {
	if g.halt {
		g.timers.Tick(int(g.scheduler.Next() - g.scheduler.Cycle()))
		return
	}

	g.inst = g.pipe.inst[0]
	g.pipe.inst[0] = g.pipe.inst[1]

	if g.GetCPSRFlag(flagT) {
		g.thumbStep()
	} else {
		g.armStep()
	}
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
	frame := g.Frame
	for frame == g.Frame {
		g.step()
	}

	if g.Frame%2 == 0 {
		g.joypad.Read()
	}

	apu.SoundBufferWrap()

	g.Sound.Play()
}

// _startHdraw
func (g *GBA) startHDraw(cyclesLate uint64) {
	g.scheduler.ScheduleEvent(scheduler.StartHBlank, g.startHBlank, video.HDRAW_LENGTH-cyclesLate)

	g.video.RenderPath.Vcount++
	if g.video.RenderPath.Vcount == video.VERTICAL_TOTAL_PIXELS {
		g.video.RenderPath.Vcount = 0
	}
	g._setRAM(ram.VCOUNT, uint32(g.video.RenderPath.Vcount), 2)

	if g.video.RenderPath.Vcount < video.VERTICAL_PIXELS {
		g.video.ShouldStall = true
	}

	dispstat := g.video.Dispstat()
	if g.video.RenderPath.Vcount == (dispstat >> 8) {
		dispstat = uint16(util.SetBit16(dispstat, video.VCOUNTER_FLAG, true))
		if util.Bit(dispstat, video.VCOUNTER_IRQ) {
			g.triggerIRQ(irqVCount)
		}
	} else {
		dispstat = uint16(util.SetBit16(dispstat, video.VCOUNTER_FLAG, false))
	}
	g.video.SetDispstat(dispstat)

	// Note: state may be recorded during callbacks, so ensure it is consistent!
	switch g.video.RenderPath.Vcount {
	case video.VERTICAL_PIXELS:
		g.video.SetDispstat(util.SetBit16(dispstat, video.VBLANK_FLAG, true))
		g.dmaTransfer(dmaVBlank)
		if util.Bit(dispstat, video.VBLANK_IRQ) {
			g.triggerIRQ(irqVBlank)
		}
	case video.VERTICAL_TOTAL_PIXELS - 1:
		g.video.SetDispstat(util.SetBit16(dispstat, video.VBLANK_FLAG, false))
	}
}

func (g *GBA) startHBlank(cyclesLate uint64) {
	g.scheduler.ScheduleEvent(scheduler.MidHBlank, g.midHBlank, video.HBLANK_LENGTH-video.HBLANK_FLIP-cyclesLate)

	// Begin Hblank
	dispstat := g.video.Dispstat()
	dispstat = util.SetBit16(dispstat, video.HBLANK_FLAG, true)
	if g.video.RenderPath.Vcount < video.VERTICAL_PIXELS {
		g.video.RenderPath.DrawScanline(g.video.RenderPath.Vcount)
		g.dmaTransfer(dmaHBlank)
	}

	if g.video.RenderPath.Vcount >= 2 && g.video.RenderPath.Vcount < video.VERTICAL_PIXELS+2 {
		// TODO
	}

	if util.Bit(dispstat, video.HBLANK_IRQ) {
		g.triggerIRQ(irqHBlank)
	}

	g.video.ShouldStall = false
	g.video.SetDispstat(dispstat)
}

func (g *GBA) midHBlank(cyclesLate uint64) {
	g.Sound.SoundClock(1232)

	dispstat := g.video.Dispstat()
	g.video.SetDispstat(util.SetBit16(dispstat, video.HBLANK_FLAG, false))
	g.scheduler.ScheduleEvent(scheduler.StartHDraw, g.startHDraw, video.HBLANK_FLIP-cyclesLate)
	if g.video.RenderPath.Vcount == 227 {
		g.Frame++
	}
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
