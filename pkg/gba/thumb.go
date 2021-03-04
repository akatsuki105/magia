package gba

import (
	"mettaur/pkg/ram"
	"mettaur/pkg/util"
)

func (g *GBA) thumbStep() {
	inst := g.thumbFetch()
	g.thumbExec(inst)
}

func (g *GBA) thumbFetch() uint16 {
	pc := g.R[15]
	g.PC = pc

	switch {
	case ram.BIOS(pc) || ram.IWRAM(pc) || ram.IO(pc) || ram.OAM(pc):
		g.timer(1)
	case ram.Palette(pc) || ram.VRAM(pc):
		g.timer(1)
	case ram.EWRAM(pc):
		g.timer(3)
	case ram.GamePak0(pc) || ram.GamePak1(pc) || ram.GamePak2(pc):
		if g.lastAddr+2 == pc {
			// sequential
			g.timer(g.cycleS(pc))
		} else {
			// non-sequential
			g.timer(g.cycleN(pc))
		}
	case ram.SRAM(pc):
		g.timer(5 * 2) // 8bit * 2
	}

	g.R[15] += 2 // Note that when reading R15, this will usually return a value of PC+2 because of read-ahead (pipelining).
	return uint16(g.RAM.Get(pc))
}

func (g *GBA) thumbExec(inst uint16) {
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
	}
}

func (g *GBA) thumbShift(inst uint16) {
	is, rs, rd := uint32((inst>>6)&0b11111), (inst>>3)&0b111, inst&0b111
	switch opcode := (inst >> 11) & 0b11; opcode {
	case 0:
		g.R[rd] = g.armLSL(g.R[rs], is)
	case 1:
		g.R[rd] = g.armLSR(g.R[rs], is)
	case 2:
		g.R[rd] = g.armASR(g.R[rs], is)
	}

	g.SetCPSRFlag(flagZ, g.R[rd] == 0)
	g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
}

func (g *GBA) thumbAddSub(inst uint16) {
	delta, rs, rd := (inst>>6)&0b111, (inst>>3)&0b111, inst&0b111
	switch opcode := (inst >> 9) & 0b11; opcode {
	case 0:
		g.R[rd] = g.R[rs] + g.R[delta]
		result := uint64(g.R[rs]) + uint64(g.R[delta])
		g.SetCPSRFlag(flagC, result > 0xffff_ffff)
		g.SetCPSRFlag(flagV, util.AddV(g.R[rs], g.R[delta], uint32(result)))
	case 1:
		g.R[rd] = g.R[rs] - g.R[delta]
		result := uint64(g.R[rs]) - uint64(g.R[delta])
		g.SetCPSRFlag(flagC, result < 0x1_0000_0000)
		g.SetCPSRFlag(flagV, util.SubV(g.R[rs], g.R[delta], uint32(result)))
	case 2:
		g.R[rd] = g.R[rs] + uint32(delta)
		result := uint64(g.R[rs]) + uint64(delta)
		g.SetCPSRFlag(flagC, result > 0xffff_ffff)
		g.SetCPSRFlag(flagV, util.AddV(g.R[rs], uint32(delta), uint32(result)))
	case 3:
		g.R[rd] = g.R[rs] - uint32(delta)
		result := uint64(g.R[rs]) - uint64(delta)
		g.SetCPSRFlag(flagC, result < 0x1_0000_0000)
		g.SetCPSRFlag(flagV, util.SubV(g.R[rs], uint32(delta), uint32(result)))
	}

	g.SetCPSRFlag(flagN, util.Bit(g.R[rd], 31))
	g.SetCPSRFlag(flagZ, g.R[rd] == 0)
	g.timer(g.cycleS(g.R[15]))
}

func (g *GBA) thumbMovCmpAddSub(inst uint16) {
	rd, nn := (inst>>8)&0b111, uint32(inst&0b1111_1111)
	lhs := g.R[rd]
	result := uint64(0)
	switch opcode := (inst >> 11) & 0b11; opcode {
	case 0:
		result = uint64(nn)
		g.R[rd] = nn
	case 1:
		result = uint64(g.R[rd]) - uint64(nn)
		g.SetCPSRFlag(flagC, result < 0x1_0000_0000)
		g.SetCPSRFlag(flagV, util.SubV(g.R[rd], uint32(nn), uint32(result)))
	case 2:
		result = uint64(g.R[rd]) + uint64(nn)
		g.R[rd] = g.R[rd] + nn
		g.SetCPSRFlag(flagC, result > 0xffff_ffff)
		g.SetCPSRFlag(flagV, util.AddV(g.R[rd], uint32(nn), uint32(result)))
	case 3:
		result = uint64(g.R[rd]) - uint64(nn)
		g.R[rd] = g.R[rd] - nn
		g.SetCPSRFlag(flagC, result < 0x1_0000_0000)
		g.SetCPSRFlag(flagV, util.SubV(lhs, uint32(nn), uint32(result)))
	}

	g.SetCPSRFlag(flagN, util.Bit(result, 31))
	g.SetCPSRFlag(flagZ, result == 0)

	g.timer(g.cycleS(g.R[15]))
}

