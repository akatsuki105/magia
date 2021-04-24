package gba

import (
	"fmt"
	"mettaur/pkg/util"
	"os"
)

func (g *GBA) thumbStep() {
	pc := util.Align2(g.R[15])
	g.pipe.inst[1] = Inst{
		inst: uint32(g.getRAM16(pc, true)),
		loc:  pc,
	}
	g.thumbExec(uint16(g.inst.inst))
	if g.pipe.ok {
		g.pipe.ok = false
		return
	}
	g.R[15] = pc + 2
}

func (g *GBA) thumbExec(inst uint16) {
	// g.thumbInst(inst)
	switch {
	case isThumbShift(inst):
		g.thumbShift(inst)
	case isThumbAddSub(inst):
		g.thumbAddSub(inst)
	case isThumbMovCmpAddSub(inst):
		g.thumbMovCmpAddSub(inst)
	case isThumbALU(inst):
		g.thumbALU(inst)
	case isHiRegisterBX(inst):
		g.thumbHiRegisterBX(inst)
	case isThumbLoadPCRel(inst):
		g.thumbLoadPCRel(inst)
	case isThumbLoadStoreRegOfs(inst):
		g.thumbLoadStoreRegOfs(inst)
	case isThumbLoadStoreSBH(inst):
		g.thumbLoadStoreSBH(inst)
	case isThumbLoadStoreImmOfs(inst):
		g.thumbLoadStoreImmOfs(inst)
	case isThumbLoadStoreH(inst):
		g.thumbLoadStoreH(inst)
	case isThumbLoadSPRel(inst):
		g.thumbLoadSPRel(inst)
	case isThumbStack(inst):
		g.thumbStack(inst)
	case isThumbStackMultiple(inst):
		g.thumbStackMultiple(inst)
	case isThumbGetAddr(inst):
		g.thumbGetAddr(inst)
	case isThumbMoveSP(inst):
		g.thumbMoveSP(inst)
	case isThumbCondBranch(inst):
		g.thumbCondBranch(inst)
	case isThumbSWI(inst):
		g.thumbSWI(inst)
	case isThumbB(inst):
		g.thumbB(inst)
	case isThumbLinkBranch1(inst):
		g.thumbLinkBranch1(inst)
	case isThumbLinkBranch2(inst):
		g.thumbLinkBranch2(inst)
	default:
		fmt.Fprintf(os.Stderr, "invalid THUMB opcode(0x%04x) in 0x%08x\n", inst, g.inst.loc)
		g.Exit("")
	}
}

func (g *GBA) thumbShift(inst uint16) {
	is, rs, rd := uint32((inst>>6)&0b11111), (inst>>3)&0b111, inst&0b111
	switch opcode := (inst >> 11) & 0b11; opcode {
	case 0:
		g.R[rd] = g.armLSL(g.R[rs], is, true, true)
	case 1:
		g.R[rd] = g.armLSR(g.R[rs], is, true, true)
	case 2:
		g.R[rd] = g.armASR(g.R[rs], is, true, true)
	}

	g.SetCPSRFlag(flagZ, g.R[rd] == 0)
	g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
}

func (g *GBA) thumbAddSub(inst uint16) {
	delta, rs, rd := (inst>>6)&0b111, (inst>>3)&0b111, inst&0b111
	lhs, rhs := g.R[rs], g.R[delta]
	switch opcode := (inst >> 9) & 0b11; opcode {
	case 0: // ADD Rd,Rs,Rn
		result := uint64(lhs) + uint64(rhs)
		g.R[rd] = lhs + rhs
		g.SetCPSRFlag(flagC, util.AddC(result))
		g.SetCPSRFlag(flagV, util.AddV(lhs, rhs, uint32(result)))
	case 1: // SUB Rd,Rs,Rn
		result := uint64(lhs) - uint64(rhs)
		g.R[rd] = lhs - rhs
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(lhs, rhs, uint32(result)))
	case 2: // ADD Rd,Rs,#nn
		result := uint64(lhs) + uint64(delta)
		g.R[rd] = lhs + uint32(delta)
		g.SetCPSRFlag(flagC, util.AddC(result))
		g.SetCPSRFlag(flagV, util.AddV(lhs, uint32(delta), uint32(result)))
	case 3: // SUB Rd,Rs,#nn
		result := uint64(lhs) - uint64(delta)
		g.R[rd] = lhs - uint32(delta)
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(lhs, uint32(delta), uint32(result)))
	}

	g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	g.SetCPSRFlag(flagZ, g.R[rd] == 0)
}

