package gba

import (
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
	io [12]byte
}

func NewDMA() [4]*DMA       { return [4]*DMA{&DMA{}, &DMA{}, &DMA{}, &DMA{}} }
func (ch *DMA) src() uint32 { return util.LE32(ch.io[:]) }
func (ch *DMA) dst() uint32 { return util.LE32(ch.io[4:]) }
func (ch *DMA) cnt() uint32 { return util.LE32(ch.io[8:]) }
func (ch *DMA) setSrc(v uint32) {
	ch.io[0], ch.io[1], ch.io[2], ch.io[3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
}
func (ch *DMA) setCnt(v uint32) {
	ch.io[8], ch.io[9], ch.io[10], ch.io[11] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
}
func isDMA0IO(addr uint32) bool { return 0x0400_00B0 <= addr && addr <= 0x0400_00BB }
func isDMA1IO(addr uint32) bool { return 0x0400_00BC <= addr && addr <= 0x0400_00C7 }
func isDMA2IO(addr uint32) bool { return 0x0400_00C8 <= addr && addr <= 0x0400_00D3 }
func isDMA3IO(addr uint32) bool { return 0x0400_00D4 <= addr && addr <= 0x0400_00DF }

func (ch *DMA) get(ofs uint32) uint32 {
	return util.LE32(ch.io[ofs:])
}
func (ch *DMA) set(ofs uint32, b byte) bool {
	old := byte(ch.cnt() >> 24)
	ch.io[ofs] = b
	if ofs == 11 {
		return !util.Bit(old, 7) && util.Bit(b, 7) && (ch.timing() == 0)
	}
	return false
}

func (ch *DMA) dstCnt() (int64, bool) {
	switch (ch.cnt() >> (16 + 5)) & 0b11 {
	case 0:
		return int64(ch.size()) / 8, false
	case 1:
		return -int64(ch.size()) / 8, false
	case 3:
		return int64(ch.size()) / 8, true
	default:
		return 0, false
	}
}
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
func (ch *DMA) disable() {
	ch.setCnt(ch.cnt() & 0x7fff_ffff)
}
func (ch *DMA) wordCount(i int) int {
	wordCount := ch.cnt() & 0xffff
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
		if !ch.enabled() || ch.timing() != t {
			continue
		}

		g.timer(2)

		wc, size := ch.wordCount(i), ch.size()
		src, dst := ch.src(), ch.dst()
		srcInc := ch.srcCnt()
		dstInc, _ := ch.dstCnt()
		for wc > 0 {
			switch size {
			case 16:
				g.setRAM16(dst, g.getRAM16(src, true), true)
			case 32:
				g.setRAM32(dst, g.getRAM32(src, true), true)
			}

			dst, src = uint32(int64(dst)+dstInc), uint32(int64(src)+srcInc)
			wc--
		}

		if ch.irq() {
			g.triggerIRQ(IRQID(irqDMA0 + i))
		}

		if !ch.repeat() {
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
	src, dst, cnt := g.dma[ch].src(), g.dma[ch].dst(), g.dma[ch].cnt()
	for i := 0; i < 4; i++ {
		val := g.getRAM32(src, true)
		g.setRAM32(dst, val, true)

		if ch == 1 {
			g.fifoACopy(val)
		} else {
			g.fifoBCopy(val)
		}

		switch (cnt >> (16 + 7)) & 0b11 {
		case 0:
			src += 4
		case 1:
			src -= 4
		}
	}

	if g.dma[ch].irq() {
		g.triggerIRQ(IRQID(irqDMA0 + ch))
	}
}
