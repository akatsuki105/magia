package gba

import "mettaur/pkg/util"

// IsThumbShift returns instruction is shift instruction
func IsThumbShift(inst uint16) bool {
	cond1 := !util.Bit(inst, 15) && !util.Bit(inst, 14) && !util.Bit(inst, 13) // 0b000
	cond2 := !(util.Bit(inst, 12) && util.Bit(inst, 11))                       // not 0b11
	return cond1 && cond2
}

// IsThumbAddSub returns instruction is add or sub
func IsThumbAddSub(inst uint16) bool {
	return !util.Bit(inst, 15) && !util.Bit(inst, 14) && !util.Bit(inst, 13) && util.Bit(inst, 12) && util.Bit(inst, 11) // 0b00011
}

// IsThumbMovCmpAddSub returns instruction is move or cmp or add or sub
func IsThumbMovCmpAddSub(inst uint16) bool {
	return !util.Bit(inst, 15) && !util.Bit(inst, 14) && util.Bit(inst, 13) // 0b001
}

// IsThumbALU returns instruction is ALU operation
func IsThumbALU(inst uint16) bool {
	return !util.Bit(inst, 15) && util.Bit(inst, 14) && !util.Bit(inst, 13) && !util.Bit(inst, 12) && !util.Bit(inst, 11) && !util.Bit(inst, 10) // 0b010000
}

// IsHiRegister returns instruction is hi register operation or bx
func IsHiRegisterBX(inst uint16) bool {
	return !util.Bit(inst, 15) && util.Bit(inst, 14) && !util.Bit(inst, 13) && !util.Bit(inst, 12) && !util.Bit(inst, 11) && util.Bit(inst, 10) // 0b010001
}
