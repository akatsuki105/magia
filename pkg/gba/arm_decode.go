package gba

import "mettaur/pkg/util"

// IsArmALU returns instruction is arithmetic instruction
// 27-26: 00
func IsArmALU(inst uint32) bool {
	return !util.Bit(inst, 27) && !util.Bit(inst, 26) // 00
}

// branch

// IsArmBranch returns instruction is either b or bl
// 27-25: 101
func IsArmBranch(inst uint32) bool {
	return util.Bit(inst, 27) && !util.Bit(inst, 26) && util.Bit(inst, 25) // 101
}

// IsArmBX returns instruction is bx
func IsArmBX(inst uint32) bool {
	mask := uint32(0b0000_0001_0010_1111_1111_1111_0001_0000)
	return inst&mask == mask
}

// IsArmSWI returns instruction is `SWI{cond} nn`
// 27-24: 1111
func IsArmSWI(inst uint32) bool {
	return util.Bit(inst, 27) && util.Bit(inst, 26) && util.Bit(inst, 25) && util.Bit(inst, 24) // 1111
}

// IsArmUND returns instruction is UND
// 27-25: 011
func IsArmUND(inst uint32) bool {
	return !util.Bit(inst, 27) && util.Bit(inst, 26) && util.Bit(inst, 25) && util.Bit(inst, 4) // 011
}

// multiply

// IsArmMPY returns instruction is word-multiply instruction
// 27-25: 000
// 7-4: 1001
func IsArmMPY(inst uint32) bool {
	upper := !util.Bit(inst, 27) && !util.Bit(inst, 26) && !util.Bit(inst, 25)
	lower := util.Bit(inst, 7) && !util.Bit(inst, 6) && !util.Bit(inst, 5) && util.Bit(inst, 4) // 1001
	return upper && lower
}

// IsArmMPY16 returns instruction is halfword-multiply instruction
// 27-25: 000
// 20: 0
// 7: 1
// 4: 0
func IsArmMPY16(inst uint32) bool {
	upper := !util.Bit(inst, 27) && !util.Bit(inst, 26) && !util.Bit(inst, 25)
	return upper && !util.Bit(inst, 20) && util.Bit(inst, 7) && !util.Bit(inst, 4)
}

// loadstore

// IsArmLDR returns instruction is Load instruction
// 27-26: 01
func IsArmLDR(inst uint32) bool {
	return !util.Bit(inst, 27) && util.Bit(inst, 26) && util.Bit(inst, 20)
}

// IsArmSTR returns instruction is Store instruction
// 27-26: 01
func IsArmSTR(inst uint32) bool {
	return !util.Bit(inst, 27) && util.Bit(inst, 26) && !util.Bit(inst, 20)
}

// laodstore2

// IsArmLDRH returns instruction is Load-Halfword instruction
// 27-25: 000
func IsArmLDRH(inst uint32) bool {
	cond1 := !util.Bit(inst, 27) && !util.Bit(inst, 26) && !util.Bit(inst, 25)
	cond2 := !util.Bit(inst, 11) && !util.Bit(inst, 10) && !util.Bit(inst, 9) && !util.Bit(inst, 8) && util.Bit(inst, 7)
	cond3 := util.Bit(inst, 4)
	condLDRH := util.Bit(inst, 20) && !util.Bit(inst, 6) && util.Bit(inst, 5)
	return cond1 && cond2 && cond3 && condLDRH
}

// IsArmLDRSB returns instruction is Load-Sign-Byte instruction
// 27-25: 000
func IsArmLDRSB(inst uint32) bool {
	cond1 := !util.Bit(inst, 27) && !util.Bit(inst, 26) && !util.Bit(inst, 25)
	cond2 := !util.Bit(inst, 11) && !util.Bit(inst, 10) && !util.Bit(inst, 9) && !util.Bit(inst, 8) && util.Bit(inst, 7)
	cond3 := util.Bit(inst, 4)
	condLDRSB := util.Bit(inst, 20) && util.Bit(inst, 6) && !util.Bit(inst, 5)
	return cond1 && cond2 && cond3 && condLDRSB
}

// IsArmLDRSH returns instruction is Load-Sign-Halfword instruction
// 27-25: 000
func IsArmLDRSH(inst uint32) bool {
	cond1 := !util.Bit(inst, 27) && !util.Bit(inst, 26) && !util.Bit(inst, 25)
	cond2 := !util.Bit(inst, 11) && !util.Bit(inst, 10) && !util.Bit(inst, 9) && !util.Bit(inst, 8) && util.Bit(inst, 7)
	cond3 := util.Bit(inst, 4)
	condLDRSH := util.Bit(inst, 20) && util.Bit(inst, 6) && util.Bit(inst, 5)
	return cond1 && cond2 && cond3 && condLDRSH
}

// IsArmSTRH returns instruction is Store-Halfword instruction
// 27-25: 000
func IsArmSTRH(inst uint32) bool {
	cond1 := !util.Bit(inst, 27) && !util.Bit(inst, 26) && !util.Bit(inst, 25)
	cond2 := !util.Bit(inst, 11) && !util.Bit(inst, 10) && !util.Bit(inst, 9) && !util.Bit(inst, 8) && util.Bit(inst, 7)
	cond3 := util.Bit(inst, 4)
	condSTRH := !util.Bit(inst, 20) && !util.Bit(inst, 6) && util.Bit(inst, 5)
	return cond1 && cond2 && cond3 && condSTRH
}

// laodstore3

// IsArmStack returns instruction is push/pop instruction
// 27-25: 100
func IsArmStack(inst uint32) bool {
	return util.Bit(inst, 27) && !util.Bit(inst, 26) && !util.Bit(inst, 25) // 100
}

// psr

// IsArmMRS returns instruction is ???
// 27-23: 00010
func IsArmMRS(inst uint32) bool {
	cond1 := !util.Bit(inst, 27) && !util.Bit(inst, 26) && !util.Bit(inst, 25) // 000
	cond2 := util.Bit(inst, 24) && !util.Bit(inst, 23)                         // 10
	return cond1 && cond2 && !util.Bit(inst, 21)
}

// IsArmMSR returns instruction is ???
// 27-26: 00
// 24-23: 10
func IsArmMSR(inst uint32) bool {
	cond1 := !util.Bit(inst, 27) && !util.Bit(inst, 26) // 00
	cond2 := util.Bit(inst, 24) && !util.Bit(inst, 23)  // 10
	return cond1 && cond2 && util.Bit(inst, 21)
}
