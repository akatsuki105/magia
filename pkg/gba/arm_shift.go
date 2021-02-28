package gba

import "mettaur/pkg/util"

func (g *GBA) armLSL(val uint32, is uint32) uint32 {
	switch {
	case is == 0:
		return val
	case is > 32:
		g.SetCPSRFlag(flagC, false)
		return 0
	default:
		carry := util.ToBool(val >> (32 - is) & 0b1)
		g.SetCPSRFlag(flagC, carry)
		return util.LSL(val, uint(is))
	}
}

func (g *GBA) armLSR(val uint32, is uint32) uint32 {
	if is == 0 {
		is = 32
	}
	carry := util.ToBool(val >> (is - 1) & 0b1)
	g.SetCPSRFlag(flagC, carry)
	return util.LSR(val, uint(is))
}

func (g *GBA) armASR(val uint32, is uint32) uint32 {
	if is == 0 {
		is = 32
	}
	carry := util.ToBool(val >> (is - 1) & 0b1)
	g.SetCPSRFlag(flagC, carry)
	return util.ASR(val, uint(is))
}

func (g *GBA) armROR(val uint32, is uint32) uint32 {
	if is == 0 {
		is = 1
	}
	carry := util.ToBool(val >> (is - 1) & 0b1)
	g.SetCPSRFlag(flagC, carry)
	return util.ROR(val, uint(is))
}
