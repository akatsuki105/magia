package gba

import (
	"fmt"
	"mettaur/pkg/util"
	"os"
)

func (g *GBA) thumbStep() {
	g.pipe.inst[1] = Inst{
		inst: uint32(g.getRAM16(g.R[15], true)),
		loc:  g.R[15],
	}
	g.thumbExec(uint16(g.inst.inst))
	if g.pipe.ok {
		g.pipe.ok = false
		return
	}
	g.R[15] += 2
}

func (g *GBA) thumbExec(inst uint16) {
	// g.thumbInst(inst)
	switch {
	case IsThumbShift(inst):
		g.thumbShift(inst)
	case IsThumbAddSub(inst):
		g.thumbAddSub(inst)
	case IsThumbMovCmpAddSub(inst):
		g.thumbMovCmpAddSub(inst)
	case IsThumbALU(inst):
		g.thumbALU(inst)
	case IsHiRegisterBX(inst):
		g.thumbHiRegisterBX(inst)
	case IsThumbLoadPCRel(inst):
		g.thumbLoadPCRel(inst)
	case IsThumbLoadStoreRegOfs(inst):
		g.thumbLoadStoreRegOfs(inst)
	case IsThumbLoadStoreSBH(inst):
		g.thumbLoadStoreSBH(inst)
	case IsThumbLoadStoreImmOfs(inst):
		g.thumbLoadStoreImmOfs(inst)
	case IsThumbLoadStoreH(inst):
		g.thumbLoadStoreH(inst)
	case IsThumbLoadSPRel(inst):
		g.thumbLoadSPRel(inst)
	case IsThumbStack(inst):
		g.thumbStack(inst)
	case IsThumbStackMultiple(inst):
		g.thumbStackMultiple(inst)
	case IsThumbGetAddr(inst):
		g.thumbGetAddr(inst)
	case IsThumbMoveSP(inst):
		g.thumbMoveSP(inst)
	case IsThumbCondBranch(inst):
		g.thumbCondBranch(inst)
	case IsThumbSWI(inst):
		g.thumbSWI(inst)
	case IsThumbB(inst):
		g.thumbB(inst)
	case IsThumbLinkBranch1(inst):
		g.thumbLinkBranch1(inst)
	case IsThumbLinkBranch2(inst):
		g.thumbLinkBranch2(inst)
	default:
		fmt.Fprintf(os.Stderr, "invalid THUMB opcode(0x%04x) in 0x%08x\n", inst, g.inst.loc)
		panic("")
	}
}

func (g *GBA) thumbShift(inst uint16) {
	is, rs, rd := uint32((inst>>6)&0b11111), (inst>>3)&0b111, inst&0b111
	switch opcode := (inst >> 11) & 0b11; opcode {
	case 0:
		fmt.Sprintf("lsl r%d, r%d, #%d\n", rd, rs, is)
		g.R[rd] = g.armLSL(g.R[rs], is, true)
	case 1:
		fmt.Sprintf("lsr r%d, r%d, #%d\n", rd, rs, is)
		g.R[rd] = g.armLSR(g.R[rs], is, true)
	case 2:
		fmt.Sprintf("asr r%d, r%d, #%d\n", rd, rs, is)
		g.R[rd] = g.armASR(g.R[rs], is, true)
	}

	g.SetCPSRFlag(flagZ, g.R[rd] == 0)
	g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
}

