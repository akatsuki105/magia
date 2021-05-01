package gba

import (
	"mettaur/pkg/ram"
	"mettaur/pkg/util"
)

const (
	SoundATimer = 10
	SoundBTimer = 14
)

var (
	wsN  = [4]int{4, 3, 2, 8}
	wsS0 = [2]int{2, 1}
	wsS1 = [2]int{4, 1}
	wsS2 = [2]int{8, 1}
)

func (g *GBA) cycleN(addr uint32) int {
	switch {
	case ram.EWRAM(addr):
		return 3
	case ram.GamePak0(addr):
		offset := ram.IOOffset(ram.WAITCNT)
		idx := g.RAM.IO[offset] >> 2 & 0b11
		return wsN[idx] + 1
	case ram.GamePak1(addr):
		idx := g._getRAM(ram.WAITCNT) >> 5 & 0b11
		return wsN[idx] + 1
	case ram.GamePak2(addr):
		idx := g._getRAM(ram.WAITCNT) >> 8 & 0b11
		return wsN[idx] + 1
	case ram.SRAM(addr):
		idx := g._getRAM(ram.WAITCNT) & 0b11
		return wsN[idx] + 1
	}
	return 1
}

func (g *GBA) cycleS(addr uint32) int {
	switch {
	case ram.EWRAM(addr):
		return 3
	case ram.GamePak0(addr):
		offset := ram.IOOffset(ram.WAITCNT)
		idx := g.RAM.IO[offset] >> 4 & 0b1
		return wsS0[idx] + 1
	case ram.GamePak1(addr):
		idx := g._getRAM(ram.WAITCNT) >> 7 & 0b1
		return wsS1[idx] + 1
	case ram.GamePak2(addr):
		idx := g._getRAM(ram.WAITCNT) >> 10 & 0b1
		return wsS2[idx] + 1
	case ram.SRAM(addr):
		idx := g._getRAM(ram.WAITCNT) & 0b11
		return wsN[idx] + 1
	}
	return 1
}

func (g *GBA) waitBus(addr uint32, size int, s bool) int {
	busWidth := ram.BusWidth(addr)
	if busWidth == 8 {
		return 5 * (size / 8)
	}

	if size > busWidth {
		if s {
			return 2 * g.cycleS(addr)
		}
		return g.cycleN(addr) + g.cycleS(addr+2)
	}

	if s {
		return g.cycleS(addr)
	}
	return g.cycleN(addr)
}

func (g *GBA) timer(c int) {
	g.cycle += c
	irqs := g.Tick(c)
	for i, irq := range irqs {
		if irq {
			g.triggerIRQ(IRQID(i + 3))
		}
	}
}

type Timers [4]*Timer

func newTimers() Timers { return Timers{&Timer{}, &Timer{}, &Timer{}, &Timer{}} }

type Timer struct {
	Count   uint16
	Next    int // if this value is 0, count up timer.Count
	Reload  uint16
	Control byte
}

func (t *Timer) cascade() bool { return util.Bit(t.Control, 2) }
func (t *Timer) irq() bool     { return util.Bit(t.Control, 6) }
func (t *Timer) enable() bool  { return util.Bit(t.Control, 7) }
func (t *Timer) increment(inc int) bool {
	previous := t.Count
	t.Count += uint16(inc)
	return t.Count < previous // if overflow occurs
}
func (t *Timer) overflow() bool {
	t.Count = t.Reload
	return t.irq()
}

// IsIO returns true if addr is for Timer IO register.
func IsTimerIO(addr uint32) bool { return (addr >= 0x0400_0100) && (addr < 0x0400_0110) }
func (ts *Timers) GetIO(offset uint32) uint32 {
	idx, ofs := offset/4, offset%4
	switch ofs {
	case 0:
		return uint32(ts[idx].Control)<<16 | uint32(ts[idx].Count)
	case 1:
		return uint32(ts[idx].Count >> 8)
	case 2:
		return uint32(ts[idx].Control)
	case 3:
		return 0
	}
	return 0
}

func (ts *Timers) SetIO(offset uint32, b byte) {
	idx, ofs := offset/4, offset%4
	switch ofs {
	case 0:
		ts[idx].Reload = (ts[idx].Reload & 0xff00) | uint16(b)
	case 1:
		ts[idx].Reload = (ts[idx].Reload & 0xff) | (uint16(b) << 8)
	case 2:
		previous := util.Bit(ts[idx].Control, 7)
		ts[idx].Control = b
		// The reload value is copied into the counter when the timer start bit becomes changed from 0 to 1.
		if !previous && util.Bit(ts[idx].Control, 7) {
			ts[idx].Count = ts[idx].Reload
		}
	}
}

var clockShift = [4]byte{0, 6, 8, 10}

func (g *GBA) Tick(cycles int) [4]bool {
	overflow, irq := false, [4]bool{}
	ts := &g.timers

	if ts[0].enable() {
		ts[0].Next += cycles
		inc := ts[0].Next >> clockShift[ts[0].Control&0b11]
		if inc > 0 {
			ts[0].Next -= inc << clockShift[ts[0].Control&0b11]
			overflow = ts[0].increment(inc)
			if overflow {
				cnth := uint16(g._getRAM(ram.SOUNDCNT_H))
				if !util.Bit(cnth, SoundATimer) {
					g.fifoALoad()
					if fifoALen <= 0x10 { // Request more data per DMA
						g.dmaTransferFifo(1)
					}
				}
				if !util.Bit(cnth, SoundBTimer) {
					g.fifoBLoad()
					if fifoBLen <= 0x10 {
						g.dmaTransferFifo(2)
					}
				}
				if ts[0].overflow() {
					irq[0] = true
				}
			}
		}
	}

	for i := 1; i < 4; i++ {
		if !ts[i].enable() {
			overflow = false
			continue
		}

		inc := 0
		if ts[i].cascade() {
			if overflow {
				inc = 1
			}
		} else {
			ts[i].Next += cycles
			inc = ts[i].Next >> clockShift[ts[i].Control&0b11]
			ts[i].Next -= inc << clockShift[ts[i].Control&0b11]
		}

		if inc > 0 {
			overflow = ts[i].increment(inc)
			if overflow {
				if i == 1 {
					cnth := uint16(g._getRAM(ram.SOUNDCNT_H))
					if util.Bit(cnth, SoundATimer) {
						g.fifoALoad()
						if fifoALen <= 0x10 {
							g.dmaTransferFifo(1)
						}
					}
					if util.Bit(cnth, SoundBTimer) {
						g.fifoBLoad()
						if fifoBLen <= 0x10 {
							g.dmaTransferFifo(2)
						}
					}
				}

				if ts[i].overflow() {
					irq[i] = true
				}
			}
		}
	}

	return irq
}
