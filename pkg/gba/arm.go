package gba

import (
	"fmt"
	"mettaur/pkg/util"
)

const (
	lsl = 0
	lsr = 1
	asr = 2
	ror = 3
)

func (g *GBA) armStep() {
	inst := g.armFetch()
	g.armExec(inst)
}

func (g *GBA) armFetch() uint32 {
	pc := g.R[15]
	if g.lastAddr+4 == pc {
		// sequential
	} else {
		// non-sequential
	}
	return g.RAM.Get(pc)
}

func (g *GBA) armExec(inst uint32) {
	cond := Cond(inst >> 28)
	if g.Check(cond) {
		switch {
		case IsSWI(inst):
			g.armSWI(inst)
		case IsBranch(inst) || IsBX(inst):
			g.armBranch(inst)
		case IsStack(inst):
			if util.Bit(inst, 20) {
				g.armLDM(inst)
			} else {
				g.armSTM(inst)
			}
		case IsLDR(inst):
			g.armLDR(inst)
		case IsSTR(inst):
			g.armSTR(inst)
		case IsLDRH(inst):
			g.armLDRH(inst)
		case IsLDRSB(inst):
			g.armLDRSB(inst)
		case IsLDRSH(inst):
			g.armLDRSH(inst)
		case IsSTRH(inst):
			g.armSTRH(inst)
		case IsMRS(inst):
			g.armMRS(inst)
		case IsMSR(inst):
			g.armMSR(inst)
		case IsMPY(inst):
			g.armMPY(inst)
		case IsALU(inst):
			g.armALU(inst)
		}
	} else {
		g.timer(g.cycleS(g.R[15]))
	}
	g.R[15] += 4
}

func (g *GBA) armSWI(inst uint32) {
	nn := inst & 0b1111_1111 // ignore 23-8bit on GBA
	fmt.Println(nn)
}

func (g *GBA) armBranch(inst uint32) {
	if IsBX(inst) {
		g.armBX(inst)
	}
	if util.Bit(inst, 24) {
		g.armBL(inst)
	} else {
		g.armB(inst)
	}
}

