package gba

func isThumbShift(inst uint16) bool {
	cond1 := inst&0b1110_0000_0000_0000 == 0b0000_0000_0000_0000    // 15-13: 0b000
	cond2 := !(inst&0b0001_1000_0000_0000 == 0b0001_1000_0000_0000) // not 12-11: 0b11
	return cond1 && cond2
}

// 15-11: 00011
func isThumbAddSub(inst uint16) bool { return inst&0b1111_1000_0000_0000 == 0b0001_1000_0000_0000 }

// 15-13: 001
func isThumbMovCmpAddSub(inst uint16) bool {
	return inst&0b1110_0000_0000_0000 == 0b0010_0000_0000_0000
}

// 15-10: 0100_00
func isThumbALU(inst uint16) bool { return inst&0b1111_1100_0000_0000 == 0b0100_0000_0000_0000 }

// 15-10: 0100_01
func isHiRegisterBX(inst uint16) bool { return inst&0b1111_1100_0000_0000 == 0b0100_0100_0000_0000 }

// 15-11: 0100_1
func isThumbLoadPCRel(inst uint16) bool { return inst&0b1111_1000_0000_0000 == 0b0100_1000_0000_0000 }

// 15-12: 0101 && 9: 0
func isThumbLoadStoreRegOfs(inst uint16) bool {
	return inst&0b1111_0010_0000_0000 == 0b0101_0000_0000_0000
}

// 15-12: 0101 && 9: 1
func isThumbLoadStoreSBH(inst uint16) bool {
	return inst&0b1111_0010_0000_0000 == 0b0101_0010_0000_0000
}

// 15-13: 011
func isThumbLoadStoreImmOfs(inst uint16) bool {
	return inst&0b1110_0000_0000_0000 == 0b0110_0000_0000_0000
}

// 15-12: 1000
func isThumbLoadStoreH(inst uint16) bool { return inst&0b1111_0000_0000_0000 == 0b1000_0000_0000_0000 }

// 15-12: 1001
func isThumbLoadSPRel(inst uint16) bool { return inst&0b1111_0000_0000_0000 == 0b1001_0000_0000_0000 }

// 15-12: 1011 & 10-9: 10
func isThumbStack(inst uint16) bool { return inst&0b1111_0110_0000_0000 == 0b1011_0100_0000_0000 }

// 15-12: 1100
func isThumbStackMultiple(inst uint16) bool {
	return inst&0b1111_0000_0000_0000 == 0b1100_0000_0000_0000
}

// 15-12: 1010
func isThumbGetAddr(inst uint16) bool { return inst&0b1111_0000_0000_0000 == 0b1010_0000_0000_0000 }

// 15-8: 1011_0000
func isThumbMoveSP(inst uint16) bool { return inst&0b1111_1111_0000_0000 == 0b1011_0000_0000_0000 }

// 15-12: 1101
func isThumbCondBranch(inst uint16) bool {
	cond1 := inst&0b1111_0000_0000_0000 == 0b1101_0000_0000_0000 // 15-12: 1101
	cond2 := ((inst >> 8) & 0b1111) < 14                         // 14: bkpt(unused), 15: swi(below)
	return cond1 && cond2
}

// 15-8: 1101_1111
func isThumbSWI(inst uint16) bool { return inst&0b1111_1111_0000_0000 == 0b1101_1111_0000_0000 }

// 15-11: 1110_0
func isThumbB(inst uint16) bool { return inst&0b1111_1000_0000_0000 == 0b1110_0000_0000_0000 }

// 15-11: 1111_0
func isThumbLinkBranch1(inst uint16) bool { return inst&0b1111_1000_0000_0000 == 0b1111_0000_0000_0000 }

// 15-11: 1111_1
func isThumbLinkBranch2(inst uint16) bool { return inst&0b1111_1000_0000_0000 == 0b1111_1000_0000_0000 }