func (g *GBA) thumbAddSub(inst uint16) {
	delta, rs, rd := (inst>>6)&0b111, (inst>>3)&0b111, inst&0b111
	lhs, rhs := g.R[rs], g.R[delta]
	switch opcode := (inst >> 9) & 0b11; opcode {
	case 0:
		// ADD Rd,Rs,Rn
		fmt.Sprintf("add r%d, r%d, r%d\n", rd, rs, delta)
		result := uint64(lhs) + uint64(rhs)
		g.R[rd] = lhs + rhs
		g.SetCPSRFlag(flagC, util.AddC(result))
		g.SetCPSRFlag(flagV, util.AddV(lhs, rhs, uint32(result)))
	case 1:
		// SUB Rd,Rs,Rn
		fmt.Sprintf("sub r%d, r%d, r%d\n", rd, rs, delta)
		result := uint64(lhs) - uint64(rhs)
		g.R[rd] = lhs - rhs
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(lhs, rhs, uint32(result)))
	case 2:
		// ADD Rd,Rs,#nn
		fmt.Sprintf("add r%d, r%d, 0x%04x\n", rd, rs, delta)
		result := uint64(lhs) + uint64(delta)
		g.R[rd] = lhs + uint32(delta)
		g.SetCPSRFlag(flagC, util.AddC(result))
		g.SetCPSRFlag(flagV, util.AddV(lhs, uint32(delta), uint32(result)))
	case 3:
		// SUB Rd,Rs,#nn
		fmt.Sprintf("sub r%d, r%d, 0x%04x\n", rd, rs, delta)
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
	case 0:
		// MOV Rd, #nn
		fmt.Sprintf("mov r%d, 0x%04x\n", rd, nn)
		result = uint64(nn)
		g.R[rd] = nn
	case 1:
		// CMP
		fmt.Sprintf("cmp r%d, 0x%04x\n", rd, nn)
		result = uint64(g.R[rd]) - uint64(nn)
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(g.R[rd], uint32(nn), uint32(result)))
	case 2:
		// ADD
		fmt.Sprintf("add r%d, 0x%04x\n", rd, nn)
		result = uint64(g.R[rd]) + uint64(nn)
		g.R[rd] = g.R[rd] + nn
		g.SetCPSRFlag(flagC, util.AddC(result))
		g.SetCPSRFlag(flagV, util.AddV(lhs, uint32(nn), uint32(result)))
	case 3:
		// SUB
		fmt.Sprintf("sub r%d, 0x%04x\n", rd, nn)
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
	opcode := (inst >> 6) & 0b1111

	result := uint64(0)
	mnemonic := "unknown"
	switch opcode {
	case 0:
		// AND
		mnemonic = "and"
		g.R[rd] = g.R[rd] & g.R[rs]
		result = uint64(g.R[rd])
	case 1:
		// EOR(xor)
		mnemonic = "eor"
		g.R[rd] = g.R[rd] ^ g.R[rs]
		result = uint64(g.R[rd])
	case 2:
		// LSL
		mnemonic = "lsl"
		g.R[rd] = g.armLSL(g.R[rd], g.R[rs]&0xff, true) // Rd = Rd << (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 3:
		// LSR
		mnemonic = "lsr"
		g.R[rd] = g.armLSR(g.R[rd], g.R[rs]&0xff, true) // Rd = Rd >> (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 4:
		// ASR
		mnemonic = "asr"
		g.R[rd] = g.armASR(g.R[rd], g.R[rs]&0xff, true) // Rd = Rd >> (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 5:
		// ADC
		mnemonic = "adc"
		result = uint64(g.R[rd]) + uint64(g.R[rs]) + uint64(util.BoolToInt(g.GetCPSRFlag(flagC)))
		g.R[rd] = g.R[rd] + g.R[rs] + uint32(util.BoolToInt(g.GetCPSRFlag(flagC))) // Rd = Rd + Rs + Carry
		g.SetCPSRFlag(flagC, util.AddC(result))
		g.SetCPSRFlag(flagV, util.AddV(lhs, rhs, uint32(result)))
	case 6:
		// SBC
		mnemonic = "sbc"
		result = uint64(g.R[rd]) - uint64(g.R[rs]) - uint64(util.BoolToInt(!g.GetCPSRFlag(flagC)))
		g.R[rd] = g.R[rd] - g.R[rs] - uint32(util.BoolToInt(!g.GetCPSRFlag(flagC))) // Rd = Rd - Rs - NOT Carry
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(lhs, rhs, uint32(result)))
	case 7:
		// ROR
		mnemonic = "ror"
		g.R[rd] = g.armROR(g.R[rd], g.R[rs]&0xff, true) // Rd = Rd ROR (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 8:
		mnemonic = "tst"
		result = uint64(g.R[rd] & g.R[rs]) // TST Rd,Rs
	case 9:
		mnemonic = "neg"
		rhs := g.R[rs]
		result = 0 - uint64(g.R[rs])
		g.R[rd] = -g.R[rs] // Rd = -Rs
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(0, rhs, g.R[rd]))
	case 10:
		// CMP
		mnemonic = "cmp"
		result = uint64(g.R[rd]) - uint64(g.R[rs]) // Void = Rd - Rs
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(g.R[rd], g.R[rs], uint32(result)))
	case 11:
		mnemonic = "cmn"
		result = uint64(g.R[rd]) + uint64(g.R[rs]) // Void = Rd + Rs
		g.SetCPSRFlag(flagC, util.AddC(result))
		g.SetCPSRFlag(flagV, util.AddV(g.R[rd], g.R[rs], uint32(result)))
	case 12:
		mnemonic = "orr"
		g.R[rd] = g.R[rd] | g.R[rs]
		result = uint64(g.R[rd])
	case 13:
		mnemonic = "mul"
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
		mnemonic = "bic"
		g.R[rd] = g.R[rd] & ^g.R[rs] // BIC{S} Rd,Rs
		result = uint64(g.R[rd])
	case 15:
		mnemonic = "mvn"
		g.R[rd] = ^g.R[rs]
		result = uint64(g.R[rd])
	}
	fmt.Sprintf("%s r%d, r%d\n", mnemonic, rd, rs)

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
	case 0:
		// ADD Rd,Rs(Rd = Rd+Rs)
		fmt.Sprintf("add r%d, r%d\n", rd, rs)
		g.R[rd] = rdval + rsval
	case 1:
		// CMP Rd,Rs(Void = Rd-Rs)
		fmt.Sprintf("cmp r%d, r%d\n", rd, rs)
		result := uint64(rdval) - uint64(rsval)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
		g.SetCPSRFlag(flagZ, uint32(result) == 0)
		g.SetCPSRFlag(flagC, util.SubC(result))
		g.SetCPSRFlag(flagV, util.SubV(rdval, rsval, uint32(result)))
	case 2:
		// MOV Rd,Rs(Rd=Rs)
		fmt.Sprintf("mov r%d, r%d\n", rd, rs)
		g.R[rd] = rsval
	case 3:
		// BX Rs(PC = Rs)
		rd = 15
		fmt.Sprintf("bx r%d\n", rs)
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
	fmt.Sprintf("ldr r%d, [pc, #%d]\n", rd, nn)
	pc := util.Align4(g.inst.loc + 4)
	g.R[rd] = g.getRAM32(pc+nn, false)
	g.timer(1)
}

