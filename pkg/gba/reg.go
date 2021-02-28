package gba

import "mettaur/pkg/util"

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
	R                                                 [16]uint32
	R8Fiq, R9Fiq, R10Fiq, R11Fiq, R12Fiq              uint32
	R13Fiq, R13Svc, R13Abt, R13Irq, R13Und            uint32
	R14Fiq, R14Svc, R14Abt, R14Irq, R14Und            uint32
	CPSR, SPSRFiq, SPSRSvc, SPSRAbt, SPSRIrq, SPSRUnd uint32
}

// SetCPSRFlag sets CPSR flag
func (r *Reg) SetCPSRFlag(idx int, flag bool) {
	if idx < 0 || idx > 31 {
		return
	}
	r.CPSR = r.CPSR | (uint32(util.BoolToInt(flag)) << idx)
}

// GetCPSRFlag get CPSR flag
func (r *Reg) GetCPSRFlag(idx int) bool {
	if idx < 0 || idx > 31 {
		return false
	}
	return util.ToBool((r.CPSR >> idx) & 1)
}
