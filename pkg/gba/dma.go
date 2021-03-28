package gba

import (
	"fmt"
	"mettaur/pkg/util"
)

type dmaTiming uint32

const (
	dmaImmediate dmaTiming = iota
	dmaVBlank
	dmaHBlank
	dmaSpecial
)

type DMA struct {
	src, dst uint32
	cnt      uint32
}

func isDMA0IO(addr uint32) bool { return 0x0400_00B0 <= addr && addr <= 0x0400_00BB }
func isDMA1IO(addr uint32) bool { return 0x0400_00BC <= addr && addr <= 0x0400_00C7 }
func isDMA2IO(addr uint32) bool { return 0x0400_00C8 <= addr && addr <= 0x0400_00D3 }
func isDMA3IO(addr uint32) bool { return 0x0400_00D4 <= addr && addr <= 0x0400_00DF }

func (d *DMA) set(ofs uint32, b byte) bool {
	switch {
	case ofs < 4:
		d.src = (d.src & util.Mask[ofs]) | uint32(b<<(8*ofs))
		return false
	case ofs < 8:
		d.dst = (d.dst & util.Mask[ofs-4]) | uint32(b<<(8*(ofs-4)))
		return false
	case ofs < 12:
		old := d.cnt
		d.cnt = (d.cnt & util.Mask[ofs-8]) | uint32(b<<(8*(ofs-8)))
		return !util.Bit(old, 16+15) && d.enabled() && d.timing() == dmaImmediate
	}
	return false
}

func (d *DMA) dstCnt() int64 {
	switch (d.cnt >> (16 + 5)) & 0b11 {
	case 0, 3:
		return int64(d.size()) / 8
	case 1:
		return -int64(d.size()) / 8
	default:
		return 0
	}
}
func (d *DMA) srcCnt() int64 {
	switch (d.cnt >> (16 + 7)) & 0b11 {
	case 0:
		return int64(d.size()) / 8
	case 1:
		return -int64(d.size()) / 8
	default:
		return 0
	}
}
func (d *DMA) repeat() bool { return util.Bit(d.cnt, 16+9) }
func (d *DMA) size() int {
	if util.Bit(d.cnt, 16+10) {
		return 32
	}
	return 16
}
func (d *DMA) timing() dmaTiming { return dmaTiming((d.cnt >> (16 + 12)) & 0b11) }
func (d *DMA) irq() bool         { return util.Bit(d.cnt, 16+14) }
func (d *DMA) enabled() bool     { return util.Bit(d.cnt, 16+15) }
func (d *DMA) disable() {
	d.cnt &= 0x7fff_ffff
}
func (d *DMA) wordCount(i int) int {
	wordCount := d.cnt & 0xffff
	if wordCount == 0 {
		wordCount = 0x4000
		if i == 3 {
			wordCount = 0x10000
		}
	}
	return int(wordCount)
}

func (g *GBA) dmaTransfer(t dmaTiming) {
	for i, ch := range g.dma {
		if !ch.enabled() {
			continue
		}
		if ch.timing() != t {
			continue
		}

		fmt.Printf("DMA%d start", i)
		g.timer(2)

		wc := ch.wordCount(i)
		size := ch.size()
		for wc > 0 {
			switch size {
			case 16:
				g.setRAM16(ch.dst, g.getRAM16(ch.src, true), true)
			case 32:
				g.setRAM32(ch.dst, g.getRAM32(ch.src, true), true)
			}

			ch.dst = uint32(int64(ch.dst) + ch.dstCnt())
			ch.src = uint32(int64(ch.src) + ch.srcCnt())

			wc--
		}

		if !ch.repeat() {
			ch.disable()
		}
		if ch.irq() {
			g.triggerIRQ(irqDMA0 + i)
		}
	}
}
