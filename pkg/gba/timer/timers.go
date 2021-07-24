package timer

import (
	"github.com/pokemium/magia/pkg/gba/ram"
	"github.com/pokemium/magia/pkg/gba/scheduler"
	"github.com/pokemium/magia/pkg/util"
)

func (ts *Timers) Tick(cycles int) {
	if ts.InExec {
		ts.cycles += cycles
		return
	}

	cycles += ts.cycles
	ts.scheduler.Add(uint64(cycles))
	for {
		if ts.scheduler.Next() > ts.scheduler.Cycle() {
			break
		}
		ts.scheduler.DoEvent()
	}
}

// IsIO returns true if addr is for Timer IO register.
func IsIO(addr uint32) bool { return (addr >= 0x0400_0100) && (addr < 0x0400_0110) }

type Timers struct {
	Enable    byte
	InExec    bool
	cycles    int
	timers    [4]*Timer
	scheduler *scheduler.Scheduler
	ram       *ram.RAM
	irq       func(int, uint64)
	dma       func(int)
}

func New(s *scheduler.Scheduler, ram *ram.RAM, irq func(int, uint64), dma func(int)) *Timers {
	p := &Timers{
		scheduler: s,
		ram:       ram,
		irq:       irq,
		dma:       dma,
	}
	p.timers = [4]*Timer{NewTimer(p, 0), NewTimer(p, 1), NewTimer(p, 2), NewTimer(p, 3)}
	return p
}

func (ts *Timers) WriteTMCNTL(timerId int, reload uint16) {
	ts.timers[timerId].reload = reload
}

func (ts *Timers) WriteTMCNTH(timerId int, control uint16) {
	timer := ts.timers[timerId]
	timer.UpdateRegister(0)

	prescaleTable := [4]uint32{0, 6, 8, 10}
	prescaleBits := prescaleTable[control&0b11]

	oldFlags := timer.flags
	timer.flags = (timer.flags & 0xffff_fff0) | prescaleBits
	timer.flags = util.SetBit32(timer.flags, CountUp, timer.id > 0 && util.Bit(control, 2))
	timer.flags = util.SetBit32(timer.flags, DoIrq, util.Bit(control, 6))
	timer.flags = util.SetBit32(timer.flags, Enable, util.Bit(control, 7))

	reschedule := false
	if util.Bit(oldFlags, Enable) != util.Bit(timer.flags, Enable) {
		reschedule = true
		if util.Bit(timer.flags, Enable) {
			timer.setCounter(timer.reload)
		}
	} else if util.Bit(oldFlags, CountUp) != util.Bit(timer.flags, CountUp) {
		reschedule = true
	} else if (oldFlags & PrescaleBits) != (timer.flags & PrescaleBits) {
		reschedule = true
	}

	if reschedule {
		ts.scheduler.DescheduleEvent(timer.event.name)
		if util.Bit(timer.flags, Enable) && !util.Bit(timer.flags, CountUp) {
			tickMask := uint64((1 << prescaleBits) - 1)
			timer.lastEvent = ts.scheduler.Cycle() & ^tickMask
			timer.UpdateRegister(0)
		}
	}
}
