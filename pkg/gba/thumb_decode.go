package gba

// IsThumbShift returns instruction is shift instruction
func IsThumbShift(inst uint16) bool {
	cond1 := inst&0b1110_0000_0000_0000 == 0b0000_0000_0000_0000    // 15-13: 0b000
	cond2 := !(inst&0b0001_1000_0000_0000 == 0b0001_1000_0000_0000) // not 12-11: 0b11
	return cond1 && cond2
}

// IsThumbAddSub returns instruction is add or sub
// 15-11: 00011
func IsThumbAddSub(inst uint16) bool {
	return inst&0b1111_1000_0000_0000 == 0b0001_1000_0000_0000
}

// IsThumbMovCmpAddSub returns instruction is move or cmp or add or sub
// 15-13: 001
func IsThumbMovCmpAddSub(inst uint16) bool {
	return inst&0b1110_0000_0000_0000 == 0b0010_0000_0000_0000
}

// IsThumbALU returns instruction is ALU operation
// 15-10: 0100_00
func IsThumbALU(inst uint16) bool {
	return inst&0b1111_1100_0000_0000 == 0b0100_0000_0000_0000
}

// IsHiRegisterBX returns instruction is hi register operation or bx
// 15-10: 0100_01
func IsHiRegisterBX(inst uint16) bool {
	return inst&0b1111_1100_0000_0000 == 0b0100_0100_0000_0000
}

// IsThumbLoadPCRel returns instruction is load PC-relative instruction
// 15-11: 0100_1
func IsThumbLoadPCRel(inst uint16) bool {
	return inst&0b1111_1000_0000_0000 == 0b0100_1000_0000_0000
}

// IsThumbLoadStoreRegOfs returns instruction is load/store with register offset instruction
// 15-12: 0101 && 9: 0
func IsThumbLoadStoreRegOfs(inst uint16) bool {
	return inst&0b1111_0010_0000_0000 == 0b0101_0000_0000_0000
}

// IsThumbLoadStoreSBH returns instruction is load/store sign-extended byte/halfword instruction
// 15-12: 0101 && 9: 1
func IsThumbLoadStoreSBH(inst uint16) bool {
	return inst&0b1111_0010_0000_0000 == 0b0101_0010_0000_0000
}

// IsThumbLoadStoreImmOfs returns instruction is load/store with immediate offset instruction
// 15-13: 011
func IsThumbLoadStoreImmOfs(inst uint16) bool {
	return inst&0b1110_0000_0000_0000 == 0b0110_0000_0000_0000
}

// IsThumbLoadStoreH returns instruction is load/store halfword instruction
// 15-12: 1000
func IsThumbLoadStoreH(inst uint16) bool {
	return inst&0b1111_0000_0000_0000 == 0b1000_0000_0000_0000
}

// IsThumbLoadSPRel returns instruction is load/store SP-relative instruction
// 15-12: 1001
func IsThumbLoadSPRel(inst uint16) bool {
	return inst&0b1111_0000_0000_0000 == 0b1001_0000_0000_0000
}

// IsThumbStack returns instruction is push/pop instruction
// 15-12: 1011 & 10-9: 10
func IsThumbStack(inst uint16) bool {
	return inst&0b1111_0110_0000_0000 == 0b1011_0100_0000_0000
}

// IsThumbStackMultiple returns instruction is ldm/stm instruction
// 15-12: 1100
func IsThumbStackMultiple(inst uint16) bool {
	return inst&0b1111_0000_0000_0000 == 0b1100_0000_0000_0000
}

// IsThumbGetAddr returns instruction is get relative-address instruction
// 15-12: 1010
func IsThumbGetAddr(inst uint16) bool {
	return inst&0b1111_0000_0000_0000 == 0b1010_0000_0000_0000
}

// IsThumbMoveSP returns instruction is stack movement instruction
// 15-8: 1011_0000
func IsThumbMoveSP(inst uint16) bool {
	return inst&0b1111_1111_0000_0000 == 0b1011_0000_0000_0000
}

// IsThumbCondBranch returns instruction is conditional branch instruction
// 15-12: 1101
func IsThumbCondBranch(inst uint16) bool {
	cond1 := inst&0b1111_0000_0000_0000 == 0b1101_0000_0000_0000 // 15-12: 1101
	cond2 := ((inst >> 8) & 0b1111) < 14                         // 14: bkpt(unused), 15: swi(below)
	return cond1 && cond2
}

// IsThumbSWI returns instruction is swi
// 15-8: 1101_1111
func IsThumbSWI(inst uint16) bool {
	return inst&0b1111_1111_0000_0000 == 0b1101_1111_0000_0000
}

// IsThumbB returns instruction is b(jump)
// 15-11: 1110_0
func IsThumbB(inst uint16) bool {
	return inst&0b1111_1000_0000_0000 == 0b1110_0000_0000_0000
}

// IsThumbLinkBranch1 returns instruction is the first of long branch with link
// 15-11: 1111_0
func IsThumbLinkBranch1(inst uint16) bool {
	return inst&0b1111_1000_0000_0000 == 0b1111_0000_0000_0000
}

// IsThumbLinkBranch2 returns instruction is the second of long branch with link
// 15-11: 1111_1
func IsThumbLinkBranch2(inst uint16) bool {
	return inst&0b1111_1000_0000_0000 == 0b1111_1000_0000_0000
}
