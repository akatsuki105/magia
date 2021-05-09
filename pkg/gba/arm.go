package gba

import (
	"fmt"
	"mettaur/pkg/util"
	"os"
)

const (
	lsl = iota
	lsr
	asr
	ror
)

func (g *GBA) armStep() {
	pc := util.Align2(g.R[15])
	g.pipe.inst[1] = Inst{
		inst: g.getRAM32(pc, true),
		loc:  pc,
	}
	g.armExec(g.inst.inst)
	if g.pipe.ok {
		g.pipe.ok = false
		return
	}
	g.R[15] = pc + 4
}

func (g *GBA) armExec(inst uint32) {
	if cond := Cond(inst >> 28); g.Check(cond) {
		switch {
		case isArmSWI(inst):
			g.armSWI(inst)
		case isArmBL(inst):
			g.armBL(inst)
		case isArmB(inst):
			g.armB(inst)
		case isArmBX(inst):
			g.armBX(inst)
		case isArmLDM(inst):
			g.armLDM(inst)
		case isArmSTM(inst):
			g.armSTM(inst)
		case isArmLDR(inst):
			g.armLDR(inst)
		case isArmSTR(inst):
			g.armSTR(inst)
		case isArmLDRH(inst):
			g.armLDRH(inst)
		case isArmLDRSB(inst):
			g.armLDRSB(inst)
		case isArmLDRSH(inst):
			g.armLDRSH(inst)
		case isArmSTRH(inst):
			g.armSTRH(inst)
		case isArmMRS(inst):
			g.armMRS(inst)
		case isArmMSR(inst):
			g.armMSR(inst)
		case isArmSWP(inst):
			fmt.Fprintf(os.Stderr, "SWI is unsupported in 0x%08x\n", g.inst.loc)
			g.Exit("")
		case isArmMPY(inst):
			g.armMPY(inst)
		case isArmALU(inst):
			g.armALU(inst)
		default:
			fmt.Fprintf(os.Stderr, "invalid ARM opcode(0x%08x) in 0x%08x\n", inst, g.inst.loc)
			g.Exit("")
		}
	}
}

func (g *GBA) armSWI(inst uint32) {
	if debug {
		nn := byte(inst >> 16)
		g.printSWI(nn)
	}
	g.exception(swiVec, SWI)
}

func (g *GBA) armB(inst uint32) {
	nn := int32(inst)
	nn = (nn << 8) >> 6
	g.R[15] = util.AddInt32(g.inst.loc+8, nn)
	g.pipelining()
}

func (g *GBA) armBL(inst uint32) {
	nn := int32(inst)
	nn = (nn << 8) >> 6
	g.R[14] = g.inst.loc + 4
	g.R[15] = util.AddInt32(g.inst.loc+8, nn)
	g.pipelining()
}

func (g *GBA) armBX(inst uint32) {
	rnval := g.R[inst&0b1111]
	if util.Bit(rnval, 0) {
		g.SetCPSRFlag(flagT, true)
		g.R[15] = rnval - 1
	} else {
		g.R[15] = rnval
	}
	g.pipelining()
}

func (g *GBA) armLDM(inst uint32) {
	if s := util.Bit(inst, 22); s {
		g._armLDMUsr(inst)
		return
	}
	g._armLDM(inst)
}

// LDM with S(bit22)
func (g *GBA) _armLDMUsr(inst uint32) {
	if util.Bit(inst, 15) {
		g._armLDM(inst)
		g.restorePrivMode()
		g.checkIRQ()
		return
	}
	mode := g.getPrivMode()
	g.setPrivMode(USR)
	g._armLDM(inst)
	g.setPrivMode(mode)
}

