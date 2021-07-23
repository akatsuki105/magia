package timer

import (
	"github.com/pokemium/magia/pkg/gba/scheduler"
	"github.com/pokemium/magia/pkg/util"
)

const (
	ControlCascade = 2
	ControlIrq     = 6
	ControlEnable  = 7
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

func (t *Timer) increment(inc int) bool {
	previous := t.Count
	t.Count += uint16(inc)
	return t.Count < previous // if overflow occurs
}

func (t *Timer) overflow() bool {
	t.Count += t.Reload
	return util.Bit(t.Control, ControlIrq)
}