func (g *GBA) thumbLoadStoreRegOfs(inst uint16) {
	ro, rb, rd := (inst>>6)&0b111, (inst>>3)&0b111, inst&0b111

	opcode := (inst >> 10) & 0b11
	mnemonic := "unknown"
	switch opcode {
	case 0:
		// STR Rd,[Rb,Ro]
		mnemonic = "str"
		g.setRAM32(g.R[rb]+g.R[ro], g.R[rd], false) // N
		g.timer(g.cycleS2N())                       // -S + 2N
	case 1:
		mnemonic = "strb"
		g.setRAM8(g.R[rb]+g.R[ro], byte(g.R[rd]), false) // STRB Rd,[Rb,Ro] (BYTE[Rb+Ro] = Rd)
		g.timer(g.cycleS2N())
	case 2:
		mnemonic = "ldr"
		g.R[rd] = g.getRAM32(g.R[rb]+g.R[ro], false) // LDR Rd,[Rb,Ro] (Rd = WORD[Rb+Ro])
		g.timer(1)
	case 3:
		// LDRB Rd,[Rb,Ro]
		mnemonic = "ldrb"
		g.R[rd] = uint32(g.getRAM8(g.R[rb]+g.R[ro], false))
		g.timer(1)
	}
	fmt.Sprintf("%s r%d, [r%d, r%d]\n", mnemonic, rd, rb, ro)
}

