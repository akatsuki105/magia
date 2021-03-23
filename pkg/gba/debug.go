package gba

import (
	"fmt"
	"mettaur/pkg/ram"
	"mettaur/pkg/util"
)

const (
	ROM = 0x0800_0000
)

type Debug struct {
}

var breakPoint []uint32 = []uint32{
	// ROM + 0x77f4,
}

var debugCounter = 0
var debugCounterOn = false

func (g *GBA) in(start, end uint32) bool {
	return g.inst.loc >= start && g.inst.loc <= end
}

func (g *GBA) thumbInst(inst uint16) {
	if inst != 0 {
		fmt.Printf("Thumb pc, inst, cycle: 0x%04x, 0x%02x, %d:%d\n", g.inst.loc, inst, g.line, g.cycle)
	}
}
func (g *GBA) armInst(inst uint32) {
	if inst != 0 {
		fmt.Printf("ARM pc, inst, cycle: 0x%04x, 0x%04x, %d:%d\n", g.inst.loc, inst, g.line, g.cycle)
	}
}
func (g *GBA) printInst(inst uint32) {
	if inst != 0 {
		t := g.GetCPSRFlag(flagT)
		if t {
			fmt.Printf("Thumb pc, inst, cycle: 0x%04x, 0x%02x, %d:%d\n", g.inst.loc, inst, g.line, g.cycle)
			return
		}
		fmt.Printf("ARM pc, inst, cycle: 0x%04x, 0x%04x, %d:%d\n", g.inst.loc, inst, g.line, g.cycle)
	}
}

func (g *GBA) printIRQExceptions() {
	flag := uint16(g._getRAM(ram.IE)) & uint16(g._getRAM(ram.IF))
	switch {
	case util.Bit(flag, irqVBlank):
		fmt.Println("exception occurred: IRQ VBlank")
	case util.Bit(flag, irqHBlank):
		fmt.Println("exception occurred: IRQ HBlank")
	case util.Bit(flag, irqVCount):
		fmt.Println("exception occurred: IRQ VCount")
	case util.Bit(flag, irqTimer0):
		fmt.Println("exception occurred: IRQ Timer0")
	case util.Bit(flag, irqTimer1):
		fmt.Println("exception occurred: IRQ Timer1")
	case util.Bit(flag, irqTimer2):
		fmt.Println("exception occurred: IRQ Timer2")
	case util.Bit(flag, irqTimer3):
		fmt.Println("exception occurred: IRQ Timer3")
	case util.Bit(flag, irqSerial):
		fmt.Println("exception occurred: IRQ Serial")
	case util.Bit(flag, irqDMA0):
		fmt.Println("exception occurred: IRQ DMA0")
	case util.Bit(flag, irqDMA1):
		fmt.Println("exception occurred: IRQ DMA1")
	case util.Bit(flag, irqDMA2):
		fmt.Println("exception occurred: IRQ DMA2")
	case util.Bit(flag, irqDMA3):
		fmt.Println("exception occurred: IRQ DMA3")
	case util.Bit(flag, irqKEY):
		fmt.Println("exception occurred: IRQ KEY")
	case util.Bit(flag, irqGamePak):
		fmt.Println("exception occurred: IRQ GamePak")
	}
}

func (g *GBA) printBGMap0() {
	g.GPU.PrintBGMap0()
}

func (g *GBA) exitDebug() {
}

func (g *GBA) printCPSRFlag() string {
	n, z, c, v, i, f, t := g.GetCPSRFlag(flagN), g.GetCPSRFlag(flagZ), g.GetCPSRFlag(flagC), g.GetCPSRFlag(flagV), g.GetCPSRFlag(flagI), g.GetCPSRFlag(flagF), g.GetCPSRFlag(flagT)
	result := "["
	if n {
		result += "N"
	} else {
		result += "-"
	}
	if z {
		result += "Z"
	} else {
		result += "-"
	}
	if c {
		result += "C"
	} else {
		result += "-"
	}
	if v {
		result += "V"
	} else {
		result += "-"
	}
	if i {
		result += "I"
	} else {
		result += "-"
	}
	if f {
		result += "F"
	} else {
		result += "-"
	}
	if t {
		result += "T"
	} else {
		result += "-"
	}
	return result + "]"
}

func (g *GBA) printPSR() {
	str := ` CPSR: 0x%08x %s SPSR_fiq: 0x%08x SPSR_svc: 0x%08x SPSR_abt: 0x%08x SPSR_irq: 0x%08x SPSR_und: 0x%08x
`
	fmt.Printf(str, g.CPSR, g.printCPSRFlag(), g.SPSRBank[0], g.SPSRBank[1], g.SPSRBank[2], g.SPSRBank[3], g.SPSRBank[4])
}

func (g *GBA) printR14Bank() {
	str := ` R14_fiq: 0x%08x R14_svc: 0x%08x R14_abt: 0x%08x R14_irq: 0x%08x R14_und: 0x%08x R14_usr: 0x%08x
`
	fmt.Printf(str, g.R14Bank[0], g.R14Bank[1], g.R14Bank[2], g.R14Bank[3], g.R14Bank[4], g.R14Bank[5])
}

func (g *GBA) printRegister() {
	str := ` r0: %08x   r1: %08x   r2: %08x   r3: %08x
 r4: %08x   r5: %08x   r6: %08x   r7: %08x
 r8: %08x   r9: %08x  r10: %08x  r11: %08x
 r12: %08x  r13: %08x  r14: %08x  r15: %08x
`
	fmt.Printf(str, g.R[0], g.R[1], g.R[2], g.R[3], g.R[4], g.R[5], g.R[6], g.R[7], g.R[8], g.R[9], g.R[10], g.R[11], g.R[12], g.R[13], g.R[14], g.R[15])
}

func (g *GBA) printLCD() {
	str := ` dispcnt: %04x dispstat: %04x LY: %d
`
	fmt.Printf(str, uint16(g._getRAM(ram.DISPCNT)), uint16(g._getRAM(ram.DISPSTAT)), byte(g._getRAM(ram.VCOUNT)))
}

func (g *GBA) printSWI(nn byte) {
	state := "ARM"
	if g.GetCPSRFlag(flagT) {
		state = "THUMB"
	}

	switch nn {
	case 0x05:
		fmt.Printf("%s.VBlankIntrWait() in %04x\n", state, g.inst.loc)
	case 0x0c:
		fmt.Printf("%s.CPUFastSet(0x%x, 0x%x, 0x%x) in %04x\n", state, g.R[0], g.R[1], g.R[2], g.inst.loc)
	default:
		fmt.Printf("%s.SWI(%x) in %04x\n", state, nn, g.inst.loc)
	}
}

func (g *GBA) printPC() {
	fmt.Printf(" PC: %04x\n", g.pipe.inst[0].loc)
}

func (g *GBA) printIRQRegister() {
	str := ` IME: %d IE: %02x IF: %02x
`
	fmt.Printf(str, uint16(g._getRAM(ram.IME)), uint16(g._getRAM(ram.IE)), byte(g._getRAM(ram.IF)))
}

func (g *GBA) printRAM(addr uint32) {
	value := g._getRAM(addr)
	fmt.Printf("Word[0x%08x] => 0x%08x\n", addr, value)
}
