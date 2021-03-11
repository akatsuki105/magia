package gba

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
	R13Bank                              [5]uint32 // fiq, svc, abt, irq, und
	R14Bank                              [5]uint32 // fiq, svc, abt, irq, und
	CPSR                                 uint32
	SPSRBank                             [5]uint32 // fiq, svc, abt, irq, und
}

func NewReg() *Reg {
	r := [16]uint32{}
	r[15] = 0x08000000
	return &Reg{
		R: r,
	}
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

// GetOSMode get Processor mode
func (r *Reg) getOSMode() Mode {
	return Mode(r.CPSR & 0b11111)
}

// SetOSMode set Processor mode
func (r *Reg) setOSMode(mode Mode) {
	r.saveReg(mode)
	r.CPSR = (r.CPSR & 0b1111_1111_1111_1111_1111_1111_1110_0000) | uint32(mode)
}

func (r *Reg) saveReg(mode Mode) {
	switch mode {
	case FIQ:
		r.R8Fiq = r.R[8]
		r.R9Fiq = r.R[9]
		r.R10Fiq = r.R[10]
		r.R11Fiq = r.R[11]
		r.R12Fiq = r.R[12]
		r.R13Bank[0] = r.R[13]
		r.R14Bank[0] = r.R[14]
		r.SPSRBank[0] = r.CPSR
	case IRQ:
		r.R13Bank[3] = r.R[13]
		r.R14Bank[3] = r.R[14]
		r.SPSRBank[3] = r.CPSR
	case SWI:
		r.R13Bank[1] = r.R[13]
		r.R14Bank[1] = r.R[14]
		r.SPSRBank[1] = r.CPSR
	case ABT:
		r.R13Bank[2] = r.R[13]
		r.R14Bank[2] = r.R[14]
		r.SPSRBank[2] = r.CPSR
	case UND:
		r.R13Bank[4] = r.R[13]
		r.R14Bank[4] = r.R[14]
		r.SPSRBank[4] = r.CPSR
	}
}

func (r *Reg) restoreReg(mode Mode) {
	switch mode {
	case FIQ:
		r.R[8] = r.R8Fiq
		r.R[9] = r.R9Fiq
		r.R[10] = r.R10Fiq
		r.R[11] = r.R11Fiq
		r.R[12] = r.R12Fiq
		r.R[13] = r.R13Bank[0]
		r.R[14] = r.R14Bank[0]
		r.CPSR = r.SPSRBank[0]
	case IRQ:
		r.R[13] = r.R13Bank[3]
		r.R[14] = r.R14Bank[3]
		r.CPSR = r.SPSRBank[3]
	case SWI:
		r.R[13] = r.R13Bank[1]
		r.R[14] = r.R14Bank[1]
		r.CPSR = r.SPSRBank[1]
	case ABT:
		r.R[13] = r.R13Bank[2]
		r.R[14] = r.R14Bank[2]
		r.CPSR = r.SPSRBank[2]
	case UND:
		r.R[13] = r.R13Bank[4]
		r.R[14] = r.R14Bank[4]
		r.CPSR = r.SPSRBank[4]
	}
}

func (m Mode) String() string {
	switch m {
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
	return "USR"
}
