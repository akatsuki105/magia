package timer

import "github.com/pokemium/magia/pkg/gba/ram"

func (ts *Timers) Set8(addr uint32, val byte) {
	ofs := ram.IOOffset(addr)

	val16 := uint16(0)
	if ofs&1 == 1 {
		val16 = uint16(val) << 8
		val16 |= uint16(ts.ram.IO[ofs-1])
	} else {
		val16 = uint16(val)
		val16 |= uint16(ts.ram.IO[ofs+1]) << 8
	}

	ts.Set16(addr&0xffff_fffe, val16)
}

func (ts *Timers) Set16(addr uint32, val uint16) {
	switch addr {
	case ram.TM0CNT:
		ts.WriteTMCNTL(0, val)
		return
	case ram.TM1CNT:
		ts.WriteTMCNTL(1, val)
		return
	case ram.TM2CNT:
		ts.WriteTMCNTL(2, val)
		return
	case ram.TM3CNT:
		ts.WriteTMCNTL(3, val)
		return

	case ram.TM0CNT + 2:
		val &= 0x00c7
		ts.WriteTMCNTH(0, val)
	case ram.TM1CNT + 2:
		val &= 0x00c7
		ts.WriteTMCNTH(1, val)
	case ram.TM2CNT + 2:
		val &= 0x00c7
		ts.WriteTMCNTH(2, val)
	case ram.TM3CNT + 2:
		val &= 0x00c7
		ts.WriteTMCNTH(3, val)

	default:
		return
	}

	ofs := ram.IOOffset(addr)
	ts.ram.IO[ofs] = byte(val)
	ts.ram.IO[ofs+1] = byte(val >> 8)
}

func (ts *Timers) Set32(addr uint32, val uint32) {
	ts.Set16(addr, uint16(val))
	ts.Set16(addr+2, uint16(val>>16))
}