func (g *GBA) _armLDM(inst uint32) {
	p, u := util.Bit(inst, 24), util.Bit(inst, 23)
	rn := inst >> 16 & 0b1111
	rnval := g.R[rn]

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
				if rs != int(rn) {
					g.R[rn] += 4
				}
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
				if rs != int(rn) {
					g.R[rn] -= 4
				}
				n++
			}
		}
	}

	g.timer(1)
	if util.Bit(inst, 15) {
		g.pipelining()
	}

	// Pre-indexing, write-back is optional
	writeBack := util.Bit(inst, 21)
	if p && !writeBack {
		g.R[rn] = rnval
	}
}

func (g *GBA) armSTM(inst uint32) {
	if s := util.Bit(inst, 22); s {
		g._armSTMUsr(inst)
		return
	}
	g._armSTM(inst)
}

func (g *GBA) _armSTMUsr(inst uint32) {
	mode := g.getPrivMode()
	g.setPrivMode(USR)
	g._armSTM(inst)
	g.setPrivMode(mode)
}

func (g *GBA) _armSTM(inst uint32) {
	p, u := util.Bit(inst, 24), util.Bit(inst, 23)
	rn := (inst >> 16) & 0b1111
	rnval := g.R[rn]

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

	// Pre-indexing, write-back is optional
	writeBack := util.Bit(inst, 21)
	if p && !writeBack {
		g.R[rn] = rnval
	}
}

func (g *GBA) armRegShiftOffset(inst uint32) uint32 {
	ofs := uint32(0)
	if util.Bit(inst, 25) {
		is := inst >> 7 & 0b11111 // I = 1 shift reg
		rm := inst & 0b1111
		switch shiftType := inst >> 5 & 0b11; shiftType {
		case lsl:
			ofs = g.armLSL(g.R[rm], is, false, true)
		case lsr:
			ofs = g.armLSR(g.R[rm], is, false, true)
		case asr:
			ofs = g.armASR(g.R[rm], is, false, true)
		case ror:
			ofs = g.armROR(g.R[rm], is, false, true)
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

	// writeBack
	if pre {
		// Pre-indexing, write-back is optional
		if writeBack := util.Bit(inst, 21); writeBack {
			if rn != rd { // if rn is equal to rd, don't write back
				g.R[rn] = addr
			}
		}
	} else {
		// Post-indexing, write-back is ALWAYS enabled
		if plus {
			addr += ofs
		} else {
			addr -= ofs
		}
		if rn != rd { // if rn is equal to rd, don't write back
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
	rdval := g.R[rd]
	if rd == 15 { // https://github.com/jsmolka/gba-tests/blob/a6447c5404c8fc2898ddc51f438271f832083b7e/arm/single_transfer.asm#L94
		rdval += 4
	}
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
		g.setRAM8(addr, byte(rdval), false)
	} else {
		g.setRAM32(addr, rdval, false)
	}

	// writeBack
	if pre {
		// Pre-indexing, write-back is optional
		if writeBack := util.Bit(inst, 21); writeBack {
			g.R[rn] = addr
		}
	} else {
		// Post-indexing, write-back is ALWAYS enabled
		if plus {
			addr += ofs
		} else {
			addr -= ofs
		}
		g.R[rn] = addr
	}

	g.timer(g.cycleS2N())
}

func (g *GBA) armALUOp2(inst uint32) uint32 {
	if !util.Bit(inst, 25) { // op rd, rn
		// register
		is := (inst >> 7) & 0b11111
		rm := inst & 0b1111

		salt := uint32(0)
		isRegister := util.Bit(inst, 4)
		if isRegister {
			g.timer(1)
			is = g.R[(inst>>8)&0b1111] & 0b1111_1111
			if rm == 15 {
				salt = 4
			}
		}

		carryMut := util.Bit(inst, 20)
		switch shiftType := (inst >> 5) & 0b11; shiftType {
		case lsl:
			return g.armLSL(g.R[rm]+salt, is, carryMut, !isRegister)
		case lsr:
			return g.armLSR(g.R[rm]+salt, is, carryMut, !isRegister)
		case asr:
			return g.armASR(g.R[rm]+salt, is, carryMut, !isRegister)
		case ror:
			return g.armROR(g.R[rm]+salt, is, carryMut, !isRegister)
		}
		return g.R[rm] + salt
	}

	// immediate(op rd, imm)
	op2 := inst & 0b1111_1111
	is := uint32((inst>>8)&0b1111) * 2
	carryMut := util.Bit(inst, 20)
	op2 = g.armROR(op2, is, carryMut, false)
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
	g.restorePrivMode()
	g.pipelining()
	g.checkIRQ()
}

func (g *GBA) armAND(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = rnval & op2

	s := util.Bit(inst, 20)
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

	s := util.Bit(inst, 20)
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

	s := util.Bit(inst, 20)
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

	s := util.Bit(inst, 20)
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

	s := util.Bit(inst, 20)
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
	carry := uint32(0)
	if g.GetCPSRFlag(flagC) {
		carry = 1
	}
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = rnval + op2 + carry

	s := util.Bit(inst, 20)
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
	carry := uint32(0)
	if g.GetCPSRFlag(flagC) {
		carry = 1
	}
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = rnval - op2 + carry - 1

	s := util.Bit(inst, 20)
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
	carry := uint32(0)
	if g.GetCPSRFlag(flagC) {
		carry = 1
	}
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = op2 - rnval + carry - 1

	s := util.Bit(inst, 20)
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
	if s := util.Bit(inst, 20); s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
	}
}

func (g *GBA) armTEQ(inst uint32) {
	rnval, op2 := g.armALURn(inst), g.armALUOp2(inst)
	result := rnval ^ op2
	if s := util.Bit(inst, 20); s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
	}
}

func (g *GBA) armCMP(inst uint32) {
	rnval, op2 := g.armALURn(inst), g.armALUOp2(inst)
	result := rnval - op2
	if s := util.Bit(inst, 20); s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
		g.SetCPSRFlag(flagC, util.SubC(uint64(rnval)-uint64(op2)))
		g.SetCPSRFlag(flagV, util.SubV(rnval, op2, result))
	}
}