func (g *GBA) armB(inst uint32) {
	nn := inst & 0b1111_1111_1111_1111_1111_1111
	g.R[15] = g.R[15] + 8 + nn*4
	g.timer(2*g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
}

func (g *GBA) armBL(inst uint32) {
	nn := inst & 0b1111_1111_1111_1111_1111_1111
	g.R[14] = g.R[15] + 4
	g.R[15] = g.R[15] + 8 + nn*4
	g.timer(2*g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
}

func (g *GBA) armBX(inst uint32) {
	rn := g.R[inst&0b1111]
	if util.Bit(rn, 0) {
		g.SetCPSRFlag(flagT, true)
		g.R[15] = rn - 1
		return
	}
	g.R[15] = rn
}

func (g *GBA) armLDM(inst uint32) {
	p, u := util.Bit(inst, 24), util.Bit(inst, 23)
	rn := inst >> 16 & 0b1111
	switch {
	case p && u: // IB
		for rs := 0; rs < 15; rs++ {
			if util.Bit(inst, rs) {
				g.R[rn] += 4
				g.R[rn] = g.RAM.Get(g.R[rs])
			}
		}
	case !p && u: // IA, pop
		for rs := 0; rs < 15; rs++ {
			if util.Bit(inst, rs) {
				g.R[rn] = g.RAM.Get(g.R[rs])
				g.R[rn] += 4
			}
		}
	case p && !u: // DB, push
		for rs := 0; rs < 15; rs++ {
			if util.Bit(inst, rs) {
				g.R[rn] -= 4
				g.R[rn] = g.RAM.Get(g.R[rs])
			}
		}
	case !p && !u: // DA
		for rs := 0; rs < 15; rs++ {
			if util.Bit(inst, rs) {
				g.R[rn] = g.RAM.Get(g.R[rs])
				g.R[rn] -= 4
			}
		}
	}
}

func (g *GBA) armSTM(inst uint32) {
	p, u := util.Bit(inst, 24), util.Bit(inst, 23)
	rn := inst >> 16 & 0b1111
	switch {
	case p && u: // IB
		for rs := 0; rs < 15; rs++ {
			if util.Bit(inst, rs) {
				g.R[rn] += 4
				g.RAM.Set32(g.R[rn], g.R[rs])
			}
		}
	case !p && u: // IA
		for rs := 0; rs < 15; rs++ {
			if util.Bit(inst, rs) {
				g.RAM.Set32(g.R[rn], g.R[rs])
				g.R[rn] += 4
			}
		}
	case p && !u: // DB, push
		for rs := 0; rs < 15; rs++ {
			if util.Bit(inst, rs) {
				g.R[rn] -= 4
				g.RAM.Set32(g.R[rn], g.R[rs])
			}
		}
	case !p && !u: // DA
		for rs := 0; rs < 15; rs++ {
			if util.Bit(inst, rs) {
				g.RAM.Set32(g.R[rn], g.R[rs])
				g.R[rn] -= 4
			}
		}
	}
}

func (g *GBA) armRegShiftOffset(inst uint32) uint32 {
	ofs := uint32(0)
	if util.Bit(inst, 25) {
		is := inst >> 7 & 0b11111 // I = 1
		shiftType := inst >> 5 & 0b11
		rm := inst & 0b1111
		switch shiftType {
		case lsl:
			ofs = g.armLSL(g.R[rm], is)
		case lsr:
			ofs = g.armLSR(g.R[rm], is)
		case asr:
			ofs = g.armASR(g.R[rm], is)
		case ror:
			ofs = g.armROR(g.R[rm], is)
		}
	} else {
		ofs = inst & 0b1111_1111_1111 // I = 0
	}
	return ofs
}

func (g *GBA) armLDR(inst uint32) {
	pre, plus, byteUnit := util.Bit(inst, 24), util.Bit(inst, 23), util.Bit(inst, 22)
	rn, rd := inst>>16&0b1111, inst>>12&0b1111
	ofs := g.armRegShiftOffset(inst)

	addr := g.R[rn]
	if pre {
		if plus {
			addr += ofs
		} else {
			addr -= ofs
		}
	}
	g.R[rd] = g.RAM.Get(addr)
	if byteUnit {
		g.R[rd] = g.R[rd] & 0xff
	}
	if !pre {
		if writeBack := util.Bit(inst, 21); writeBack {
			if plus {
				addr += ofs
			} else {
				addr -= ofs
			}
			g.R[rn] = addr
		}
	}
	g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
}

func (g *GBA) armSTR(inst uint32) {
	pre, plus, byteUnit := util.Bit(inst, 24), util.Bit(inst, 23), util.Bit(inst, 22)
	rn, rd := inst>>16&0b1111, inst>>12&0b1111
	ofs := g.armRegShiftOffset(inst)

	addr := g.R[rn]
	if pre {
		if plus {
			addr += ofs
		} else {
			addr -= ofs
		}
	}
	g.RAM.Set32(addr, g.R[rd])
	if byteUnit {
		g.R[rd] = g.R[rd] & 0xff
	}
	if !pre {
		if writeBack := util.Bit(inst, 21); writeBack {
			if plus {
				addr += ofs
			} else {
				addr -= ofs
			}
			g.R[rn] = addr
		}
	}
	g.timer(2 * g.cycleN(g.R[15]))
}

func (g *GBA) armALUOp2(inst uint32) uint32 {
	if util.Bit(inst, 25) {
		// immediate
		is := inst >> 7 & 0b1111
		if isRegister := inst >> 4 & 0b1; util.ToBool(isRegister) {
			g.timer(1)
			is = g.R[inst>>8&0b1111]
		}

		rm := inst & 0b1111
		switch shiftType := inst >> 5 & 0b11; shiftType {
		case lsl:
			return g.armLSL(g.R[rm], is)
		case lsr:
			return g.armLSR(g.R[rm], is)
		case asr:
			return g.armASR(g.R[rm], is)
		case ror:
			return g.armROR(g.R[rm], is)
		}
	}

	// register
	op2 := inst & 0b1111_1111
	is := uint(inst >> 8 & 0b1111)
	if is == 0 {
		is = 1
	}
	util.ROR(op2, is)
	return op2
}

func (g *GBA) armALU(inst uint32) {
	switch opcode := inst >> 21 & 0b1111; opcode {
	case 0x0:
		g.armAND(inst)
	case 0x1:
		g.armEOR(inst)
	case 0x2:
		g.armSUB(inst)
	case 0x3:
		g.armRSB(inst)
	case 0x4:
		g.armADD(inst)
	case 0x5:
		g.armADC(inst)
	case 0x6:
		g.armSBC(inst)
	case 0x7:
		g.armRSC(inst)
	case 0x8:
		g.armTST(inst)
	case 0x9:
		g.armTEQ(inst)
	case 0xa:
		g.armCMP(inst)
	case 0xb:
		g.armCMN(inst)
	case 0xc:
		g.armORR(inst)
	case 0xd:
		g.armMOV(inst)
	case 0xe:
		g.armBIC(inst)
	case 0xf:
		g.armMVN(inst)
	}

	g.timer(g.cycleS(g.R[15]))
	if rd := inst >> 12 & 0b1111; rd == 15 {
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
	}
}

func (g *GBA) armAND(inst uint32) {
	rd, rn, op2 := inst>>12&0b1111, inst>>16&0b1111, g.armALUOp2(inst)
	g.R[rd] = g.R[rn] & op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armEOR(inst uint32) {
	rd, rn, op2 := inst>>12&0b1111, inst>>16&0b1111, g.armALUOp2(inst)
	g.R[rd] = g.R[rn] ^ op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armSUB(inst uint32) {
	rd, rn, op2 := inst>>12&0b1111, inst>>16&0b1111, g.armALUOp2(inst)
	g.R[rd] = g.R[rn] - op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		v := (g.R[rn] ^ op2) & (g.R[rn] ^ g.R[rd]) & 0x8000_0000
		g.SetCPSRFlag(flagV, util.ToBool(v))
	}
}

func (g *GBA) armRSB(inst uint32) {
	rd, rn, op2 := inst>>12&0b1111, inst>>16&0b1111, g.armALUOp2(inst)
	g.R[rd] = op2 - g.R[rn]
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		v := (op2 ^ g.R[rn]) & (op2 ^ g.R[rd]) & 0x8000_0000
		g.SetCPSRFlag(flagV, util.ToBool(v))
	}
}

func (g *GBA) armADD(inst uint32) {
	rd, rn, op2 := inst>>12&0b1111, inst>>16&0b1111, g.armALUOp2(inst)
	g.R[rd] = g.R[rn] + op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		v := ^(g.R[rn] ^ op2) & (g.R[rn] ^ g.R[rd]) & 0x8000_0000
		g.SetCPSRFlag(flagV, util.ToBool(v))
	}
}

func (g *GBA) armADC(inst uint32) {
	rd, rn, op2 := inst>>12&0b1111, inst>>16&0b1111, g.armALUOp2(inst)
	carry := uint32(0)
	if g.GetCPSRFlag(flagC) {
		carry = 1
	}
	g.R[rd] = g.R[rn] + op2 + carry
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		v := ^(g.R[rn] ^ op2) & (g.R[rn] ^ g.R[rd]) & 0x8000_0000
		g.SetCPSRFlag(flagV, util.ToBool(v))
	}
}

