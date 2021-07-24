package timer

import (
	"github.com/pokemium/magia/pkg/gba/apu"
	"github.com/pokemium/magia/pkg/gba/ram"
	"github.com/pokemium/magia/pkg/gba/scheduler"
	"github.com/pokemium/magia/pkg/util"
)

const (
	PrescaleBits = 0b1111
	CountUp      = 4
	DoIrq        = 5
	Enable       = 6
)

const (
	SoundATimer = 10
	SoundBTimer = 14
)

type Event struct {
	name     scheduler.EventName
	callback func(uint64)
}

type Timer struct {
	p         *Timers
	id        int
	reload    uint16
	lastEvent uint64
	event     *Event
	flags     uint32
}

func NewTimer(p *Timers, id int) *Timer {
	t := &Timer{
		p:  p,
		id: id,
	}

	names := [4]scheduler.EventName{scheduler.Timer0Update, scheduler.Timer1Update, scheduler.Timer2Update, scheduler.Timer3Update}
	t.event = &Event{
		name:     names[id],
		callback: t.Overflow,
	}
	return t
}

// GBATimerUpdate
// this callback is triggerd on timer's overflow
func (t *Timer) Overflow(cyclesLate uint64) {
	if util.Bit(t.flags, CountUp) {
		t.setCounter(t.reload)
	} else {
		t.UpdateRegister(cyclesLate)
	}

	if util.Bit(t.flags, DoIrq) {
		t.p.irq(0x03+t.id, cyclesLate)
	}

	if t.id < 2 {
		cnth := uint16(t.p.ram.Get(ram.SOUNDCNT_H))
		if int((cnth>>SoundATimer)&0b1) == t.id {
			apu.FifoALoad()
			if apu.FifoALen <= 0x10 {
				t.p.dma(1)
			}
		}
		if int((cnth>>SoundBTimer)&0b1) == t.id {
			apu.FifoBLoad()
			if apu.FifoBLen <= 0x10 {
				t.p.dma(2)
			}
		}
	}

	if t.id < 3 {
		nextTimer := t.p.timers[t.id+1]
		if util.Bit(nextTimer.flags, CountUp) {
			// cascade
			counter := nextTimer.counter() + 1
			nextTimer.setCounter(counter)
			if counter == 0 && util.Bit(nextTimer.flags, Enable) {
				// overflow is occured on next timer too.
				nextTimer.Overflow(cyclesLate)
			}
		}
	}
}

// GBATimerUpdateRegister
func (t *Timer) UpdateRegister(cyclesLate uint64) {
	if !util.Bit(t.flags, Enable) || util.Bit(t.flags, CountUp) {
		return
	}

	// Align Timer
	prescaleBits := t.flags & PrescaleBits
	currentTime := t.p.scheduler.Cycle() - cyclesLate
	tickMask := uint64((1 << prescaleBits) - 1)
	currentTime &= ^tickMask

	// Update register
	tickIncrement := currentTime - t.lastEvent
	t.lastEvent = currentTime
	tickIncrement >>= prescaleBits
	counter := t.counter()
	tickIncrement += uint64(counter)
	for tickIncrement >= 0x10000 {
		tickIncrement -= (0x10000 - uint64(t.reload))
	}
	t.setCounter(uint16(tickIncrement))

	tickIncrement = (0x10000 - tickIncrement) << prescaleBits
	currentTime += tickIncrement
	currentTime &= ^tickMask
	t.p.scheduler.DescheduleEvent(t.event.name)
	t.p.scheduler.ScheduleEventAbsolute(t.event.name, t.event.callback, currentTime)
}

// TMnCNT_LO
func tmcntl(timerId int) uint32 {
	return ram.TM0CNT + uint32(timerId)*4
}

func (t *Timer) counter() uint16 {
	ofs := ram.IOOffset(tmcntl(t.id))
	lower := t.p.ram.IO[ofs]
	upper := t.p.ram.IO[ofs+1]
	return uint16(upper)<<8 | uint16(lower)
}

func (t *Timer) setCounter(reload uint16) {
	ofs := ram.IOOffset(tmcntl(t.id))
	t.p.ram.IO[ofs] = byte(reload)
	t.p.ram.IO[ofs+1] = byte(reload >> 8)
}