func (g *GBA) thumbMovCmpAddSub(inst uint16) {
	rd, nn := (inst>>8)&0b111, uint32(inst&0b1111_1111)
	lhs := g.R[rd]
	result := uint64(0)
	switch opcode := (inst >> 11) & 0b11; opcode {
	case 0: // MOV Rd, #nn
		result = uint64(nn)
		g.R[rd] = nn
	case 1: // CMP
		result = uint64(g.R[rd]) - uint64(nn)
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(g.R[rd], uint32(nn), uint32(result)))
	case 2: // ADD
		result = uint64(g.R[rd]) + uint64(nn)
		g.R[rd] = g.R[rd] + nn
		g.SetCPSRFlag(flagC, util.AddC(result))
		g.SetCPSRFlag(flagV, util.AddV(lhs, uint32(nn), uint32(result)))
	case 3: // SUB
		result = uint64(g.R[rd]) - uint64(nn)
		g.R[rd] = g.R[rd] - nn
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(lhs, uint32(nn), uint32(result)))
	}

	g.SetCPSRFlag(flagN, util.Bit(result, 31))
	g.SetCPSRFlag(flagZ, uint32(result) == 0)
}

func (g *GBA) thumbALU(inst uint16) {
	rs, rd := (inst>>3)&0b111, inst&0b111
	lhs, rhs := g.R[rd], g.R[rs]
	result := uint64(0)
	switch opcode := (inst >> 6) & 0b1111; opcode {
	case 0: // AND
		g.R[rd] = g.R[rd] & g.R[rs]
		result = uint64(g.R[rd])
	case 1: // EOR(xor)
		g.R[rd] = g.R[rd] ^ g.R[rs]
		result = uint64(g.R[rd])
	case 2: // LSL
		is := g.R[rs] & 0xff
		g.R[rd] = g.armLSL(g.R[rd], is, is > 0, false) // Rd = Rd << (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 3: // LSR
		is := g.R[rs] & 0xff
		g.R[rd] = g.armLSR(g.R[rd], is, is > 0, false) // Rd = Rd >> (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 4: // ASR
		is := g.R[rs] & 0xff
		g.R[rd] = g.armASR(g.R[rd], is, is > 0, false) // Rd = Rd >> (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 5: // ADC
		result = uint64(g.R[rd]) + uint64(g.R[rs]) + uint64(util.BoolToInt(g.GetCPSRFlag(flagC)))
		g.R[rd] = g.R[rd] + g.R[rs] + uint32(util.BoolToInt(g.GetCPSRFlag(flagC))) // Rd = Rd + Rs + Carry
		g.SetCPSRFlag(flagC, util.AddC(result))
		g.SetCPSRFlag(flagV, util.AddV(lhs, rhs, uint32(result)))
	case 6: // SBC
		result = uint64(g.R[rd]) - uint64(g.R[rs]) - uint64(util.BoolToInt(!g.GetCPSRFlag(flagC)))
		g.R[rd] = g.R[rd] - g.R[rs] - uint32(util.BoolToInt(!g.GetCPSRFlag(flagC))) // Rd = Rd - Rs - NOT Carry
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(lhs, rhs, uint32(result)))
	case 7: // ROR
		is := g.R[rs] & 0xff
		g.R[rd] = g.armROR(g.R[rd], is, is > 0, false) // Rd = Rd ROR (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 8:
		result = uint64(g.R[rd] & g.R[rs]) // TST Rd,Rs
	case 9:
		rhs := g.R[rs]
		result = 0 - uint64(g.R[rs])
		g.R[rd] = -g.R[rs] // Rd = -Rs
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(0, rhs, g.R[rd]))
	case 10: // CMP
		result = uint64(g.R[rd]) - uint64(g.R[rs]) // Void = Rd - Rs
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(g.R[rd], g.R[rs], uint32(result)))
	case 11:
		result = uint64(g.R[rd]) + uint64(g.R[rs]) // Void = Rd + Rs
		g.SetCPSRFlag(flagC, util.AddC(result))
		g.SetCPSRFlag(flagV, util.AddV(g.R[rd], g.R[rs], uint32(result)))
	case 12:
		g.R[rd] = g.R[rd] | g.R[rs]
		result = uint64(g.R[rd])
	case 13:
		b1, b2, b3 := (g.R[rd]>>8)&0xff, (g.R[rd]>>16)&0xff, (g.R[rd]>>24)&0xff
		switch {
		case b3 > 0:
			g.timer(4)
		case b2 > 0:
			g.timer(3)
		case b1 > 0:
			g.timer(2)
		default:
			g.timer(1)
		}

		g.R[rd] = g.R[rd] * g.R[rs] // MUL{S} Rd,Rs
		result = uint64(g.R[rd])
		g.SetCPSRFlag(flagC, false)
	case 14:
		g.R[rd] = g.R[rd] & ^g.R[rs] // BIC{S} Rd,Rs
		result = uint64(g.R[rd])
	case 15:
		g.R[rd] = ^g.R[rs]
		result = uint64(g.R[rd])
	}

	g.SetCPSRFlag(flagN, util.Bit(result, 31))
	g.SetCPSRFlag(flagZ, uint32(result) == 0)
}

