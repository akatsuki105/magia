package gba

import (
	"github.com/pokemium/magia/pkg/util"
)

type Carry struct {
	carry   uint32
	mutable bool
	set     func(bool)
}

func lslArm(val uint32, is uint32, carry Carry, imm bool) uint32 {
	switch {
	case is == 0 && imm:
		return val
	case is > 32:
		if carry.mutable {
			carry.set(false)
		}
		return 0
	default:
		c := val&(1<<(32-is)) > 0
		if is > 0 && carry.mutable {
			carry.set(c)
		}
		return util.LSL(val, uint(is))
	}
}

func lsrArm(val uint32, is uint32, carry Carry, imm bool) uint32 {
	if is == 0 && imm {
		is = 32
	}
	c := val&(1<<(is-1)) > 0
	if is > 0 && carry.mutable {
		carry.set(c)
	}
	return util.LSR(val, uint(is))
}

func asrArm(val uint32, is uint32, carry Carry, imm bool) uint32 {
	if (is == 0 && imm) || is > 32 {
		is = 32
	}
	c := val&(1<<(is-1)) > 0
	if is > 0 && carry.mutable {
		carry.set(c)
	}
	return util.ASR(val, uint(is))
}

func rorArm(val uint32, is uint32, carry Carry, imm bool) uint32 {
	if is == 0 && imm {
		carry.set(util.Bit(val, 0))
		return util.ROR(((val & ^(uint32(1))) | carry.carry), 1)
	}
	c := (val>>(is-1))&0b1 > 0
	if is > 0 && carry.mutable {
		carry.set(c)
	}
	return util.ROR(val, uint(is))
}
