package gba

import (
	"fmt"
	"mettaur/pkg/util"
	"os"
)

const (
	lsl = 0
	lsr = 1
	asr = 2
	ror = 3
)

func (g *GBA) armStep() {
	g.pipe.inst[1] = Inst{
		inst: g.getRAM32(g.R[15], true),
		loc:  g.R[15],
	}
	g.armExec(g.inst.inst)
	if g.pipe.ok {
		g.pipe.ok = false
		return
	}
	g.R[15] += 4
}

func (g *GBA) armExec(inst uint32) {
	// g.armInst(inst)
	cond := Cond(inst >> 28)
	if g.Check(cond) {
		switch {
		case IsArmSWI(inst):
			g.armSWI(inst)
		case IsArmBranch(inst) || IsArmBX(inst):
			g.armBranch(inst)
		case IsArmStack(inst):
			if util.Bit(inst, 20) {
				g.armLDM(inst)
			} else {
				g.armSTM(inst)
			}
		case IsArmLDR(inst):
			g.armLDR(inst)
		case IsArmSTR(inst):
			g.armSTR(inst)
		case IsArmLDRH(inst):
			g.armLDRH(inst)
		case IsArmLDRSB(inst):
			g.armLDRSB(inst)
		case IsArmLDRSH(inst):
			g.armLDRSH(inst)
		case IsArmSTRH(inst):
			g.armSTRH(inst)
		case IsArmMRS(inst):
			g.armMRS(inst)
		case IsArmMSR(inst):
			g.armMSR(inst)
		case IsArmSWP(inst):
			fmt.Fprintf(os.Stderr, "SWI is unsupported in 0x%04x\n", g.inst.loc)
		case IsArmMPY(inst):
			g.armMPY(inst)
		case IsArmALU(inst):
			g.armALU(inst)
		default:
			fmt.Fprintf(os.Stderr, "invalid opcode(0x%04x) in 0x%04x\n", inst, g.inst.loc)
		}
	}
}

func (g *GBA) armSWI(inst uint32) {
	nn := byte(inst >> 16)
	g.printSWI(nn)
	g.exception(swiVec, SWI)
}

func (g *GBA) armBranch(inst uint32) {
	switch {
	case IsArmBX(inst):
		g.armBX(inst)
	case util.Bit(inst, 24):
		g.armBL(inst)
	default:
		g.armB(inst)
	}
	g.pipelining()
}

func (g *GBA) armB(inst uint32) {
	nn := int32(inst)
	nn <<= 8
	nn >>= 6

	if nn >= 0 {
		g.R[15] = g.inst.loc + 8 + uint32(nn)
	} else {
		g.R[15] = g.inst.loc + 8 - uint32(-nn)
	}
}

func (g *GBA) armBL(inst uint32) {
	nn := int32(inst)
	nn <<= 8
	nn >>= 6
	g.R[14] = g.inst.loc + 4
	if nn >= 0 {
		g.R[15] = g.inst.loc + 8 + uint32(nn)
	} else {
		g.R[15] = g.inst.loc + 8 - uint32(-nn)
	}
}

func (g *GBA) armBX(inst uint32) {
	rnval := g.R[inst&0b1111]
	if util.Bit(rnval, 0) {
		g.SetCPSRFlag(flagT, true)
		g.R[15] = rnval - 1
	} else {
		g.R[15] = rnval
	}
}

func (g *GBA) armLDM(inst uint32) {
	if s := util.Bit(inst, 22); s {
		g._armLDMUsr(inst)
	} else {
		g._armLDM(inst)
	}
}

// LDM with S(bit22)
func (g *GBA) _armLDMUsr(inst uint32) {
	if util.Bit(inst, 15) {
		g._armLDM(inst)
		g.restoreOSMode()
		g.checkIRQ()
	} else {
		mode := g.getOSMode()
		g.setOSMode(USR)
		g._armLDM(inst)
		g.setOSMode(mode)
	}
}

