package debug

import (
	"fmt"
	"strings"

	"github.com/pokemium/magia/pkg/gba"
	"github.com/pokemium/magia/pkg/util"
)

const (
	lsl = iota
	lsr
	asr
	ror
)

func DissasembleArm(pc, inst uint32) string {
	switch {
	case gba.IsArmSWI(inst):
		nn := byte(inst >> 16)
		return fmt.Sprintf("swi 0x%x", nn)
	case gba.IsArmB(inst) || gba.IsArmBL(inst) || gba.IsArmBX(inst):
		return disassembleArmBranch(pc, inst)
	case gba.IsArmLDM(inst) || gba.IsArmSTM(inst):
		return disassembleArmStack(inst)
	case gba.IsArmLDR(inst) || gba.IsArmSTR(inst):
		return disassembleArmLDRSTR(inst)
	case gba.IsArmLDRH(inst) || gba.IsArmLDRSB(inst) || gba.IsArmLDRSH(inst) || gba.IsArmSTRH(inst):
		return disassembleArmLDRSTR2(inst)
	case gba.IsArmMRS(inst):
		return disassembleArmMRS(inst)
	case gba.IsArmMSR(inst):
		return disassembleArmMSR(inst)
	case gba.IsArmSWP(inst):
		return disassembleArmSWP(inst)
	case gba.IsArmMPY(inst):
		return disassembleArmMPY(inst)
	case gba.IsArmALU(inst):
		return disassembleArmALU(inst)
	default:
		return fmt.Sprintf("invalid ARM opcode(0x%08x) in 0x%08x\n", inst, pc)
	}
}

func disassembleArmBranch(pc, inst uint32) string {
	cond := gba.Cond(inst >> 28)
	switch {
	case gba.IsArmBX(inst):
		rn := inst & 0b1111
		return fmt.Sprintf("bx%s r%d", cond, rn)
	case util.Bit(inst, 24):
		nn := int32(inst)
		nn = (nn << 8) >> 6
		return fmt.Sprintf("bl%s 0x%08x", cond, util.AddInt32(pc+8, nn))
	default:
		nn := int32(inst)
		nn = (nn << 8) >> 6
		return fmt.Sprintf("b%s 0x%08x", cond, util.AddInt32(pc+8, nn))
	}
}

func disassembleArmStack(inst uint32) string {
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

func disassembleArmLDRSTR(inst uint32) string {
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

func disassembleArmLDRSTR2(inst uint32) string {
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

func disassembleArmMRS(inst uint32) string {
	// useSpsr := (inst>>22)&0b1 > 0
	rd := (inst >> 12) & 0b1111
	return fmt.Sprintf("mrs %d,psr", rd)
}

func disassembleArmMSR(inst uint32) string {
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

	if util.Bit(inst, 25) { // register Psr[field] = Imm
		is, imm := ((inst>>8)&0b1111)*2, inst&0b1111_1111
		return fmt.Sprintf("msr %s,#%xror#%d", psr, imm, is)
	} else { // immediate Psr[field] = Rm
		rm := inst & 0b1111
		return fmt.Sprintf("msr %s,r%d", psr, rm)
	}
}

func disassembleArmSWP(inst uint32) string {
	rn, rd, rm := (inst>>16)&0b1111, (inst>>12)&0b1111, inst&0b1111

	suffix := ""
	byteUnit := util.Bit(inst, 22)
	if byteUnit {
		suffix = "b"
	}

	return fmt.Sprintf("swp%s r%d,r%d,[r%d]", suffix, rd, rm, rn)
}

func disassembleArmMPY(inst uint32) string {
	opcode := inst >> 21 & 0b1111
	switch opcode {
	case 0b0000:
		return fmt.Sprintf("mul r%d,r%d,r%d", inst>>16&0b1111, inst&0b1111, inst>>8&0b1111)
	case 0b0001:
		rd, rn, rs, rm := inst>>16&0b1111, inst>>12&0b1111, inst>>8&0b1111, inst&0b1111
		return fmt.Sprintf("mla r%d,r%d,r%d,r%d", rd, rm, rs, rn)
	case 0b0010:
		return "UMAAL is unsupported"
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

var aluOp2str = map[uint32]string{0x0: "and", 0x1: "eor", 0x2: "sub", 0x3: "rsb", 0x4: "add", 0x5: "adc", 0x6: "sbc", 0x7: "rsc", 0x8: "tst", 0x9: "teq", 0xa: "cmp", 0xb: "cmn", 0xc: "orr", 0xd: "mov", 0xe: "bic", 0xf: "mvn"}

func disassembleArmALU(inst uint32) string {
	cond := gba.Cond(inst >> 28)
	opcode := aluOp2str[inst>>21&0b1111]
	rd := inst >> 12 & 0b1111
	rn := (inst >> 16) & 0b1111
	op2 := ""
	if !util.Bit(inst, 25) {
		// register
		is := (inst >> 7) & 0b11111
		rm := inst & 0b1111

		shiftType := (inst >> 5) & 0b11
		shift := [4]string{"lsl", "lsr", "asr", "ror"}[shiftType]

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
		return fmt.Sprintf("%s%s r%d,r%d,%s", opcode, cond, rd, rn, op2)
	case 8, 9, 0xa, 0xb:
		return fmt.Sprintf("%s%s r%d,%s", opcode, cond, rn, op2)
	default:
		return fmt.Sprintf("%s%s r%d,%s", opcode, cond, rd, op2)
	}
}