func (g *GBA) armSBC(inst uint32) {
	rd, rn, op2 := inst>>12&0b1111, inst>>16&0b1111, g.armALUOp2(inst)
	carry := uint32(0)
	if g.GetCPSRFlag(flagC) {
		carry = 1
	}
	g.R[rd] = g.R[rn] - op2 + carry - 1
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		v := (g.R[rn] ^ op2) & (g.R[rn] ^ g.R[rd]) & 0x8000_0000
		g.SetCPSRFlag(flagV, util.ToBool(v))
	}
}

func (g *GBA) armRSC(inst uint32) {
	rd, rn, op2 := inst>>12&0b1111, inst>>16&0b1111, g.armALUOp2(inst)
	carry := uint32(0)
	if g.GetCPSRFlag(flagC) {
		carry = 1
	}
	g.R[rd] = op2 - g.R[rn] + carry - 1
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		v := (op2 ^ g.R[rn]) & (op2 ^ g.R[rd]) & 0x8000_0000
		g.SetCPSRFlag(flagV, util.ToBool(v))
	}
}

func (g *GBA) armTST(inst uint32) {
	rn, op2 := inst>>16&0b1111, g.armALUOp2(inst)
	result := g.R[rn] & op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
	}
}

func (g *GBA) armTEQ(inst uint32) {
	rn, op2 := inst>>16&0b1111, g.armALUOp2(inst)
	result := g.R[rn] ^ op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
	}
}