func (g *GBA) _armLDM(inst uint32) {
	p, u := util.Bit(inst, 24), util.Bit(inst, 23)
	rn := inst >> 16 & 0b1111
	rnval := g.R[rn]

	writeBack := (inst>>21)&0b1 == 1
	n := 0
	switch {
	case p && u: // IB
		for rs := 0; rs < 16; rs++ {
			if util.Bit(inst, rs) {
				g.R[rn] += 4
				g.R[rs] = g.getRAM32(g.R[rn], n > 0)
				n++
			}
		}
	case !p && u: // IA, pop
		for rs := 0; rs < 16; rs++ {
			if util.Bit(inst, rs) {
				g.R[rs] = g.getRAM32(g.R[rn], n > 0)
				g.R[rn] += 4
				n++
			}
		}
	case p && !u: // DB
		for rs := 15; rs >= 0; rs-- {
			if util.Bit(inst, rs) {
				g.R[rn] -= 4
				g.R[rs] = g.getRAM32(g.R[rn], n > 0)
				n++
			}
		}
	case !p && !u: // DA
		for rs := 15; rs >= 0; rs-- {
			if util.Bit(inst, rs) {
				g.R[rs] = g.getRAM32(g.R[rn], n > 0)
				g.R[rn] -= 4
				n++
			}
		}
	}

	g.timer(1)
	if util.Bit(inst, 15) {
		g.pipelining()
	}
	if !writeBack {
		g.R[rn] = rnval
	}
}

func (g *GBA) armSTM(inst uint32) {
	if s := util.Bit(inst, 22); s {
		g._armSTMUsr(inst)
	} else {
		g._armSTM(inst)
	}
}

func (g *GBA) _armSTMUsr(inst uint32) {
	mode := g.getOSMode()
	g.setOSMode(USR)
	g._armSTM(inst)
	g.setOSMode(mode)
}

func (g *GBA) _armSTM(inst uint32) {
	p, u := util.Bit(inst, 24), util.Bit(inst, 23)
	rn := (inst >> 16) & 0b1111
	rnval := g.R[rn]

	writeBack := (inst>>21)&0b1 == 1
	n := 0
	switch {
	case p && u: // IB
		for rs := 0; rs < 16; rs++ {
			if util.Bit(inst, rs) {
				g.R[rn] += 4
				g.setRAM32(g.R[rn], g.R[rs], n > 0)
				n++
			}
		}
	case !p && u: // IA
		for rs := 0; rs < 16; rs++ {
			if util.Bit(inst, rs) {
				g.setRAM32(g.R[rn], g.R[rs], n > 0)
				g.R[rn] += 4
				n++
			}
		}
	case p && !u: // DB, push
		for rs := 15; rs >= 0; rs-- {
			if util.Bit(inst, rs) {
				g.R[rn] -= 4
				g.setRAM32(g.R[rn], g.R[rs], n > 0)
				n++
			}
		}
	case !p && !u: // DA
		for rs := 15; rs >= 0; rs-- {
			if util.Bit(inst, rs) {
				g.setRAM32(g.R[rn], g.R[rs], n > 0)
				g.R[rn] -= 4
				n++
			}
		}
	}

	g.timer(g.cycleS2N())
	if !writeBack {
		g.R[rn] = rnval
	}
}

func (g *GBA) armRegShiftOffset(inst uint32) uint32 {
	ofs := uint32(0)
	if util.Bit(inst, 25) {
		is := inst >> 7 & 0b11111 // I = 1 shift reg
		shiftType := inst >> 5 & 0b11
		rm := inst & 0b1111
		switch shiftType {
		case lsl:
			ofs = g.armLSL(g.R[rm], is, false)
		case lsr:
			ofs = g.armLSR(g.R[rm], is, false)
		case asr:
			ofs = g.armASR(g.R[rm], is, false)
		case ror:
			ofs = g.armROR(g.R[rm], is, false)
		}
	} else {
		ofs = inst & 0b1111_1111_1111 // I = 0 immediate
	}
	return ofs
}

