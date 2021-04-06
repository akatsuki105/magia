package gba

import (
	"fmt"
	"mettaur/pkg/util"
	"strings"
)

func armDecode(pc, inst uint32) string {
	switch {
	case IsArmSWI(inst):
		nn := byte(inst >> 16)
		return fmt.Sprintf("swi 0x%x", nn)
	case IsArmBranch(inst) || IsArmBX(inst):
		return armDecodeBranch(pc, inst)
	case IsArmLDM(inst) || IsArmSTM(inst):
		return armDecodeStack(inst)
	case IsArmLDR(inst) || IsArmSTR(inst):
		return armDecodeLDRSTR(inst)
	case IsArmLDRH(inst) || IsArmLDRSB(inst) || IsArmLDRSH(inst) || IsArmSTRH(inst):
		return armDecodeLDRSTR2(inst)
	case IsArmMRS(inst):
		return armDecodeMRS(inst)
	case IsArmMSR(inst):
		return armDecodeMSR(inst)
	case IsArmSWP(inst):
		return fmt.Sprintf("SWI is unsupported in 0x%08x", pc)
	case IsArmMPY(inst):
		return armDecodeMPY(inst)
	case IsArmALU(inst):
		return armDecodeALU(inst)
	default:
		return fmt.Sprintf("invalid ARM opcode(0x%08x) in 0x%08x\n", inst, pc)
	}
}

func armDecodeBranch(pc, inst uint32) string {
	switch {
	case IsArmBX(inst):
		rn := inst & 0b1111
		return fmt.Sprintf("bx r%d", rn)
	case util.Bit(inst, 24):
		nn := int32(inst)
		nn <<= 8
		nn >>= 6
		if nn >= 0 {
			return fmt.Sprintf("bl 0x%08x", pc+8+uint32(nn))
		} else {
			return fmt.Sprintf("bl 0x%08x", pc+8-uint32(-nn))
		}
	default:
		nn := int32(inst)
		nn <<= 8
		nn >>= 6
		if nn >= 0 {
			return fmt.Sprintf("b 0x%08x", pc+8+uint32(nn))
		} else {
			return fmt.Sprintf("b 0x%08x", pc+8-uint32(-nn))
		}
	}
}

func armDecodeStack(inst uint32) string {
	p, u := util.Bit(inst, 24), util.Bit(inst, 23)
	rn := inst >> 16 & 0b1111

	opcode := "stm"
	if util.Bit(inst, 20) {
		opcode = "ldm"
	}
	switch {
	case p && u: // IB
		opcode += "ib"
	case !p && u: // IA, pop
		opcode += "ia"
	case p && !u: // DB
		opcode += "db"
	case !p && !u: // DA
		opcode += "da"
	}

	rlist := "{"
	for rs := 0; rs < 16; rs++ {
		if util.Bit(inst, rs) {
			rlist += fmt.Sprintf("r%d, ", rs)
		}
	}
	rlist = strings.TrimSuffix(rlist, ", ") + "}"

	writeBack := ""
	if util.Bit(inst, 21) {
		writeBack = "!"
	}
	return fmt.Sprintf("%s r%d%s, %s", opcode, rn, writeBack, rlist)
}

func armDecodeLDRSTR(inst uint32) string {
	opcode := "str"
	if util.Bit(inst, 20) {
		opcode = "ldr"
	}
	if util.Bit(inst, 22) {
		opcode += "b"
	}

	plus := "-"
	if util.Bit(inst, 23) {
		plus = "+"
	}

	rn, rd := (inst>>16)&0b1111, (inst>>12)&0b1111

	ofs := ""
	if util.Bit(inst, 25) {
		is := inst >> 7 & 0b11111 // I = 1 shift reg
		shiftType := inst >> 5 & 0b11
		rm := inst & 0b1111
		switch shiftType {
		case lsl:
			ofs = fmt.Sprintf("%sr%d, lsl#%d", plus, rm, is)
		case lsr:
			ofs = fmt.Sprintf("%sr%d, lsr#%d", plus, rm, is)
		case asr:
			ofs = fmt.Sprintf("%sr%d, asr#%d", plus, rm, is)
		case ror:
			ofs = fmt.Sprintf("%sr%d, ror#%d", plus, rm, is)
		}
	} else {
		ofs = fmt.Sprintf("#%s0x%x", plus, inst&0b1111_1111_1111) // I = 0 immediate
	}

	pre := util.Bit(inst, 24)
	if pre {
		return fmt.Sprintf("%s r%d,[r%d,%s]", opcode, rd, rn, ofs)
	} else {
		return fmt.Sprintf("%s r%d,[r%d],%s", opcode, rd, rn, ofs)
	}
}

func armDecodeLDRSTR2(inst uint32) string {
	opcode := "unsupported ldrstr2"
	tmp, isLoad := (inst>>5)&0b11, util.Bit(inst, 20)
	switch {
	case tmp == 1 && !isLoad:
		opcode = "strh"
	case tmp == 2 && isLoad:
		opcode = "ldrd"
	case tmp == 3 && !isLoad:
		opcode = "strd"
	case tmp == 1 && isLoad:
		opcode = "ldrh"
	case tmp == 2 && isLoad:
		opcode = "ldrsb"
	case tmp == 3 && isLoad:
		opcode = "ldrsh"
	}

	plus := "-"
	if util.Bit(inst, 23) {
		plus = "+"
	}

	ofs := ""
	if util.Bit(inst, 22) {
		ofs = fmt.Sprintf("#%s0x%x", plus, (((inst>>8)&0b1111)<<4)|(inst&0b1111))
	} else {
		ofs = fmt.Sprintf("%sr%d", plus, inst&0b1111)
	}

	rn, rd := (inst>>16)&0b1111, (inst>>12)&0b1111
	pre := util.Bit(inst, 24)
	if pre {
		return fmt.Sprintf("%s r%d, [r%d,%s]", opcode, rd, rn, ofs)
	} else {
		return fmt.Sprintf("%s r%d, [r%d], %s", opcode, rd, rn, ofs)
	}
}