func (g *GBA) armCMP(inst uint32) {
	rn, op2 := inst>>16&0b1111, g.armALUOp2(inst)
	result := g.R[rn] - op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
		v := (g.R[rn] ^ op2) & (g.R[rn] ^ result) & 0x8000_0000
		g.SetCPSRFlag(flagV, util.ToBool(v))
	}
}

func (g *GBA) armCMN(inst uint32) {
	rn, op2 := inst>>16&0b1111, g.armALUOp2(inst)
	result := g.R[rn] + op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
		v := ^(g.R[rn] ^ op2) & (g.R[rn] ^ result) & 0x8000_0000
		g.SetCPSRFlag(flagV, util.ToBool(v))
	}
}

func (g *GBA) armORR(inst uint32) {
	rd, rn, op2 := inst>>12&0b1111, inst>>16&0b1111, g.armALUOp2(inst)
	g.R[rd] = g.R[rn] | op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armMOV(inst uint32) {
	rd, op2 := inst>>12&0b1111, g.armALUOp2(inst)
	g.R[rd] = op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armBIC(inst uint32) {
	rd, rn, op2 := inst>>12&0b1111, inst>>16&0b1111, g.armALUOp2(inst)
	g.R[rd] = g.R[rn] & ^op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armMVN(inst uint32) {
	rd, op2 := inst>>12&0b1111, g.armALUOp2(inst)
	g.R[rd] = ^op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armMPY(inst uint32) {
	switch inst >> 21 & 0b1111 {
	case 0:
		g.armMUL(inst)
	case 1:
		g.armMLA(inst)
	case 4:
		// umull
		g.armUMULL(inst)
	case 5:
		// umlal
		g.armUMLAL(inst)
	case 6:
		// smull
		g.armSMULL(inst)
	case 7:
		// smlal
		g.armSMLAL(inst)
	}
}

// Rd=Rm*Rs
func (g *GBA) armMUL(inst uint32) {
	rd, rs := inst>>16&0b1111, inst>>8&0b1111
	g.R[rd] *= g.R[rs]
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}

	g.timer(g.cycleS(g.R[15]))
	switch {
	case g.R[rs]&0xfff0 == 0xfff0:
		g.timer(1)
	case g.R[rs]&0xff00 == 0xff00:
		g.timer(2)
	case g.R[rs]&0xf000 == 0xf000:
		g.timer(3)
	default:
		g.timer(4)
	}
}

// Rd=Rm*Rs+Rn
func (g *GBA) armMLA(inst uint32) {
	rd, rn, rs := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111
	g.R[rd] = g.R[rd]*g.R[rs] + g.R[rn]
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
	g.timer(g.cycleS(g.R[15]) + 1)
	switch {
	case g.R[rs]&0xfff0 == 0xfff0:
		g.timer(1)
	case g.R[rs]&0xff00 == 0xff00:
		g.timer(2)
	case g.R[rs]&0xf000 == 0xf000:
		g.timer(3)
	default:
		g.timer(4)
	}
}

// RdHiLo=Rm*Rs
func (g *GBA) armUMULL(inst uint32) {
	rdHi, rdLo, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
	result := uint64(g.R[rs]) * uint64(g.R[rm])
	g.R[rdHi], g.R[rdLo] = uint32(result>>32), uint32(result)
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rdHi], 31))
	}
	g.timer(g.cycleS(g.R[15]) + 1)
	switch {
	case g.R[rs]&0xfff0 == 0xfff0:
		g.timer(1)
	case g.R[rs]&0xff00 == 0xff00:
		g.timer(2)
	case g.R[rs]&0xf000 == 0xf000:
		g.timer(3)
	default:
		g.timer(4)
	}
}

