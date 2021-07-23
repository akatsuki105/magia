package timer

import (
	"github.com/pokemium/magia/pkg/gba/apu"
	"github.com/pokemium/magia/pkg/gba/scheduler"
	"github.com/pokemium/magia/pkg/util"
)

const (
	SoundATimer = 10
	SoundBTimer = 14
)

// IsIO returns true if addr is for Timer IO register.
func IsIO(addr uint32) bool { return (addr >= 0x0400_0100) && (addr < 0x0400_0110) }

type Timers struct {
	Enable            byte
	InExec            bool
	accumulatedCycles int
	cnth              func() uint16
	timers            [4]*Timer
	dma               func(int)
}

func New(cnth func() uint16, s *scheduler.Scheduler, dma func(int)) Timers {
	return Timers{
		cnth:   cnth,
		timers: [4]*Timer{NewTimer(s), NewTimer(s), NewTimer(s), NewTimer(s)},
		dma:    dma,
	}
}

func (ts *Timers) GetIO(offset uint32) uint32 {
	idx, ofs := offset/4, offset%4
	switch ofs {
	case 0:
		return uint32(ts.timers[idx].Control)<<16 | uint32(ts.timers[idx].Count)
	case 1:
		return uint32(ts.timers[idx].Count >> 8)
	case 2:
		return uint32(ts.timers[idx].Control)
	case 3:
		return 0
	}
	return 0
}

func (ts *Timers) SetIO(offset uint32, b byte) {
	idx, ofs := offset/4, offset%4
	switch ofs {
	case 0:
		ts.timers[idx].Reload = (ts.timers[idx].Reload & 0xff00) | uint16(b)
	case 1:
		ts.timers[idx].Reload = (ts.timers[idx].Reload & 0xff) | (uint16(b) << 8)
	case 2:
		if util.Bit(b, 7) {
			ts.Enable |= (1 << idx)
		} else {
			ts.Enable &= ^(1 << idx)
		}
		previous := util.Bit(ts.timers[idx].Control, 7)
		ts.timers[idx].Control = b
		// The reload value is copied into the counter when the timer start bit becomes changed from 0 to 1.
		if !previous && util.Bit(b, 7) {
			ts.timers[idx].Count = ts.timers[idx].Reload
			ts.timers[idx].Next = 0
		}
	}
}

var clockShift = [4]byte{0, 6, 8, 10}

func (ts *Timers) Tick(cycles int) [4]bool {
	if ts.InExec {
		ts.accumulatedCycles += cycles
		return [4]bool{}
	}

	cycles += ts.accumulatedCycles
	ts.accumulatedCycles = 0
	if ts.Enable == 0 || cycles == 0 {
		return [4]bool{}
	}

	overflow, irq := false, [4]bool{}
	for i := 0; i < 4; i++ {
		if !ts.timers[i].enable() {
			overflow = false
			continue
		}

		inc := 0
		if i > 0 && ts.timers[i].cascade() {
			if overflow {
				inc = 1
			}
		} else {
			ts.timers[i].Next += cycles
			inc = ts.timers[i].Next >> clockShift[ts.timers[i].Control&0b11]
			ts.timers[i].Next -= (inc << clockShift[ts.timers[i].Control&0b11])
		}

		if inc > 0 {
			overflow = ts.timers[i].increment(inc)
			if overflow {
				if (ts.cnth()>>SoundATimer)&0b1 == uint16(i) {
					apu.FifoALoad()
					if apu.FifoALen <= 0x10 {
						ts.dma(1)
					}
				}
				if (ts.cnth()>>SoundBTimer)&0b1 == uint16(i) {
					apu.FifoBLoad()
					if apu.FifoBLen <= 0x10 {
						ts.dma(2)
					}
				}

				if ts.timers[i].overflow() {
					irq[i] = true
				}
			}
		}
	}

	return irq
}