func (g *GBA) thumbHiRegisterBXOperand(r uint16) uint32 {
	if r == 15 {
		return g.inst.loc + 4
	}
	return g.R[r]
}

func (g *GBA) thumbHiRegisterBX(inst uint16) {
	rs, rd := (inst>>3)&0b111, inst&0b111
	if util.Bit(inst, 7) {
		rd += 8
	}
	if util.Bit(inst, 6) {
		rs += 8
	}
	rsval, rdval := g.thumbHiRegisterBXOperand(rs), g.thumbHiRegisterBXOperand(rd)

	opcode := (inst >> 8) & 0b11
	switch opcode {
	case 0: // ADD Rd,Rs(Rd = Rd+Rs)
		g.R[rd] = rdval + rsval
	case 1: // CMP Rd,Rs(Void = Rd-Rs)
		result := uint64(rdval) - uint64(rsval)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
		g.SetCPSRFlag(flagZ, uint32(result) == 0)
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(rdval, rsval, uint32(result)))
	case 2: // MOV Rd,Rs(Rd=Rs)
		g.R[rd] = rsval
	case 3: // BX Rs(PC = Rs)
		rd = 15
		if util.Bit(rsval, 0) {
			g.R[15] = rsval - 1
		} else {
			g.SetCPSRFlag(flagT, false) // switch to ARM
			if rs == 15 {
				g.R[15] = util.Align4(g.inst.loc + 4)
			} else {
				g.R[15] = rsval
			}
		}
	}

	if opcode != 1 && rd == 15 {
		g.pipelining()
	}
}

func (g *GBA) thumbLoadPCRel(inst uint16) {
	rd, nn := (inst>>8)&0b111, uint32(inst&0b1111_1111)*4
	pc := util.Align4(g.inst.loc + 4)
	g.R[rd] = g.getRAM32(pc+nn, false)
	g.timer(1)
}

func (g *GBA) thumbLoadStoreRegOfs(inst uint16) {
	ro, rb, rd := (inst>>6)&0b111, (inst>>3)&0b111, inst&0b111

	switch opcode := (inst >> 10) & 0b11; opcode {
	case 0: // STR Rd,[Rb,Ro]
		g.setRAM32(g.R[rb]+g.R[ro], g.R[rd], false) // N
		g.timer(g.cycleS2N())                       // -S + 2N
	case 1: // STRB Rd,[Rb,Ro] (BYTE[Rb+Ro] = Rd)
		g.setRAM8(g.R[rb]+g.R[ro], byte(g.R[rd]), false)
		g.timer(g.cycleS2N())
	case 2: // LDR Rd,[Rb,Ro] (Rd = WORD[Rb+Ro])
		g.R[rd] = g.getRAM32(g.R[rb]+g.R[ro], false)
		g.timer(1)
	case 3: // LDRB Rd,[Rb,Ro]
		g.R[rd] = uint32(g.getRAM8(g.R[rb]+g.R[ro], false))
		g.timer(1)
	}
}