// RdHiLo=Rm*Rs+RdHiLo
func (g *GBA) armUMLAL(inst uint32) {
	rdHi, rdLo, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
	result := uint64(g.R[rs])*uint64(g.R[rm]) + uint64(g.R[rdHi])<<32 | uint64(g.R[rdLo])
	g.R[rdHi], g.R[rdLo] = uint32(result>>32), uint32(result)
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rdHi], 31))
	}
	g.timer(g.cycleS(g.R[15]) + 2)
	switch {
	case g.R[rs]&0xfff0 == 0xfff0:
		g.timer(1)
	case g.R[rs]&0xff00 == 0xff00:
		g.timer(2)
	case g.R[rs]&0xf000 == 0xf000:
		g.timer(3)
	default:
		g.timer(4)
	}
}

// RdHiLo=Rm*Rs
func (g *GBA) armSMULL(inst uint32) {
	rdHi, rdLo, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
	result := int64(int32(g.R[rs])) * int64(int32(g.R[rm]))
	g.R[rdHi], g.R[rdLo] = uint32(result>>32), uint32(result)
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rdHi], 31))
	}
	g.timer(g.cycleS(g.R[15]) + 1)
	switch {
	case g.R[rs]&0xfff0 == 0xfff0:
		g.timer(1)
	case g.R[rs]&0xff00 == 0xff00:
		g.timer(2)
	case g.R[rs]&0xf000 == 0xf000:
		g.timer(3)
	default:
		g.timer(4)
	}
}

// RdHiLo=Rm*Rs+RdHiLo
func (g *GBA) armSMLAL(inst uint32) {
	rdHi, rdLo, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
	result := int64(int32(g.R[rs]))*int64(int32(g.R[rm])) + int64(g.R[rdHi])<<32 | int64(g.R[rdLo])
	g.R[rdHi], g.R[rdLo] = uint32(result>>32), uint32(result)
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rdHi], 31))
	}
	g.timer(g.cycleS(g.R[15]) + 2)
	switch {
	case g.R[rs]&0xfff0 == 0xfff0:
		g.timer(1)
	case g.R[rs]&0xff00 == 0xff00:
		g.timer(2)
	case g.R[rs]&0xf000 == 0xf000:
		g.timer(3)
	default:
		g.timer(4)
	}
}

func (g *GBA) armLDRH(inst uint32) {
	ofs := (((inst >> 8) & 0b1111) << 4) | (inst & 0b1111) // immediate
	if !util.Bit(inst, 22) {
		// register
		rm := inst & 0b1111
		ofs = g.R[rm]
	}

	rn, rd := (inst>>16)&0b1111, (inst>>12)&0b1111
	addr := g.R[rn]
	pre := util.Bit(inst, 24)
	if pre {
		if plus := util.Bit(inst, 23); plus {
			addr += ofs
		} else {
			addr -= ofs
		}
		if writeBack := util.Bit(inst, 21); writeBack {
			g.R[rn] = addr
		}
	}
	g.R[rd] = uint32(uint16(g.RAM.Get(addr)))
	g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	if rd == 15 {
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
	}
}

func (g *GBA) armLDRSB(inst uint32) {
	ofs := (((inst >> 8) & 0b1111) << 4) | (inst & 0b1111) // immediate
	if !util.Bit(inst, 22) {
		// register
		rm := inst & 0b1111
		ofs = g.R[rm]
	}

	rn, rd := (inst>>16)&0b1111, (inst>>12)&0b1111
	addr := g.R[rn]
	pre := util.Bit(inst, 24)
	if pre {
		if plus := util.Bit(inst, 23); plus {
			addr += ofs
		} else {
			addr -= ofs
		}
		if writeBack := util.Bit(inst, 21); writeBack {
			g.R[rn] = addr
		}
	}
	g.R[rd] = uint32(byte(int32(g.RAM.Get(addr))))
	g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	if rd == 15 {
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
	}
}

