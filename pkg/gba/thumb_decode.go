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

// IsHiRegisterBX returns instruction is hi register operation or bx
func IsHiRegisterBX(inst uint16) bool {
	return !util.Bit(inst, 15) && util.Bit(inst, 14) && !util.Bit(inst, 13) && !util.Bit(inst, 12) && !util.Bit(inst, 11) && util.Bit(inst, 10) // 0b010001
}

// IsThumbLoadPCRel returns instruction is load PC-relative instruction
func IsThumbLoadPCRel(inst uint16) bool {
	return !util.Bit(inst, 15) && util.Bit(inst, 14) && !util.Bit(inst, 13) && !util.Bit(inst, 12) && util.Bit(inst, 11) // 0b01001
}

// IsThumbLoadStoreRegOfs returns instruction is load/store with register offset instruction
func IsThumbLoadStoreRegOfs(inst uint16) bool {
	cond1 := !util.Bit(inst, 15) && util.Bit(inst, 14) && !util.Bit(inst, 13) && util.Bit(inst, 12) // 0b0101
	cond2 := !util.Bit(inst, 9)
	return cond1 && cond2
}

// IsThumbLoadStoreSBH returns instruction is load/store sign-extended byte/halfword instruction
func IsThumbLoadStoreSBH(inst uint16) bool {
	cond1 := !util.Bit(inst, 15) && util.Bit(inst, 14) && !util.Bit(inst, 13) && util.Bit(inst, 12) // 0b0101
	cond2 := util.Bit(inst, 9)
	return cond1 && cond2
}

// IsThumbLoadStoreImmOfs returns instruction is load/store with immediate offset instruction
func IsThumbLoadStoreImmOfs(inst uint16) bool {
	return !util.Bit(inst, 15) && util.Bit(inst, 14) && util.Bit(inst, 13) // 0b011
}

// IsThumbLoadStoreH returns instruction is load/store halfword instruction
func IsThumbLoadStoreH(inst uint16) bool {
	return util.Bit(inst, 15) && !util.Bit(inst, 14) && !util.Bit(inst, 13) && !util.Bit(inst, 12) // 0b1000
}

// IsThumbLoadSPRel returns instruction is load/store SP-relative instruction
func IsThumbLoadSPRel(inst uint16) bool {
	return util.Bit(inst, 15) && !util.Bit(inst, 14) && !util.Bit(inst, 13) && util.Bit(inst, 12) // 0b1001
}

// IsThumbStack returns instruction is push/pop instruction
func IsThumbStack(inst uint16) bool {
	cond1 := util.Bit(inst, 15) && !util.Bit(inst, 14) && util.Bit(inst, 13) && util.Bit(inst, 12) // 0b1011
	cond2 := util.Bit(inst, 10) && !util.Bit(inst, 9)
	return cond1 && cond2
}

// IsThumbStackMultiple returns instruction is ldm/stm instruction
func IsThumbStackMultiple(inst uint16) bool {
	return util.Bit(inst, 15) && util.Bit(inst, 14) && !util.Bit(inst, 13) && !util.Bit(inst, 12) // 0b1100
}

// IsThumbGetAddr returns instruction is get relative-address instruction
func IsThumbGetAddr(inst uint16) bool {
	return util.Bit(inst, 15) && !util.Bit(inst, 14) && util.Bit(inst, 13) && !util.Bit(inst, 12) // 0b1010
}

// IsThumbMoveSP returns instruction is stack movement instruction
func IsThumbMoveSP(inst uint16) bool {
	return util.Bit(inst, 15) && !util.Bit(inst, 14) && util.Bit(inst, 13) && util.Bit(inst, 12) && !util.Bit(inst, 11) && !util.Bit(inst, 10) && !util.Bit(inst, 9) && !util.Bit(inst, 8)
}

// IsThumbCondBranch returns instruction is conditional branch instruction
func IsThumbCondBranch(inst uint16) bool {
	cond1 := util.Bit(inst, 15) && util.Bit(inst, 14) && !util.Bit(inst, 13) && util.Bit(inst, 12) // 0b1101
	cond2 := ((inst >> 8) & 0b1111) < 14                                                           // 14: bkpt(unused), 15: swi(below)
	return cond1 && cond2
}

// IsThumbSWI returns instruction is swi
func IsThumbSWI(inst uint16) bool {
	return util.Bit(inst, 15) && util.Bit(inst, 14) && !util.Bit(inst, 13) && util.Bit(inst, 12) && util.Bit(inst, 11) && util.Bit(inst, 10) && util.Bit(inst, 9) && util.Bit(inst, 8) // 0b1101_1111
}

// IsThumbB returns instruction is b(jump)
func IsThumbB(inst uint16) bool {
	return util.Bit(inst, 15) && util.Bit(inst, 14) && util.Bit(inst, 13) && !util.Bit(inst, 12) && !util.Bit(inst, 11) // 0b11100
}

// IsThumbLinkBranch1 returns instruction is the first of long branch with link
func IsThumbLinkBranch1(inst uint16) bool {
	return util.Bit(inst, 15) && util.Bit(inst, 14) && util.Bit(inst, 13) && util.Bit(inst, 12) && !util.Bit(inst, 11) // 0b11110
}

// IsThumbLinkBranch2 returns instruction is the second of long branch with link
func IsThumbLinkBranch2(inst uint16) bool {
	return util.Bit(inst, 15) && util.Bit(inst, 14) && util.Bit(inst, 13) && util.Bit(inst, 11) // 0b111x1
}
