package gba

import (
	"magia/pkg/util"
)

type dmaTiming uint32

const (
	dmaImmediate dmaTiming = iota
	dmaVBlank
	dmaHBlank
	dmaSpecial
)

type DMA struct {
	io                  [12]byte
	src, dst            uint32
	count, defaultCount int
}

func NewDMA() [4]*DMA {
	return [4]*DMA{{defaultCount: 0x4000}, {defaultCount: 0x4000}, {defaultCount: 0x4000}, {defaultCount: 0x10000}}
}
func (ch *DMA) cnt() uint32 { return util.LE32(ch.io[8:]) }
func (ch *DMA) setCnt(v uint32) {
	ch.io[8], ch.io[9], ch.io[10], ch.io[11] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
}
func isDMA0IO(addr uint32) bool { return 0x0400_00B0 <= addr && addr <= 0x0400_00BB }
func isDMA1IO(addr uint32) bool { return 0x0400_00BC <= addr && addr <= 0x0400_00C7 }
func isDMA2IO(addr uint32) bool { return 0x0400_00C8 <= addr && addr <= 0x0400_00D3 }
func isDMA3IO(addr uint32) bool { return 0x0400_00D4 <= addr && addr <= 0x0400_00DF }

func (ch *DMA) get(ofs uint32) uint32 { return util.LE32(ch.io[ofs:]) }
func (ch *DMA) set(ofs uint32, b byte) bool {
	old := byte(ch.cnt() >> 24)
	ch.io[ofs] = b
	if ofs == 11 {
		turnon := !util.Bit(old, 7) && util.Bit(b, 7)
		if turnon {
			ch.src, ch.dst = util.LE32(ch.io[0:]), util.LE32(ch.io[4:])
			ch.count = ch.wordCount()
			switch ch.size() {
			case 32:
				ch.src &= ^uint32(3)
				ch.dst &= ^uint32(3)
			case 16:
				ch.src &= ^uint32(1)
				ch.dst &= ^uint32(1)
			}
		}

		return turnon && ch.timing() == 0
	}
	return false
}

func (ch *DMA) dstCnt() int64 {
	switch (ch.cnt() >> (16 + 5)) & 0b11 {
	case 0:
		return int64(ch.size()) / 8
	case 1:
		return -int64(ch.size()) / 8
	case 3:
		return int64(ch.size()) / 8
	default:
		return 0
	}
}
func (ch *DMA) dstReload() bool { return (ch.cnt()>>(16+5))&0b11 == 3 }
func (ch *DMA) srcCnt() int64 {
	switch (ch.cnt() >> (16 + 7)) & 0b11 {
	case 0:
		return int64(ch.size()) / 8
	case 1:
		return -int64(ch.size()) / 8
	default:
		return 0
	}
}
func (ch *DMA) repeat() bool { return util.Bit(ch.cnt(), 16+9) }
func (ch *DMA) size() int {
	if util.Bit(ch.cnt(), 16+10) {
		return 32
	}
	return 16
}
func (ch *DMA) timing() dmaTiming { return dmaTiming((ch.cnt() >> (16 + 12)) & 0b11) }
func (ch *DMA) irq() bool         { return util.Bit(ch.cnt(), 16+14) }
func (ch *DMA) enabled() bool     { return util.Bit(ch.cnt(), 16+15) }
func (ch *DMA) disable()          { ch.setCnt(ch.cnt() & 0x7fff_ffff) }
func (ch *DMA) wordCount() int {
	wordCount := ch.cnt() & 0xffff
	if wordCount == 0 {
		return ch.defaultCount
	}
	return int(wordCount)
}

func (g *GBA) dmaTransfer(t dmaTiming) {
	for i, ch := range g.dma {
		if !ch.enabled() || ch.timing() != t {
			continue
		}

		size := ch.size()
		srcInc, dstInc := ch.srcCnt(), ch.dstCnt()
		for ch.count > 0 {
			switch size {
			case 16:
				g.setRAM16(ch.dst, uint16(g.getRAM16(ch.src, true)), true)
			case 32:
				g.setRAM32(ch.dst, g.getRAM32(ch.src, true), true)
			}

			ch.dst, ch.src = uint32(int64(ch.dst)+dstInc), uint32(int64(ch.src)+srcInc)
			ch.count--
		}

		if ch.irq() {
			g.triggerIRQ(IRQID(irqDMA0 + i))
		}

		if ch.repeat() {
			ch.count = ch.wordCount()
			if ch.dstReload() {
				ch.dst = util.LE32(ch.io[4:])
			}
		} else {
			ch.disable()
		}
	}
}

// Receive 4 x 32bit (16 bytes) per DMA
func (g *GBA) dmaTransferFifo(ch int) {
	if !g.isSoundMasterEnable() || !g.dma[ch].enabled() || g.dma[ch].timing() != dmaSpecial {
		return
	}

	// 32bit × 4 = 4 words
	cnt := g.dma[ch].cnt()
	for i := 0; i < 4; i++ {
		val := g.getRAM32(g.dma[ch].src, true)
		g.setRAM32(g.dma[ch].dst, val, true)

		if ch == 1 {
			g.fifoACopy(val)
		} else {
			g.fifoBCopy(val)
		}

		switch (cnt >> (16 + 7)) & 0b11 {
		case 0:
			g.dma[ch].src += 4
		case 1:
			g.dma[ch].src -= 4
		}
	}

	if g.dma[ch].irq() {
		g.triggerIRQ(IRQID(irqDMA0 + ch))
	}
}