func (g *GBA) armCMN(inst uint32) {
	rnval, op2 := g.armALURn(inst), g.armALUOp2(inst)
	result := rnval + op2
	if s := util.Bit(inst, 20); s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
		g.SetCPSRFlag(flagC, util.AddC(uint64(rnval)+uint64(op2)))
		g.SetCPSRFlag(flagV, util.AddV(rnval, op2, result))
	}
}

func (g *GBA) armORR(inst uint32) {
	rd, rnval, op2 := inst>>12&0b1111, g.armALURn(inst), g.armALUOp2(inst)
	g.R[rd] = rnval | op2

	s := util.Bit(inst, 20)
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

	s := util.Bit(inst, 20)
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

	s := util.Bit(inst, 20)
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

	s := util.Bit(inst, 20)
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
	opcode := inst >> 21 & 0b1111
	switch opcode {
	case 0b0000:
		g.armMUL(inst)
	case 0b0001:
		g.armMLA(inst)
	case 0b0010:
		fmt.Fprintf(os.Stderr, "UMAAL is unsupported in 0x%08x\n", g.inst.loc)
	case 0b0100: // umull
		g.armUMULL(inst)
	case 0b0101: // umlal
		g.armUMLAL(inst)
	case 0b0110: // smull
		g.armSMULL(inst)
	case 0b0111: // smlal
		g.armSMLAL(inst)
	default:
		fmt.Fprintf(os.Stderr, "invalid opcode(%d) is unsupported in 0x%08x\n", opcode, g.inst.loc)
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
	if s := util.Bit(inst, 20); s {
		g.SetCPSRFlag(flagZ, g.R[rd] == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	}

	g.armMPYCycle(0, g.R[rs])
}

// Rd=Rm*Rs+Rn
func (g *GBA) armMLA(inst uint32) {
	rd, rn, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
	g.R[rd] = g.R[rm]*g.R[rs] + g.R[rn]
	if s := util.Bit(inst, 20); s {
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
	if s := util.Bit(inst, 20); s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rdHi], 31))
	}

	g.armMPYCycle(1, g.R[rs])
}

