package gba

// IsArmALU returns instruction is arithmetic instruction
// 27-26: 00
func IsArmALU(inst uint32) bool {
	return inst&0b0000_1100_0000_0000_0000_0000_0000_0000 == 0
}

// branch

// IsArmBranch returns instruction is either b or bl
// 27-25: 101
func IsArmBranch(inst uint32) bool {
	return inst&0b0000_1110_0000_0000_0000_0000_0000_0000 == 0b0000_1010_0000_0000_0000_0000_0000_0000
}

// IsArmBX returns instruction is bx
func IsArmBX(inst uint32) bool {
	mask := uint32(0b0000_0001_0010_1111_1111_1111_0001_0000)
	cond1 := inst&mask == mask
	cond2 := (inst>>4)&0b1111 == 0b1
	return cond1 && cond2
}

// IsArmSWI returns instruction is `SWI{cond} nn`
// 27-24: 1111
func IsArmSWI(inst uint32) bool {
	return inst&0b0000_1111_0000_0000_0000_0000_0000_0000 == 0b0000_1111_0000_0000_0000_0000_0000_0000
}

// IsArmUND returns instruction is UND
// 27-25: 011
func IsArmUND(inst uint32) bool {
	return inst&0b0000_1110_0000_0000_0000_0000_0000_0000 == 0b0000_0110_0000_0000_0000_0000_0000_0000
}

// multiply

// IsArmMPY returns instruction is word-multiply instruction
// 27-25: 000 & 7-4: 1001
func IsArmMPY(inst uint32) bool {
	return inst&0b0000_1110_0000_0000_0000_0000_1111_0000 == 0b0000_0000_0000_0000_0000_0000_1001_0000
}

// IsArmMPY16 returns instruction is halfword-multiply instruction
// 27-25: 000 & 20: 0 & 7: 1 & 4: 0
func IsArmMPY16(inst uint32) bool {
	return inst&0b0000_1110_0001_0000_0000_0000_1001_0000 == 0b0000_0000_0000_0000_0000_0000_1000_0000
}

// loadstore

// IsArmLDR returns instruction is Load instruction
// 27-26: 01
func IsArmLDR(inst uint32) bool {
	return inst&0b0000_1100_0001_0000_0000_0000_0000_0000 == 0b0000_0100_0001_0000_0000_0000_0000_0000
}

// IsArmSTR returns instruction is Store instruction
// 27-26: 01
func IsArmSTR(inst uint32) bool {
	return inst&0b0000_1100_0001_0000_0000_0000_0000_0000 == 0b0000_0100_0000_0000_0000_0000_0000_0000
}

// laodstore2

// IsArmLDRH returns instruction is Load-Halfword instruction
// 27-25: 000 & 20: 1 & 7-4: 1011
func IsArmLDRH(inst uint32) bool {
	return inst&0b0000_1110_0001_0000_0000_0000_1111_0000 == 0b0000_0000_0001_0000_0000_0000_1011_0000
}

// IsArmLDRSB returns instruction is Load-Sign-Byte instruction
// 27-25: 000 & 20: 1 & 7-4: 1101
func IsArmLDRSB(inst uint32) bool {
	return inst&0b0000_1110_0001_0000_0000_0000_1111_0000 == 0b0000_0000_0001_0000_0000_0000_1101_0000
}

// IsArmLDRSH returns instruction is Load-Sign-Halfword instruction
// 27-25: 000 & 20: 1 & 7-4: 1111
func IsArmLDRSH(inst uint32) bool {
	return inst&0b0000_1110_0001_0000_0000_0000_1111_0000 == 0b0000_0000_0001_0000_0000_0000_1111_0000
}

// IsArmSTRH returns instruction is Store-Halfword instruction
// 27-25: 000 & 20: 0 & 7-4: 1011
func IsArmSTRH(inst uint32) bool {
	return inst&0b0000_1110_0001_0000_0000_0000_1111_0000 == 0b0000_0000_0000_0000_0000_0000_1011_0000
}

// laodstore3

// IsArmStack returns instruction is push/pop instruction
// 27-25: 100 & 20: 1
func IsArmLDM(inst uint32) bool {
	return inst&0b0000_1110_0001_0000_0000_0000_0000_0000 == 0b0000_1000_0001_0000_0000_0000_0000_0000
}

// 27-25: 100 & 20: 0
func IsArmSTM(inst uint32) bool {
	return inst&0b0000_1110_0001_0000_0000_0000_0000_0000 == 0b0000_1000_0000_0000_0000_0000_0000_0000
}

// data swap
// 27-23: 0001_0 & 21-20: 00 & 11-4: 0000_1001
func IsArmSWP(inst uint32) bool {
	return inst&0b0000_1111_1011_0000_0000_1111_1111_0000 == 0b0000_0001_0000_0000_0000_0000_1001_0000
}

// psr

// IsArmMRS returns instruction is `Move the contents of a PSR to a general-purpose register`
// 27-23: 0001_0 & 21-16: 00_1111 & 11-0: 0000_0000_0000
func IsArmMRS(inst uint32) bool {
	return inst&0b0000_1111_1011_1111_0000_1111_1111_1111 == 0b0000_0001_0000_1111_0000_0000_0000_0000
}

// IsArmMSR returns instruction is `Move to system coprocessor register from ARM register`
// 27-26: 00 & 24-23: 10 & 21-20: 10 & 15-12: 1111
func IsArmMSR(inst uint32) bool {
	return inst&0b0000_1101_1011_0000_1111_0000_0000_0000 == 0b0000_0001_0010_0000_1111_0000_0000_0000
}