func (g *GBA) thumbLoadStoreSBH(inst uint16) {
	ro, rb, rd := uint32((inst>>6)&0b111), (inst>>3)&0b111, inst&0b111

	switch opcode := (inst >> 10) & 0b11; opcode {
	case 0: // STRH Rd,[Rb,Ro]
		g.setRAM16(g.R[rb]+g.R[ro], uint16(g.R[rd]), false)
		g.timer(g.cycleS2N())
	case 1: // LDSB Rd,[Rb,Ro]
		value := int32(g.getRAM8(g.R[rb]+g.R[ro], false))
		value = (value << 24) >> 24
		g.R[rd] = uint32(value)
		g.timer(1)
	case 2: // LDRH Rd,[Rb,Ro]
		g.R[rd] = uint32(g.getRAM16(g.R[rb]+g.R[ro], false))
		g.timer(1)
	case 3: // LDSH Rd,[Rb,Ro]
		value := int32(g.getRAM16(g.R[rb]+g.R[ro], false))
		value = (value << 16) >> 16
		g.R[rd] = uint32(value)
		g.timer(1)
	}
}

func (g *GBA) thumbLoadStoreImmOfs(inst uint16) {
	nn, rb, rd := uint32((inst>>6)&0b11111), (inst>>3)&0b111, inst&0b111

	switch opcode := (inst >> 11) & 0b11; opcode {
	case 0: // STR Rd,[Rb,#nn]
		nn *= 4
		g.setRAM32(g.R[rb]+nn, g.R[rd], false)
		g.timer(g.cycleS2N())
	case 1: // LDR Rd,[Rb,#nn]
		nn *= 4
		g.R[rd] = g.getRAM32(g.R[rb]+nn, false)
		g.timer(1)
	case 2: // STRB Rd,[Rb,#nn]
		g.setRAM8(g.R[rb]+nn, byte(g.R[rd]), false)
		g.timer(g.cycleS2N())
	case 3: // LDRB Rd,[Rb,#nn]
		g.R[rd] = uint32(g.getRAM8(g.R[rb]+nn, false))
		g.timer(1)
	}
}

func (g *GBA) thumbLoadStoreH(inst uint16) {
	nn, rb, rd := uint32(((inst>>6)&0b11111)*2), (inst>>3)&0b111, inst&0b111

	switch opcode := (inst >> 11) & 0b1; opcode {
	case 0: // STRH Rd,[Rb,#nn]
		g.setRAM16(g.R[rb]+nn, uint16(g.R[rd]), false)
		g.timer(g.cycleS2N())
	case 1: // LDRH Rd,[Rb,#nn]
		g.R[rd] = uint32(g.getRAM16(g.R[rb]+nn, false))
		g.timer(1)
	}
}

func (g *GBA) thumbLoadSPRel(inst uint16) {
	rd, nn := (inst>>8)&0b111, uint32((inst&0b1111_1111)*4)

	sp, opcode := g.R[13], (inst>>11)&0b1
	switch opcode {
	case 0:
		g.setRAM32(sp+nn, g.R[rd], false)
		g.timer(g.cycleS2N())
	case 1:
		g.R[rd] = g.getRAM32(sp+nn, false)
		g.timer(1)
	}
}

