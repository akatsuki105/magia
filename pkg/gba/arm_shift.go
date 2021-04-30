package gba

import (
	"mettaur/pkg/util"
)

func (g *GBA) armLSL(val uint32, is uint32, carryVariable bool, imm bool) uint32 {
	switch {
	case is == 0 && imm:
		return val
	case is > 32:
		if carryVariable {
			g.SetCPSRFlag(flagC, false)
		}
		return 0
	default:
		carry := val&(1<<(32-is)) > 0
		if is > 0 && carryVariable {
			g.SetCPSRFlag(flagC, carry)
		}
		return util.LSL(val, uint(is))
	}
}

func (g *GBA) armLSR(val uint32, is uint32, carryVariable bool, imm bool) uint32 {
	if is == 0 && imm {
		is = 32
	}
	carry := val&(1<<(is-1)) > 0
	if is > 0 && carryVariable {
		g.SetCPSRFlag(flagC, carry)
	}
	return util.LSR(val, uint(is))
}

func (g *GBA) armASR(val uint32, is uint32, carryVariable bool, imm bool) uint32 {
	if is == 0 && imm {
		is = 32
	}
	if is > 32 {
		is = 32
	}
	carry := val&(1<<(is-1)) > 0
	if is > 0 && carryVariable {
		g.SetCPSRFlag(flagC, carry)
	}
	return util.ASR(val, uint(is))
}

func (g *GBA) armROR(val uint32, is uint32, carryVariable bool, imm bool) uint32 {
	if is == 0 && imm {
		c := uint32(0)
		if g.GetCPSRFlag(flagC) {
			c = 1
		}
		g.SetCPSRFlag(flagC, util.Bit(val, 0))
		return util.ROR(((val & ^(uint32(1))) | c), 1)
	}
	carry := (val>>(is-1))&0b1 > 0
	if is > 0 && carryVariable {
		g.SetCPSRFlag(flagC, carry)
	}
	return util.ROR(val, uint(is))
}
