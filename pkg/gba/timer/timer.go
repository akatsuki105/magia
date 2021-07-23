package timer

import (
	"github.com/pokemium/magia/pkg/gba/scheduler"
	"github.com/pokemium/magia/pkg/util"
)

type Timer struct {
	Count     uint16
	Next      int // if this value is 0, count up timer.Count
	Reload    uint16
	Control   byte
	scheduler *scheduler.Scheduler
	lastEvent uint64
}

func NewTimer(s *scheduler.Scheduler) *Timer {
	return &Timer{
		scheduler: s,
	}
}

func (t *Timer) cascade() bool { return util.Bit(t.Control, 2) }

func (t *Timer) irq() bool { return util.Bit(t.Control, 6) }

func (t *Timer) enable() bool { return util.Bit(t.Control, 7) }

func (t *Timer) increment(inc int) bool {
	previous := t.Count
	t.Count += uint16(inc)
	return t.Count < previous // if overflow occurs
}

func (t *Timer) overflow() bool {
	t.Count += t.Reload
	return t.irq()
}