func armDecodeMRS(inst uint32) string {
	// useSpsr := (inst>>22)&0b1 > 0
	rd := (inst >> 12) & 0b1111
	return fmt.Sprintf("mrs %d,psr", rd)
}
func armDecodeMSR(inst uint32) string {
	psr := "cpsr"
	if util.Bit(inst, 22) {
		psr = "spsr"
	}

	if c := util.Bit(inst, 16); c {
		psr += "_c"
	}
	if x := util.Bit(inst, 17); x {
		psr += "x"
	}
	if s := util.Bit(inst, 18); s {
		psr += "s"
	}
	if f := util.Bit(inst, 19); f {
		psr += "f"
	}

	if util.Bit(inst, 25) {
		// register Psr[field] = Imm
		is, imm := ((inst>>8)&0b1111)*2, inst&0b1111_1111
		return fmt.Sprintf("msr %s,#%xror#%d", psr, imm, is)
	} else {
		// immediate Psr[field] = Rm
		rm := inst & 0b1111
		return fmt.Sprintf("msr %s,r%d", psr, rm)
	}
}

func armDecodeMPY(inst uint32) string {
	opcode := inst >> 21 & 0b1111
	switch opcode {
	case 0b0000:
		return fmt.Sprintf("mul r%d,r%d,r%d", inst>>16&0b1111, inst&0b1111, inst>>8&0b1111)
	case 0b0001:
		rd, rn, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
		return fmt.Sprintf("mla r%d,r%d,r%d,r%d", rd, rm, rs, rn)
	case 0b0010:
		return fmt.Sprintf("UMAAL is unsupported")
	case 0b0100:
		rdHi, rdLo, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
		return fmt.Sprintf("umull r%d,r%d,r%d,r%d", rdLo, rdHi, rm, rs)
	case 0b0101:
		rdHi, rdLo, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
		return fmt.Sprintf("umlal r%d,r%d,r%d,r%d", rdLo, rdHi, rm, rs)
	case 0b0110:
		rdHi, rdLo, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
		return fmt.Sprintf("smull r%d,r%d,r%d,r%d", rdLo, rdHi, rm, rs)
	case 0b0111:
		rdHi, rdLo, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
		return fmt.Sprintf("smlal r%d,r%d,r%d,r%d", rdLo, rdHi, rm, rs)
	default:
		return fmt.Sprintf("invalid opcode(%d) is unsupported", opcode)
	}
}

func armDecodeALU(inst uint32) string {
	opcode := ""
	switch inst >> 21 & 0b1111 {
	case 0x0:
		opcode = "and"
	case 0x1:
		opcode = "eor"
	case 0x2:
		opcode = "sub"
	case 0x3:
		opcode = "rsb"
	case 0x4:
		opcode = "add"
	case 0x5:
		opcode = "adc"
	case 0x6:
		opcode = "sbc"
	case 0x7:
		opcode = "rsc"
	case 0x8:
		opcode = "tst"
	case 0x9:
		opcode = "teq"
	case 0xa:
		opcode = "cmp"
	case 0xb:
		opcode = "cmn"
	case 0xc:
		opcode = "orr"
	case 0xd:
		opcode = "mov"
	case 0xe:
		opcode = "bic"
	case 0xf:
		opcode = "mvn"
	}

	rd := inst >> 12 & 0b1111
	rn := (inst >> 16) & 0b1111
	op2 := ""
	if !util.Bit(inst, 25) {
		// register
		is := (inst >> 7) & 0b11111
		rm := inst & 0b1111

		shift := "lsl"
		switch shiftType := (inst >> 5) & 0b11; shiftType {
		case lsr:
			shift = "lsr"
		case asr:
			shift = "asr"
		case ror:
			shift = "ror"
		}

		isRegister := (inst>>4)&0b1 > 0
		if isRegister {
			rs := (inst >> 8) & 0b1111
			op2 = fmt.Sprintf("r%d,%s r%d", rm, shift, rs)
		} else {
			op2 = fmt.Sprintf("r%d,%s#%d", rm, shift, is)
		}
	} else {
		is := uint((inst>>8)&0b1111) * 2
		op2 = fmt.Sprintf("#0x%x", util.ROR(inst&0b1111_1111, is))
	}

	switch inst >> 21 & 0b1111 {
	case 0, 1, 2, 3, 4, 5, 6, 7, 0xc, 0xe:
		return fmt.Sprintf("%s r%d,r%d,%s", opcode, rd, rn, op2)
	case 8, 9, 0xa, 0xb:
		return fmt.Sprintf("%s r%d,%s", opcode, rn, op2)
	default:
		return fmt.Sprintf("%s r%d,%s", opcode, rd, op2)
	}
}
