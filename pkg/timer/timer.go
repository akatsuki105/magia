package timer

import "mettaur/pkg/util"

// IO offset
const (
	TM0CNT_L = 0x0
	TM0CNT_H = 0x2
	TM1CNT_L = 0x4
	TM1CNT_H = 0x6
	TM2CNT_L = 0x8
	TM2CNT_H = 0xa
	TM3CNT_L = 0xc
	TM3CNT_H = 0xe
)

type Timers [4]Timer

type Timer struct {
	Count   uint16
	Next    int // if this value is 0, count up timer.Count
	Reload  uint16
	Control byte
}

func (t *Timer) period() int {
	switch t.Control & 0b11 {
	case 0:
		return 1
	case 1:
		return 64
	case 2:
		return 256
	default:
		return 1024
	}
}
func (t *Timer) cascade() bool { return util.Bit(t.Control, 2) }
func (t *Timer) irq() bool     { return util.Bit(t.Control, 6) }
func (t *Timer) enable() bool  { return util.Bit(t.Control, 7) }
func (t *Timer) increment() bool {
	t.Next = t.period()
	previous := t.Count
	t.Count++
	current := t.Count
	return current < previous // if overflow occurs
}
func (t *Timer) overflow() bool {
	t.Count = t.Reload
	return t.irq()
}

// IsIO returns true if addr is for Timer IO register.
func IsIO(addr uint32) bool {
	return (addr >= 0x0400_0100) && (addr < 0x0400_0110)
}
func (ts *Timers) SetIO(offset uint32, b byte) {
	idx := offset / 4
	ofs := offset % 4
	switch ofs {
	case 0:
		ts[idx].Reload = (ts[idx].Reload & 0xff00) | uint16(b)
	case 1:
		ts[idx].Reload = (ts[idx].Reload & 0x00ff) | (uint16(b) << 8)
	case 2:
		previous := util.Bit(ts[idx].Control, 7)
		ts[idx].Control = b
		// The reload value is copied into the counter when the timer start bit becomes changed from 0 to 1.
		if !previous && util.Bit(ts[idx].Control, 7) {
			ts[idx].Count = ts[idx].Reload
		}
	}
}
