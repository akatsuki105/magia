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