func (g *GBA) armLDR(inst uint32) {
	pre, plus, byteUnit := util.Bit(inst, 24), util.Bit(inst, 23), util.Bit(inst, 22)
	rn, rd := (inst>>16)&0b1111, (inst>>12)&0b1111
	ofs := g.armRegShiftOffset(inst)

	addr := g.R[rn]
	if pre {
		if plus {
			addr += ofs
		} else {
			addr -= ofs
		}
	}
	if byteUnit {
		g.R[rd] = uint32(g.getRAM8(addr, false))
	} else {
		g.R[rd] = g.getRAM32(addr, false)
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
	if rd == 15 {
		g.pipelining()
	}
	g.timer(1)
}

func (g *GBA) armSTR(inst uint32) {
	pre, plus, byteUnit := util.Bit(inst, 24), util.Bit(inst, 23), util.Bit(inst, 22)
	rn, rd := (inst>>16)&0b1111, (inst>>12)&0b1111
	ofs := g.armRegShiftOffset(inst)

	addr := g.R[rn]
	if pre {
		if plus {
			addr += ofs
		} else {
			addr -= ofs
		}
	}
	if byteUnit {
		g.setRAM8(addr, byte(g.R[rd]), false)
	} else {
		g.setRAM32(addr, g.R[rd], false)
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
	g.timer(g.cycleS2N())
}

func (g *GBA) armALUOp2(inst uint32) uint32 {
	if !util.Bit(inst, 25) {
		// register
		is := (inst >> 7) & 0b11111
		rm := inst & 0b1111

		salt := uint32(0)
		if isRegister := (inst >> 4) & 0b1; util.ToBool(isRegister) {
			g.timer(1)
			is = g.R[(inst>>8)&0b1111] & 0b1111_1111
			if rm == 15 {
				salt = 4
			}
		}

		carryVariable := (inst>>20)&0b1 == 1
		switch shiftType := (inst >> 5) & 0b11; shiftType {
		case lsl:
			return g.armLSL(g.R[rm]+salt, is, carryVariable)
		case lsr:
			return g.armLSR(g.R[rm]+salt, is, carryVariable)
		case asr:
			return g.armASR(g.R[rm]+salt, is, carryVariable)
		case ror:
			return g.armROR(g.R[rm]+salt, is, carryVariable)
		}
		return g.R[rm] + salt
	}

	// immediate
	op2 := inst & 0b1111_1111
	is := uint((inst>>8)&0b1111) * 2
	op2 = util.ROR(op2, is)
	return op2
}

func (g *GBA) armALURn(inst uint32) uint32 {
	rn := (inst >> 16) & 0b1111
	if rn == 15 {
		if !util.Bit(inst, 25) && util.Bit(inst, 4) {
			return g.inst.loc + 12
		}
		return g.inst.loc + 8
	}
	return g.R[rn]
}

func (g *GBA) armALU(inst uint32) {
	switch opcode := inst >> 21 & 0b1111; opcode {
	case 0x0:
		g.armAND(inst)
	case 0x1:
		g.armEOR(inst)
	case 0x2:
		g.armSUB(inst) // arith
	case 0x3:
		g.armRSB(inst) // arith
	case 0x4:
		g.armADD(inst) // arith
	case 0x5:
		g.armADC(inst) // arith
	case 0x6:
		g.armSBC(inst) // arith
	case 0x7:
		g.armRSC(inst) // arith
	case 0x8:
		g.armTST(inst)
	case 0x9:
		g.armTEQ(inst)
	case 0xa:
		g.armCMP(inst) // arith
	case 0xb:
		g.armCMN(inst) // arith
	case 0xc:
		g.armORR(inst)
	case 0xd:
		g.armMOV(inst)
	case 0xe:
		g.armBIC(inst)
	case 0xf:
		g.armMVN(inst)
	}
}

func (g *GBA) armALUChangeOSMode() {
	g.restoreOSMode()
	g.pipelining()
	g.checkIRQ()
}

func (g *GBA) armAND(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = rnval & op2

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armEOR(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = rnval ^ op2

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armSUB(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = rnval - op2

	s := (inst>>20)&0b1 == 1
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		g.SetCPSRFlag(flagC, util.SubC(uint64(rnval)-uint64(op2)))
		g.SetCPSRFlag(flagV, util.SubV(rnval, op2, g.R[rd]))
	}
}

func (g *GBA) armRSB(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = op2 - rnval

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		g.SetCPSRFlag(flagC, util.SubC(uint64(op2)-uint64(rnval)))
		g.SetCPSRFlag(flagV, util.SubV(op2, rnval, g.R[rd]))
	}
}

func (g *GBA) armADD(inst uint32) {
	rd, rnval, op2 := (inst>>12)&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = rnval + op2

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		g.SetCPSRFlag(flagC, util.AddC(uint64(rnval)+uint64(op2)))
		g.SetCPSRFlag(flagV, util.AddV(rnval, op2, g.R[rd]))
	}
}

func (g *GBA) armADC(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	carry := uint32(0)
	if g.GetCPSRFlag(flagC) {
		carry = 1
	}
	g.R[rd] = rnval + op2 + carry

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		g.SetCPSRFlag(flagC, util.AddC(uint64(rnval)+uint64(op2)+uint64(carry)))
		g.SetCPSRFlag(flagV, util.AddV(rnval, op2, g.R[rd]))
	}
}