// RdHiLo=Rm*Rs+RdHiLo
func (g *GBA) armUMLAL(inst uint32) {
	rdHi, rdLo, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
	result := uint64(g.R[rs])*uint64(g.R[rm]) + (uint64(g.R[rdHi])<<32 | uint64(g.R[rdLo]))
	g.R[rdHi], g.R[rdLo] = uint32(result>>32), uint32(result)
	if s := util.Bit(inst, 20); s {
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
	if s := util.Bit(inst, 20); s {
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagN, util.Bit(g.R[rdHi], 31))
	}

	g.armMPYCycle(1, g.R[rs])
}

// RdHiLo=Rm*Rs+RdHiLo
func (g *GBA) armSMLAL(inst uint32) {
	rdHi, rdLo, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
	result := int64(int32(g.R[rs]))*int64(int32(g.R[rm])) + (int64(g.R[rdHi])<<32 | int64(g.R[rdLo]))
	g.R[rdHi], g.R[rdLo] = uint32(result>>32), uint32(result)
	if s := util.Bit(inst, 20); s {
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
			if rn != rd {
				g.R[rn] = addr
			}
		}
	}
	g.R[rd] = uint32(g.getRAM16(addr, false))
	if !pre { // Post-indexing, write-back is ALWAYS enabled
		if plus := util.Bit(inst, 23); plus {
			addr += ofs
		} else {
			addr -= ofs
		}
		if rn != rd {
			g.R[rn] = addr
		}
	}
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
	if !pre {
		// Post-indexing, write-back is ALWAYS enabled
		if plus := util.Bit(inst, 23); plus {
			addr += ofs
		} else {
			addr -= ofs
		}
		if rn != rd {
			g.R[rn] = addr
		}
	}
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
			if rn != rd {
				g.R[rn] = addr
			}
		}
	}

	val := g.getRAM16(addr, false)
	if addr%2 == 1 { // https://github.com/jsmolka/gba-tests/blob/a6447c5404c8fc2898ddc51f438271f832083b7e/arm/halfword_transfer.asm#L141
		val = ((val & 0xff) << 24) | ((val & 0xff) << 16) | ((val & 0xff) << 8) | val
	}
	g.R[rd] = uint32(int16(val))

	if !pre { // Post-indexing, write-back is ALWAYS enabled
		if plus := util.Bit(inst, 23); plus {
			addr += ofs
		} else {
			addr -= ofs
		}
		if rn != rd {
			g.R[rn] = addr
		}
	}
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
	if !pre { // Post-indexing, write-back is ALWAYS enabled
		if plus := util.Bit(inst, 23); plus {
			addr += ofs
		} else {
			addr -= ofs
		}
		g.R[rn] = addr
	}
	g.timer(g.cycleS2N())
}

// ref: https://github.com/gdkchan/gdkGBA/blob/master/arm.c#L1223
const (
	PRIV_MASK  uint32 = 0xf8ff03df
	USR_MASK   uint32 = 0xf8ff0000
	STATE_MASK uint32 = 0x01000020
)

func (g *GBA) armMRS(inst uint32) {
	rd := (inst >> 12) & 0b1111
	if useSpsr := util.Bit(inst, 22); useSpsr {
		mode := g.getPrivMode()
		g.R[rd] = g.SPSRBank[bankIdx[mode]]
		return
	}

	mask := PRIV_MASK
	if g.getPrivMode() == USR {
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
	if g.getPrivMode() == USR {
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
		spsr := g.SPSRBank[bankIdx[g.getPrivMode()]]
		g.setSPSR((spsr & ^mask) | psr)
	} else {
		currMode := g.getPrivMode()
		newMode := Mode(psr & 0b11111)
		g.CPSR &= ^mask
		g.CPSR |= psr
		g.Reg._setPrivMode(currMode, newMode)
		g.checkIRQ()
	}
}