func (g *GBA) thumbLoadStoreSBH(inst uint16) {
	ro, rb, rd := uint32((inst>>6)&0b111), (inst>>3)&0b111, inst&0b111

	opcode := (inst >> 10) & 0b11
	mnemonic := "unknown"
	switch opcode {
	case 0:
		// STRH Rd,[Rb,Ro]
		mnemonic = "strh"
		g.setRAM16(g.R[rb]+g.R[ro], uint16(g.R[rd]), false)
		g.timer(g.cycleS2N())
	case 1:
		// LDSB Rd,[Rb,Ro]
		mnemonic = "ldsb"
		value := int32(g.getRAM8(g.R[rb]+g.R[ro], false))
		value <<= 24
		value >>= 24
		g.R[rd] = uint32(value)
		g.timer(1)
	case 2:
		// LDRH Rd,[Rb,Ro]
		mnemonic = "ldrh"
		g.R[rd] = uint32(g.getRAM16(g.R[rb]+g.R[ro], false))
		g.timer(1)
	case 3:
		// LDSH Rd,[Rb,Ro]
		mnemonic = "ldsh"
		value := int32(g.getRAM16(g.R[rb]+g.R[ro], false))
		value <<= 16
		value >>= 16
		g.R[rd] = uint32(value)
		g.timer(1)
	}
	fmt.Sprintf("%s r%d, [r%d, r%d]\n", mnemonic, rd, rb, ro)
}

func (g *GBA) thumbLoadStoreImmOfs(inst uint16) {
	nn, rb, rd := uint32((inst>>6)&0b11111), (inst>>3)&0b111, inst&0b111

	opcode := (inst >> 11) & 0b11
	mnemonic := "unknown"
	switch opcode {
	case 0:
		// STR Rd,[Rb,#nn]
		mnemonic = "str"
		nn *= 4
		g.setRAM32(g.R[rb]+nn, g.R[rd], false)
		g.timer(g.cycleS2N())
	case 1:
		// LDR Rd,[Rb,#nn]
		mnemonic = "ldr"
		nn *= 4
		g.R[rd] = g.getRAM32(g.R[rb]+nn, false)
		g.timer(1)
	case 2:
		// STRB Rd,[Rb,#nn]
		mnemonic = "strb"
		g.setRAM8(g.R[rb]+nn, byte(g.R[rd]), false)
		g.timer(g.cycleS2N())
	case 3:
		// LDRB Rd,[Rb,#nn]
		mnemonic = "ldrb"
		g.R[rd] = uint32(g.getRAM8(g.R[rb]+nn, false))
		if g.inst.loc == 0x13c {

		}
		g.timer(1)
	}
	fmt.Sprintf("%s r%d, [r%d, #%d]\n", mnemonic, rd, rb, nn)
}

func (g *GBA) thumbLoadStoreH(inst uint16) {
	nn, rb, rd := uint32(((inst>>6)&0b11111)*2), (inst>>3)&0b111, inst&0b111

	opcode := (inst >> 11) & 0b1
	switch opcode {
	case 0:
		fmt.Sprintf("strh r%d, [r%d, #%d]\n", rd, rb, nn)
		g.setRAM16(g.R[rb]+nn, uint16(g.R[rd]), false) // STRH Rd,[Rb,#nn]
		g.timer(g.cycleS2N())
	case 1:
		fmt.Sprintf("ldrh r%d, [r%d, #%d]\n", rd, rb, nn)
		g.R[rd] = uint32(g.getRAM16(g.R[rb]+nn, false)) // LDRH Rd,[Rb,#nn]
		g.timer(1)
	}
}

func (g *GBA) thumbLoadSPRel(inst uint16) {
	rd, nn := (inst>>8)&0b111, uint32((inst&0b1111_1111)*4)

	sp, opcode := g.R[13], (inst>>11)&0b1
	switch opcode {
	case 0:
		fmt.Sprintf("str r%d, [sp, #%d]\n", rd, nn)
		g.setRAM32(sp+nn, g.R[rd], false)
		g.timer(g.cycleS2N())
	case 1:
		fmt.Sprintf("ldr r%d, [sp, #%d]\n", rd, nn)
		g.R[rd] = g.getRAM32(sp+nn, false)
		g.timer(1)
	}
}