func (g *GBA) armSBC(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	carry := uint32(0)
	if g.GetCPSRFlag(flagC) {
		carry = 1
	}
	g.R[rd] = rnval - op2 + carry - 1

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		g.SetCPSRFlag(flagC, util.SubC(uint64(rnval)-uint64(op2)+uint64(carry)-uint64(1)))
		g.SetCPSRFlag(flagV, util.SubV(rnval, op2, g.R[rd]))
	}
}

func (g *GBA) armRSC(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	carry := uint32(0)
	if g.GetCPSRFlag(flagC) {
		carry = 1
	}
	g.R[rd] = op2 - rnval + carry - 1

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
		g.SetCPSRFlag(flagC, util.SubC(uint64(op2)-uint64(rnval)+uint64(carry)-uint64(1)))
		g.SetCPSRFlag(flagV, util.SubV(op2, rnval, g.R[rd]))
	}
}

func (g *GBA) armTST(inst uint32) {
	rnval, op2 := g.armALURn(inst), g.armALUOp2(inst)
	result := rnval & op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
	}
}

func (g *GBA) armTEQ(inst uint32) {
	rnval, op2 := g.armALURn(inst), g.armALUOp2(inst)
	result := rnval ^ op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
	}
}

func (g *GBA) armCMP(inst uint32) {
	rnval, op2 := g.armALURn(inst), g.armALUOp2(inst)
	result := rnval - op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
		g.SetCPSRFlag(flagC, util.SubC(uint64(rnval)-uint64(op2)))
		g.SetCPSRFlag(flagV, util.SubV(rnval, op2, result))
	}
}

func (g *GBA) armCMN(inst uint32) {
	rnval, op2 := g.armALURn(inst), g.armALUOp2(inst)
	result := rnval + op2
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
		g.SetCPSRFlag(flagC, util.AddC(uint64(rnval)+uint64(op2)))
		g.SetCPSRFlag(flagV, util.AddV(rnval, op2, result))
	}
}

