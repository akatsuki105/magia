package util

// LSL logical left shift
func LSL(val uint32, shiftAmount uint) uint32 { return val << shiftAmount }

// LSR logical right shift
func LSR(val uint32, shiftAmount uint) uint32 { return val >> shiftAmount }

// ASR arithmetic right shift
func ASR(val uint32, shiftAmount uint) uint32 {
	msb := val & 0x8000_0000
	for i := uint(0); i < shiftAmount; i++ {
		val = (val >> 1) | msb
	}
	return val
}

// ROR rotate val's bit by shift
func ROR(val uint32, shiftAmount uint) uint32 {
	tmp0 := (val) >> (shiftAmount)        // XX00YY -> 00XX00
	tmp1 := (val) << (32 - (shiftAmount)) // XX00YY -> YY0000
	return tmp0 | tmp1                    // YYXX00
}