func (g *GBA) thumbALU(inst uint16) {
	rs, rd := (inst>>3)&0b111, inst&0b111
	lhs := g.R[rd]
	opcode := (inst >> 11) & 0b11

	result := uint64(0)
	switch opcode {
	case 0:
		g.R[rd] = g.R[rd] & g.R[rs] // Rd = Rd AND Rs
		result = uint64(g.R[rd])
	case 1:
		g.R[rd] = g.R[rd] ^ g.R[rs] // Rd = Rd XOR Rs
		result = uint64(g.R[rd])
	case 2:
		g.R[rd] = g.armLSL(g.R[rd], g.R[rs]&0xff) // Rd = Rd << (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 3:
		g.R[rd] = g.armLSR(g.R[rd], g.R[rs]&0xff) // Rd = Rd >> (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 4:
		g.R[rd] = g.armASR(g.R[rd], g.R[rs]&0xff) // Rd = Rd >> (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 5:
		result = uint64(g.R[rd]) + uint64(g.R[rs]) + uint64(util.BoolToInt(g.GetCPSRFlag(flagC)))
		g.R[rd] = g.R[rd] + g.R[rs] + uint32(util.BoolToInt(g.GetCPSRFlag(flagC))) // Rd = Rd + Rs + Carry
		g.SetCPSRFlag(flagC, result > 0xffff_ffff)
		g.SetCPSRFlag(flagV, util.ToBool(^(lhs^g.R[rs])&(lhs^uint32(result))&0x8000_0000))
	case 6:
		result = uint64(g.R[rd]) - uint64(g.R[rs]) + uint64(util.BoolToInt(!g.GetCPSRFlag(flagC)))
		g.R[rd] = g.R[rd] - g.R[rs] + uint32(util.BoolToInt(!g.GetCPSRFlag(flagC))) // Rd = Rd - Rs - NOT Carry
		g.SetCPSRFlag(flagC, result < 0x1_0000_0000)
		g.SetCPSRFlag(flagV, util.SubV(lhs, g.R[rs], uint32(result)))
	case 7:
		g.R[rd] = g.armROR(g.R[rd], g.R[rs]&0xff) // Rd = Rd ROR (Rs AND 0FFh)
		result = uint64(g.R[rd])
		g.timer(1)
	case 8:
		result = uint64(g.R[rd] & g.R[rs]) // TST Rd,Rs
	case 9:
		g.R[rd] = -g.R[rs] // Rd = -Rs
		result = uint64(-g.R[rs])
		g.SetCPSRFlag(flagC, result < 0x1_0000_0000)
		g.SetCPSRFlag(flagV, util.SubV(0, g.R[rs], g.R[rd]))
	case 10:
		result = uint64(g.R[rd]) - uint64(g.R[rs]) // Void = Rd - Rs
		g.SetCPSRFlag(flagC, result < 0x1_0000_0000)
		g.SetCPSRFlag(flagV, util.SubV(g.R[rd], g.R[rs], uint32(result)))
	case 11:
		result = uint64(g.R[rd]) + uint64(g.R[rs]) // Void = Rd + Rs
		g.SetCPSRFlag(flagC, result > 0xffff_ffff)
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
	g.SetCPSRFlag(flagZ, result == 0)
	g.timer(g.cycleS(g.R[15]))
}

func (g *GBA) thumbHiRegisterBX(inst uint16) {
	rs, rd := (inst>>3)&0b111, inst&0b111
	if util.Bit(inst, 7) {
		rd += 8
	}
	if util.Bit(inst, 6) {
		rs += 8
	}

	opcode := (inst >> 8) & 0b11
	switch opcode {
	case 0:
		g.R[rd] = g.R[rd] + g.R[rs] // ADD Rd,Rs(Rd = Rd+Rs)
	case 1:
		result := uint64(g.R[rd]) - uint64(g.R[rs]) // CMP Rd,Rs(Void = Rd-Rs)
		g.SetCPSRFlag(flagN, util.Bit(result, 31))
		g.SetCPSRFlag(flagZ, result == 0)
		g.SetCPSRFlag(flagC, result < 0x1_0000_0000)
		g.SetCPSRFlag(flagV, util.SubV(g.R[rd], g.R[rs], uint32(result)))
	case 2:
		g.R[rd] = g.R[rs] // MOV Rd,Rs(Rd=Rs)
	case 3:
		// BX Rs(PC = Rs)
		rd = 15
		if util.Bit(g.R[rs], 0) {
			g.R[15] = g.R[rs]
		} else {
			g.SetCPSRFlag(flagT, false) // switch to ARM
			g.R[15] = util.Align4(g.R[rs])
		}
	}

	if opcode != 1 && rd == 15 {
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
	}
	g.timer(g.cycleS(g.R[15]))
}

func (g *GBA) thumbLoadPCRel(inst uint16) {
	rd, nn := (inst>>8)&0b111, uint32(inst&0b1111_1111)*4
	pc := util.Align4(g.PC + 4)
	g.R[rd] = g.RAM.Get(pc + nn)
	g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
}

func (g *GBA) thumbLoadStoreRegOfs(inst uint16) {
	ro, rb, rd := (inst>>6)&0b111, (inst>>3)&0b111, inst&0b111

	opcode := (inst >> 10) & 0b11
	switch opcode {
	case 0:
		g.RAM.Set32(g.R[rb]+g.R[ro], g.R[rd]) // STR Rd,[Rb,Ro] (WORD[Rb+Ro] = Rd)
		g.timer(2 * g.cycleN(g.R[15]))
	case 1:
		g.RAM.Set8(g.R[rb]+g.R[ro], byte(g.R[rd])) // STRB Rd,[Rb,Ro] (BYTE[Rb+Ro] = Rd)
		g.timer(2 * g.cycleN(g.R[15]))
	case 2:
		g.R[rd] = g.RAM.Get(g.R[rb] + g.R[ro]) // LDR Rd,[Rb,Ro] (Rd = WORD[Rb+Ro])
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	case 3:
		g.R[rd] = uint32(byte(g.RAM.Get(g.R[rb] + g.R[ro]))) // LDRB Rd,[Rb,Ro] (Rd = BYTE[Rb+Ro])
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	}
}

func (g *GBA) thumbLoadStoreSBH(inst uint16) {
	ro, rb, rd := uint32((inst>>6)&0b111), (inst>>3)&0b111, inst&0b111

	opcode := (inst >> 10) & 0b11
	switch opcode {
	case 0:
		g.RAM.Set16(g.R[rb]+g.R[ro], uint16(g.R[rd])) // STRH Rd,[Rb,Ro]
		g.timer(2 * g.cycleN(g.R[15]))
	case 1:
		g.R[rd] = uint32(int8(g.RAM.Get(g.R[rb] + g.R[ro]))) // LDSB Rd,[Rb,Ro]
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	case 2:
		g.R[rd] = uint32(uint16(g.RAM.Get(g.R[rb] + g.R[ro]))) // LDRH Rd,[Rb,Ro]
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	case 3:
		g.R[rd] = uint32(int16(g.RAM.Get(g.R[rb] + g.R[ro]))) // LDSH Rd,[Rb,Ro]
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	}
}

func (g *GBA) thumbLoadStoreImmOfs(inst uint16) {
	nn, rb, rd := uint32((inst>>6)&0b11111), (inst>>3)&0b111, inst&0b111

	opcode := (inst >> 11) & 0b11
	switch opcode {
	case 0:
		g.RAM.Set32(g.R[rb]+nn*4, g.R[rd]) // STR Rd,[Rb,Ro] (WORD[Rb+Ro] = Rd)
		g.timer(2 * g.cycleN(g.R[15]))
	case 1:
		g.R[rd] = g.RAM.Get(g.R[rb] + nn*4) // LDR Rd,[Rb,Ro] (Rd = WORD[Rb+Ro])
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	case 2:
		g.RAM.Set8(g.R[rb]+nn, byte(g.R[rd])) // STRB Rd,[Rb,Ro] (BYTE[Rb+Ro] = Rd)
		g.timer(2 * g.cycleN(g.R[15]))
	case 3:
		g.R[rd] = uint32(byte(g.RAM.Get(g.R[rb] + nn))) // LDRB Rd,[Rb,Ro] (Rd = BYTE[Rb+Ro])
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	}
}

func (g *GBA) thumbLoadStoreH(inst uint16) {
	nn, rb, rd := uint32(((inst>>6)&0b11111)*2), (inst>>3)&0b111, inst&0b111

	opcode := (inst >> 11) & 0b1
	switch opcode {
	case 0:
		g.RAM.Set16(g.R[rb]+nn, uint16(g.R[rd])) // STRH Rd,[Rb,#nn]
		g.timer(2 * g.cycleN(g.R[15]))
	case 1:
		g.R[rd] = uint32(uint16(g.RAM.Get(g.R[rb] + nn))) // LDRH Rd,[Rb,#nn]
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	}
}

func (g *GBA) thumbLoadSPRel(inst uint16) {
	rd, nn := (inst>>8)&0b111, uint32((inst&0b1111_1111)*4)

	sp, opcode := g.R[13], (inst>>11)&0b1
	switch opcode {
	case 0:
		g.RAM.Set32(sp+nn, g.R[rd])
		g.timer(2 * g.cycleN(g.R[15]))
	case 1:
		g.R[rd] = g.RAM.Get(sp + nn)
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	}
}

// thumbStack push, pop
func (g *GBA) thumbStack(inst uint16) {
	rlist := inst & 0b1111_1111

	opcode := (inst >> 11) & 0b1
	switch opcode {
	case 0:
		n := 0
		for i := 0; i < 8; i++ {
			if util.ToBool(rlist & (0b1 << i)) {
				g.RAM.Set32(g.R[13], g.R[i]) // PUSH
				g.R[13] -= 4
				n++
			}
		}
		lr := util.Bit(inst, 8)
		if lr {
			g.RAM.Set32(g.R[13], g.R[14]) // PUSH lr
			g.R[13] -= 4
			n++
		}
		g.timer((n-1)*g.cycleS(g.R[15]) + 2*g.cycleN(g.R[15]))
	case 1:
		n := 0
		for i := 0; i < 8; i++ {
			if util.ToBool(rlist & (0b1 << i)) {
				g.R[i] = g.RAM.Get(g.R[13]) // POP
				g.R[13] += 4
				n++
			}
		}
		pc := util.Bit(inst, 8)
		if pc {
			g.R[15] = g.RAM.Get(g.R[13]) // POP pc
			g.R[13] += 4
			g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
		}
		g.timer(n*g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	}
}

// thumbStackMultiple ldmia, stmia
func (g *GBA) thumbStackMultiple(inst uint16) {
	rb, rlist := (inst>>8)&0b111, inst&0b1111_1111
	opcode := (inst >> 11) & 0b1
	switch opcode {
	case 0:
		n := 0
		for i := 0; i < 8; i++ {
			if util.ToBool(rlist & (0b1 << i)) {
				g.RAM.Set32(g.R[rb], g.R[i]) // STMIA
				g.R[rb] += 4
				n++
			}
		}
		g.timer((n-1)*g.cycleS(g.R[15]) + 2*g.cycleN(g.R[15]))
	case 1:
		n := 0
		for i := 0; i < 8; i++ {
			if util.ToBool(rlist & (0b1 << i)) {
				g.R[i] = g.RAM.Get(g.R[rb]) // LDMIA
				g.R[rb] += 4
				n++
			}
		}
		g.timer(n*g.cycleS(g.R[15]) + g.cycleN(g.R[15]) + 1)
	}
}

// thumbGetAddr get relative address
func (g *GBA) thumbGetAddr(inst uint16) {
	rd, nn := (inst>>8)&0b111, uint32((inst&0b1111_1111)*4)
	opcode := (inst >> 11) & 0b1
	switch opcode {
	case 0:
		g.R[rd] = util.Align4(g.PC+4) + nn // ADD  Rd,PC,#nn
	case 1:
		g.R[rd] = g.R[13] + nn // ADD  Rd,SP,#nn
	}
	g.timer(g.cycleS(g.R[15]))
}

func (g *GBA) thumbMoveSP(inst uint16) {
	nn := uint32((inst & 0b0111_111) * 4)
	opcode := (inst >> 7) & 0b1
	switch opcode {
	case 0:
		g.R[13] += nn // ADD SP,#nn
	case 1:
		g.R[13] -= nn // ADD SP,#-nn
	}
	g.timer(g.cycleS(g.R[15]))
}

func (g *GBA) thumbCondBranch(inst uint16) {
	cond := Cond((inst >> 8) & 0b1111)
	if g.Check(cond) {
		nn := int8((inst & 0b1111_1111) * 2)
		if nn > 0 {
			g.R[15] = g.PC + uint32(nn)
		} else {
			g.R[15] = g.PC - uint32(-nn)
		}
		g.timer(g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
	}
	g.timer(g.cycleS(g.R[15]))
}

func (g *GBA) thumbSWI(inst uint16) {}

func (g *GBA) thumbB(inst uint16) {
	nn := uint32(inst & 0b0111_1111_1111)
	g.R[15] = g.PC + nn
	g.timer(2*g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
}

func (g *GBA) thumbLinkBranch1(inst uint16) {
	nn := uint32(inst & 0b0111_1111_1111)
	g.R[14] = g.PC + 4 + (nn << 12)
	g.timer(g.cycleS(g.R[15]))
}

func (g *GBA) thumbLinkBranch2(inst uint16) {
	opcode, nn := (inst>>11)&0b11111, inst&0b0111_1111_1111
	g.R[15] = g.R[14] + uint32(nn<<1)
	g.R[14] = g.PC + 2 // return
	// BLX
	if opcode == 0b11101 {
		g.SetCPSRFlag(flagT, false)
	}
	g.timer(2*g.cycleS(g.R[15]) + g.cycleN(g.R[15]))
}