func (g *GBA) armORR(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = rnval | op2

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armMOV(inst uint32) {
	rd, op2 := (inst>>12)&0b1111, g.armALUOp2(inst)
	g.R[rd] = op2

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armBIC(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = rnval & ^op2

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}
}

func (g *GBA) armMVN(inst uint32) {
	rd, op2 := inst>>12&0b1111, g.armALUOp2(inst)
	g.R[rd] = ^op2

	s := inst>>20&0b1 != 0
	if rd == 15 {
		if s {
			g.armALUChangeOSMode()
		} else {
			g.pipelining()
		}
	} else if s {
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

func (g *GBA) armMPYCycle(cycle int, val uint32) {
	g.timer(cycle)
	switch {
	case val&0xfff0 == 0xfff0:
		g.timer(1)
	case val&0xff00 == 0xff00:
		g.timer(2)
	case val&0xf000 == 0xf000:
		g.timer(3)
	default:
		g.timer(4)
	}

	g.timer(g.cycleS2N())
}

// Rd=Rm*Rs
func (g *GBA) armMUL(inst uint32) {
	rd, rs, rm := inst>>16&0b1111, inst>>8&0b1111, inst&0b1111
	g.R[rd] = g.R[rm] * g.R[rs]
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}

	g.armMPYCycle(0, g.R[rs])
}

// Rd=Rm*Rs+Rn
func (g *GBA) armMLA(inst uint32) {
	rd, rn, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
	g.R[rd] = g.R[rm]*g.R[rs] + g.R[rn]
	if s := inst>>20&0b1 != 0; s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}

	g.armMPYCycle(1, g.R[rs])
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

	g.armMPYCycle(1, g.R[rs])
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

	g.armMPYCycle(2, g.R[rs])
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

	g.armMPYCycle(1, g.R[rs])
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

	g.armMPYCycle(2, g.R[rs])
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
	g.R[rd] = uint32(g.getRAM16(addr, false))
	if rd == 15 {
		g.pipelining()
	}
	g.timer(1)
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
	g.R[rd] = uint32(int8(g.getRAM8(addr, false)))
	if rd == 15 {
		g.pipelining()
	}
	g.timer(1)
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
	g.R[rd] = uint32(int16(g.getRAM16(addr, false)))
	if rd == 15 {
		g.pipelining()
	}
	g.timer(1)
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
	g.setRAM16(addr, uint16(g.R[rd]), false)
	g.timer(g.cycleS2N())
}

// ref: https://github.com/gdkchan/gdkGBA/blob/master/arm.c#L1223
const (
	PRIV_MASK  uint32 = 0xf8ff03df
	USR_MASK   uint32 = 0xf8ff0000
	STATE_MASK uint32 = 0x01000020
)

func (g *GBA) armMRS(inst uint32) {
	useSpsr := (inst>>22)&0b1 > 0
	rd := (inst >> 12) & 0b1111
	if useSpsr {
		mode := g.getOSMode()
		g.R[rd] = g.SPSRBank[bankIdx(mode)]
		return
	}

	mask := PRIV_MASK
	if g.getOSMode() == USR {
		mask = USR_MASK
	}
	g.R[rd] = g.CPSR & mask
}

func (g *GBA) armMSR(inst uint32) {
	mask := uint32(0)
	if c := util.Bit(inst, 16); c {
		mask = 0x0000_00ff
	}
	if x := util.Bit(inst, 17); x {
		mask |= 0x0000_ff00
	}
	if s := util.Bit(inst, 18); s {
		mask |= 0x00ff_0000
	}
	if f := util.Bit(inst, 19); f {
		mask |= 0xff00_0000
	}

	secMask := PRIV_MASK
	if g.getOSMode() == USR {
		secMask = USR_MASK
	}

	r := util.Bit(inst, 22)
	if r {
		secMask |= STATE_MASK
	}

	mask &= secMask
	psr := uint32(0)
	if util.Bit(inst, 25) {
		// register Psr[field] = Imm
		is, imm := ((inst>>8)&0b1111)*2, inst&0b1111_1111
		psr = util.ROR(imm, uint(is))
	} else {
		// immediate Psr[field] = Rm
		rm := inst & 0b1111
		psr = g.R[rm]
	}
	psr &= mask

	if r {
		spsr := g.SPSRBank[bankIdx(g.getOSMode())]
		g.setSPSR((spsr & ^mask) | psr)
	} else {
		currMode := g.getOSMode()
		newMode := Mode(psr & 0b11111)
		g.CPSR &= ^mask
		g.CPSR |= psr
		g.copyRegToBank(currMode)
		g.copyBankToReg(newMode)
		g.checkIRQ()
	}
}