// thumbStack push, pop
func (g *GBA) thumbStack(inst uint16) {
	rlist := inst & 0b1111_1111

	opcode := (inst >> 11) & 0b1
	switch opcode {
	case 0:
		n := 0
		lr := util.Bit(inst, 8)
		if lr {
			g.R[13] -= 4
			g.setRAM32(g.R[13], g.R[14], n > 0) // PUSH lr
			n++
		}
		for i := 7; i >= 0; i-- {
			if util.ToBool(rlist & (0b1 << i)) {
				g.R[13] -= 4
				g.setRAM32(g.R[13], g.R[i], n > 0) // PUSH
				n++
			}
		}
		g.timer(g.cycleS2N())
	case 1:
		n := 0
		for i := 0; i < 8; i++ {
			if util.ToBool(rlist & (0b1 << i)) {
				g.R[i] = g.getRAM32(g.R[13], n > 0) // POP
				g.R[13] += 4
				n++
			}
		}
		pc := util.Bit(inst, 8)
		if pc {
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
	opcode := (inst >> 11) & 0b1
	switch opcode {
	case 0:
		fmt.Sprintf("stmia r%d!, {", rb)
		n := 0
		for i := 0; i < 8; i++ {
			if util.ToBool(rlist & (0b1 << i)) {
				g.setRAM32(g.R[rb], g.R[i], n > 0) // STMIA
				g.R[rb] += 4
				n++
			}
		} // (n-1)S + N
		g.timer(g.cycleS2N()) // (n-2)S + 2N
	case 1:
		fmt.Sprintf("ldmia r%d!, {", rb)
		n := 0
		for i := 0; i < 8; i++ {
			if util.ToBool(rlist & (0b1 << i)) {
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
	opcode := (inst >> 11) & 0b1
	switch opcode {
	case 0:
		fmt.Sprintf("add r%d, pc, #%d\n", rd, nn)
		g.R[rd] = (util.Align4(g.inst.loc + 4)) + nn // ADD  Rd,PC,#nn
	case 1:
		fmt.Sprintf("add r%d, sp, #%d\n", rd, nn)
		g.R[rd] = g.R[13] + nn // ADD  Rd,SP,#nn
	}
}

func (g *GBA) thumbMoveSP(inst uint16) {
	nn := uint32((inst & 0b0111_1111) * 4)
	opcode := (inst >> 7) & 0b1
	switch opcode {
	case 0:
		fmt.Sprintf("add sp, #%d\n", nn)
		g.R[13] += nn // ADD SP,#nn
	case 1:
		fmt.Sprintf("add sp, #-%d\n", nn)
		g.R[13] -= nn // ADD SP,#-nn
	}
}

func (g *GBA) thumbCondBranch(inst uint16) {
	cond := Cond((inst >> 8) & 0b1111)
	if g.Check(cond) {
		nn := int8(byte(inst & 0b1111_1111))
		if nn > 0 {
			g.R[15] = g.inst.loc + 4 + uint32(nn)*2
		} else {
			g.R[15] = g.inst.loc + 4 - uint32(-nn)*2
		}
		fmt.Sprintf("b%s 0x%04x\n", cond, g.R[15])
		g.pipelining()
	}
}

func (g *GBA) thumbSWI(inst uint16) {
	nn := byte(inst)
	g.printSWI(nn)
	g.exception(swiVec, SWI)
}

func (g *GBA) thumbB(inst uint16) {
	nn := int32(inst)
	nn <<= 21
	nn >>= 20

	if nn > 0 {
		g.R[15] = g.inst.loc + 4 + uint32(nn)
	} else {
		g.R[15] = g.inst.loc + 4 - uint32(-nn)
	}
	fmt.Sprintf("b 0x%04x\n", g.R[15])
	g.pipelining()
}

func (g *GBA) thumbLinkBranch1(inst uint16) {
	nn := int32(inst)
	nn <<= 21
	nn >>= 9
	g.R[14] = g.inst.loc + 4
	if nn > 0 {
		g.R[14] += uint32(nn)
	} else {
		g.R[14] -= uint32(-nn)
	}
}

func (g *GBA) thumbLinkBranch2(inst uint16) {
	nn := inst & 0b0111_1111_1111
	g.R[15] = g.R[14] + uint32(nn<<1)
	g.R[14] = g.inst.loc + 2 // return
	if g.R[14]&1 == 0 {
		g.R[14]++
	}

	fmt.Sprintf("bl 0x%04x\n", g.R[15])
	g.pipelining()
}