// thumbStack push, pop
func (g *GBA) thumbStack(inst uint16) {
	rlist := inst & 0b1111_1111

	switch opcode := (inst >> 11) & 0b1; opcode {
	case 0:
		n, lr := 0, util.Bit(inst, 8)
		if lr {
			g.R[13] -= 4
			g.setRAM32(g.R[13], g.R[14], n > 0) // PUSH lr
			n++
		}
		for i := 7; i >= 0; i-- {
			if rlist&(0b1<<i) > 0 {
				g.R[13] -= 4
				g.setRAM32(g.R[13], g.R[i], n > 0) // PUSH
				n++
			}
		}
		g.timer(g.cycleS2N())
	case 1:
		n := 0
		for i := 0; i < 8; i++ {
			if rlist&(0b1<<i) > 0 {
				g.R[i] = g.getRAM32(g.R[13], n > 0) // POP
				g.R[13] += 4
				n++
			}
		}
		if pc := util.Bit(inst, 8); pc {
			g.R[15] = g.getRAM32(g.R[13], n > 0) // POP pc
			g.R[15] = util.Align2(g.R[15])
			g.R[13] += 4
			g.pipelining()
		}
		g.timer(1)
	}
}

// thumbStackMultiple ldmia, stmia
func (g *GBA) thumbStackMultiple(inst uint16) {
	rb, rlist := (inst>>8)&0b111, inst&0b1111_1111

	switch opcode := (inst >> 11) & 0b1; opcode {
	case 0:
		n := 0
		for i := 0; i < 8; i++ {
			if rlist&(0b1<<i) > 0 {
				g.setRAM32(g.R[rb], g.R[i], n > 0) // STMIA
				g.R[rb] += 4
				n++
			}
		} // (n-1)S + N
		g.timer(g.cycleS2N()) // (n-2)S + 2N
	case 1:
		n := 0
		for i := 0; i < 8; i++ {
			if rlist&(0b1<<i) > 0 {
				g.R[i] = g.getRAM32(g.R[rb], n > 0) // LDMIA
				g.R[rb] += 4
				n++
			}
		} // (n-1)S + N
		g.timer(1) // (n-1)S + N + 1
	}
}

// thumbGetAddr get relative address
func (g *GBA) thumbGetAddr(inst uint16) {
	rd, nn := (inst>>8)&0b111, uint32((inst&0b1111_1111)*4)

	switch opcode := (inst >> 11) & 0b1; opcode {
	case 0:
		g.R[rd] = (util.Align4(g.inst.loc + 4)) + nn // ADD  Rd,PC,#nn
	case 1:
		g.R[rd] = g.R[13] + nn // ADD  Rd,SP,#nn
	}
}

func (g *GBA) thumbMoveSP(inst uint16) {
	nn := uint32((inst & 0b0111_1111) * 4)
	switch opcode := (inst >> 7) & 0b1; opcode {
	case 0:
		g.R[13] += nn // ADD SP,#nn
	case 1:
		g.R[13] -= nn // ADD SP,#-nn
	}
}

func (g *GBA) thumbCondBranch(inst uint16) {
	if cond := Cond((inst >> 8) & 0b1111); g.Check(cond) {
		nn := int32(int8(byte(inst & 0b1111_1111)))
		g.R[15] = util.AddInt32(g.inst.loc+4, nn*2)
		g.pipelining()
	}
}

func (g *GBA) thumbSWI(inst uint16) {
	if debug {
		nn := byte(inst)
		g.printSWI(nn)
	}
	g.exception(swiVec, SWI)
}

func (g *GBA) thumbB(inst uint16) {
	nn := int32(inst)
	nn = (nn << 21) >> 20
	g.R[15] = util.AddInt32(g.inst.loc+4, nn)
	g.pipelining()
}

func (g *GBA) thumbLinkBranch1(inst uint16) {
	nn := int32(inst)
	nn = (nn << 21) >> 9
	g.R[14] = g.inst.loc + 4
	g.R[14] = util.AddInt32(g.R[14], nn)
}

func (g *GBA) thumbLinkBranch2(inst uint16) {
	nn := inst & 0b0111_1111_1111
	g.R[15] = g.R[14] + uint32(nn<<1)
	g.R[14] = g.inst.loc + 2 // return
	if g.R[14]&1 == 0 {
		g.R[14]++
	}

	g.pipelining()
}
