package gba

import "fmt"

const (
	flagN = 31
	flagZ = 30
	flagC = 29
	flagV = 28
	flagQ = 27
	flagI = 7
	flagF = 6
	flagT = 5
)

// Mode represents OS mode
type Mode byte

// OS mode
const (
	USR Mode = 0b10000
	FIQ      = 0b10001
	IRQ      = 0b10010
	SWI      = 0b10011
	ABT      = 0b10111
	UND      = 0b11011
	SYS      = 0b11111
)

// Reg represents register
type Reg struct {
	R                                    [16]uint32
	R8Fiq, R9Fiq, R10Fiq, R11Fiq, R12Fiq uint32
	R8Usr, R9Usr, R10Usr, R11Usr, R12Usr uint32
	R13Bank                              [6]uint32 // fiq, svc, abt, irq, und, usr
	R14Bank                              [6]uint32 // fiq, svc, abt, irq, und, usr
	CPSR                                 uint32
	SPSRBank                             [5]uint32 // fiq, svc, abt, irq, und
}

func NewReg() *Reg {
	reg := &Reg{}
	return reg
}

func (r *Reg) softReset() {
	r.R[15] = 0x0000_00ac
	cpsr := uint32(0)
	cpsr |= SWI
	cpsr |= 1 << 6
	cpsr |= 1 << 7
	r.CPSR = cpsr
}

func bankIdx(mode Mode) int {
	switch mode {
	case FIQ:
		return 0
	case IRQ:
		return 3
	case SWI:
		return 1
	case ABT:
		return 2
	case UND:
		return 4
	case USR, SYS:
		return 5
	}
	return -1
}

// SetCPSRFlag sets CPSR flag
func (r *Reg) SetCPSRFlag(idx int, flag bool) {
	if idx < 0 || idx > 31 {
		return
	}
	if flag {
		r.CPSR = r.CPSR | (1 << idx)
	} else {
		r.CPSR = r.CPSR & ^(1 << idx)
	}
}

// GetCPSRFlag get CPSR flag
func (r *Reg) GetCPSRFlag(idx int) bool {
	if idx < 0 || idx > 31 {
		return false
	}
	return ((r.CPSR >> idx) & 1) == 1
}

// getOSMode get Processor mode
func (r *Reg) getOSMode() Mode {
	return Mode(r.CPSR & 0b11111)
}
func (r *Reg) isSysMode() bool {
	return Mode(r.CPSR&0b11111) == SYS
}

// SetOSMode set Processor mode
// ref: arm_mode_set
func (r *Reg) setOSMode(mode Mode) {
	curr := r.getOSMode()
	r.CPSR = (r.CPSR & 0b1111_1111_1111_1111_1111_1111_1110_0000) | uint32(mode)
	r.copyRegToBank(curr)
	r.copyBankToReg(mode)
}

// ref: arm_spsr_to_cpsr
func (r *Reg) restoreOSMode() {
	currMode := r.getOSMode()
	r.CPSR = r.SPSRBank[bankIdx(currMode)]
	prevMode := r.getOSMode()
	r.copyRegToBank(currMode)
	r.copyBankToReg(prevMode)
}

// save CPSR into SPSR
// ref: arm_regs_to_bank
func (r *Reg) copyRegToBank(mode Mode) {
	if mode != FIQ {
		r.R8Usr = r.R[8]
		r.R9Usr = r.R[9]
		r.R10Usr = r.R[10]
		r.R11Usr = r.R[11]
		r.R12Usr = r.R[12]
	}

	switch mode {
	case USR, SYS:
		r.R13Bank[5] = r.R[13]
		r.R14Bank[5] = r.R[14]
	case FIQ:
		r.R8Fiq = r.R[8]
		r.R9Fiq = r.R[9]
		r.R10Fiq = r.R[10]
		r.R11Fiq = r.R[11]
		r.R12Fiq = r.R[12]
		r.R13Bank[0] = r.R[13]
		r.R14Bank[0] = r.R[14]
	case IRQ:
		r.R13Bank[3] = r.R[13]
		r.R14Bank[3] = r.R[14]
	case SWI:
		r.R13Bank[1] = r.R[13]
		r.R14Bank[1] = r.R[14]
	case ABT:
		r.R13Bank[2] = r.R[13]
		r.R14Bank[2] = r.R[14]
	case UND:
		r.R13Bank[4] = r.R[13]
		r.R14Bank[4] = r.R[14]
	}
}

// ref: arm_spsr_set
func (r *Reg) setSPSR(value uint32) {
	mode := r.getOSMode()
	switch mode {
	case FIQ:
		r.SPSRBank[0] = value
	case IRQ:
		r.SPSRBank[3] = value
	case SWI:
		r.SPSRBank[1] = value
	case ABT:
		r.SPSRBank[2] = value
	case UND:
		r.SPSRBank[4] = value
	}
}

// ref: arm_bank_to_regs
func (r *Reg) copyBankToReg(mode Mode) {
	if mode != FIQ {
		r.R[8] = r.R8Usr
		r.R[9] = r.R9Usr
		r.R[10] = r.R10Usr
		r.R[11] = r.R11Usr
		r.R[12] = r.R12Usr
	}

	switch mode {
	case USR, SYS:
		r.R[13] = r.R13Bank[5]
		r.R[14] = r.R14Bank[5]
	case FIQ:
		r.R[8] = r.R8Fiq
		r.R[9] = r.R9Fiq
		r.R[10] = r.R10Fiq
		r.R[11] = r.R11Fiq
		r.R[12] = r.R12Fiq
		r.R[13] = r.R13Bank[0]
		r.R[14] = r.R14Bank[0]
	case IRQ:
		r.R[13] = r.R13Bank[3]
		r.R[14] = r.R14Bank[3]
	case SWI:
		r.R[13] = r.R13Bank[1]
		r.R[14] = r.R14Bank[1]
	case ABT:
		r.R[13] = r.R13Bank[2]
		r.R[14] = r.R14Bank[2]
	case UND:
		r.R[13] = r.R13Bank[4]
		r.R[14] = r.R14Bank[4]
	}
}

func (m Mode) String() string {
	switch m {
	case USR:
		return "USR"
	case FIQ:
		return "FIQ"
	case IRQ:
		return "IRQ"
	case SWI:
		return "SWI"
	case ABT:
		return "ABT"
	case UND:
		return "UND"
	case SYS:
		return "SYS"
	}
	return fmt.Sprintf("Unknown(%d)", m)
}