func (g *GBA) armLDRSH(inst uint32) {
	ofs := (((inst >> 8) & 0b1111) << 4) | (inst & 0b1111) // immediate
	if !util.Bit(inst, 22) {
		// register
		rm := inst & 0b1111
		ofs = g.R[rm]
	}

	rn, rd := (inst>>16)&0b1111, (inst>>12)&0b1111
	addr := g.R[rn]
	pre := util.Bit(inst, 24)
	if pre {
		if plus := util.Bit(inst, 23); plus {
			addr += ofs
		} else {
			addr -= ofs
		}
		if writeBack := util.Bit(inst, 21); writeBack {
			g.R[rn] = addr
		}
	}
	g.R[rd] = uint32(int16(int32(g.RAM.Get(addr))))
	g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	if rd == 15 {
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
	}
}

func (g *GBA) armSTRH(inst uint32) {
	ofs := (((inst >> 8) & 0b1111) << 4) | (inst & 0b1111) // immediate
	if !util.Bit(inst, 22) {
		// register
		rm := inst & 0b1111
		ofs = g.R[rm]
	}

	rn, rd := (inst>>16)&0b1111, (inst>>12)&0b1111
	addr := g.R[rn]
	pre := util.Bit(inst, 24)
	if pre {
		if plus := util.Bit(inst, 23); plus {
			addr += ofs
		} else {
			addr -= ofs
		}
		if writeBack := util.Bit(inst, 21); writeBack {
			g.R[rn] = addr
		}
	}
	g.RAM.Set16(addr, uint16(g.R[rd]))
	g.timer(2 * g.cycleN(g.R[15]))
}

func (g *GBA) armMRS(inst uint32) {
	useSpsr := util.ToBool(inst >> 22 & 0b1)
	rd := (inst >> 12) & 0b1111
	if useSpsr {
		switch g.GetOSMode() {
		case FIQ:
			g.R[rd] = g.SPSRFiq
		case IRQ:
			g.R[rd] = g.SPSRIrq
		case SWI:
			g.R[rd] = g.SPSRSvc
		case ABT:
			g.R[rd] = g.SPSRAbt
		case UND:
			g.R[rd] = g.SPSRUnd
		}
		return
	}
	g.R[rd] = g.CPSR
	g.timer(g.cycleS(g.R[15]))
}

func (g *GBA) armMSR(inst uint32) {
	useSpsr := util.ToBool(inst >> 22 & 0b1)
	value := uint32(0)
	if util.Bit(inst, 25) {
		// immediate Psr[field] = Rm
		rm := inst & 0b1111
		value = g.R[rm]
	} else {
		// register Psr[field] = Imm
		is, imm := ((inst>>8)&0b1111)*2, inst&0b1111_1111
		value = util.ROR(imm, uint(is))
	}

	var psr *uint32
	if useSpsr {
		switch g.GetOSMode() {
		case FIQ:
			psr = &g.SPSRFiq
		case IRQ:
			psr = &g.SPSRIrq
		case SWI:
			psr = &g.SPSRSvc
		case ABT:
			psr = &g.SPSRAbt
		case UND:
			psr = &g.SPSRUnd
		default:
			psr = &g.CPSR
		}
	} else {
		psr = &g.CPSR
	}

	b0, b1, b2, b3 := (*psr)&0xff, ((*psr)>>8)&0xff, ((*psr)>>16)&0xff, ((*psr)>>24)&0xff
	if f := util.Bit(inst, 19); f {
		b3 = (value >> 24) & 0xff
	}
	if c := util.Bit(inst, 16); c {
		b0 = value & 0xff
	}
	*psr = b3<<24 | b2<<16 | b1<<8 | b0
	g.timer(g.cycleS(g.R[15]))
}
