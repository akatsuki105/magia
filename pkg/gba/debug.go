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

type History struct {
	inst Inst
	reg  Reg
}

// 0: oldest -> 9: newest
var histories [10]History = [10]History{}

var breakPoint []uint32 = []uint32{}

func (g *GBA) breakpoint() {
	fmt.Printf("Breakpoint: 0x%04x\n", g.inst.loc)
	printRegister(g.Reg)
	printPSR(g.Reg)
	g.printLCD()
	fmt.Println()

	counter++
	// if counter == 1 {
	// 	g.Exit("")
	// }
}

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

func (g *GBA) printBGMap(bg int) {
	g.GPU.PrintBGMap(bg)
}
func (g *GBA) printPalette() {
	g.GPU.PrintPalette()
}

func printCPSRFlag(r Reg) string {
	n, z, c, v, i, f, t := r.GetCPSRFlag(flagN), r.GetCPSRFlag(flagZ), r.GetCPSRFlag(flagC), r.GetCPSRFlag(flagV), r.GetCPSRFlag(flagI), r.GetCPSRFlag(flagF), r.GetCPSRFlag(flagT)
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

func printPSR(r Reg) {
	str := ` CPSR: 0x%08x %s SPSR_fiq: 0x%08x SPSR_svc: 0x%08x SPSR_abt: 0x%08x SPSR_irq: 0x%08x SPSR_und: 0x%08x
`
	fmt.Printf(str, r.CPSR, printCPSRFlag(r), r.SPSRBank[0], r.SPSRBank[1], r.SPSRBank[2], r.SPSRBank[3], r.SPSRBank[4])
}

func (g *GBA) printR13Bank(r Reg) {
	str := ` R13_fiq: 0x%08x R13_svc: 0x%08x R13_abt: 0x%08x R13_irq: 0x%08x R13_und: 0x%08x R13_usr: 0x%08x
`
	fmt.Printf(str, r.R13Bank[0], r.R13Bank[1], r.R13Bank[2], r.R13Bank[3], r.R13Bank[4], r.R13Bank[5])
}

func (g *GBA) printR14Bank(r Reg) {
	str := ` R14_fiq: 0x%08x R14_svc: 0x%08x R14_abt: 0x%08x R14_irq: 0x%08x R14_und: 0x%08x R14_usr: 0x%08x
`
	fmt.Printf(str, r.R14Bank[0], r.R14Bank[1], r.R14Bank[2], r.R14Bank[3], r.R14Bank[4], r.R14Bank[5])
}

func printRegister(r Reg) {
	str := ` r0: %08x   r1: %08x   r2: %08x   r3: %08x
 r4: %08x   r5: %08x   r6: %08x   r7: %08x
 r8: %08x   r9: %08x  r10: %08x  r11: %08x
 r12: %08x  r13: %08x  r14: %08x  r15: %08x
`
	fmt.Printf(str, r.R[0], r.R[1], r.R[2], r.R[3], r.R[4], r.R[5], r.R[6], r.R[7], r.R[8], r.R[9], r.R[10], r.R[11], r.R[12], r.R[13], r.R[14], r.R[15])
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
		// fmt.Printf("%s.VBlankIntrWait() in %04x\n", state, g.inst.loc)
	case 0x0b:
		fmt.Printf("%s.CPUSet(0x%x, 0x%x, 0x%x) in %04x\n", state, g.R[0], g.R[1], g.R[2], g.inst.loc)
	case 0x0c:
		fmt.Printf("%s.CPUFastSet(0x%x, 0x%x, 0x%x) in %04x\n", state, g.R[0], g.R[1], g.R[2], g.inst.loc)
	default:
		fmt.Printf("%s.SWI(%x) in %04x\n", state, nn, g.inst.loc)
	}
}

func (g *GBA) printPC() {
	fmt.Printf(" PC: %04x\n", g.pipe.inst[0].loc)
}
func (g *GBA) PC() uint32 {
	return g.inst.loc
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

func (g *GBA) pushHistory() {
	if g.inst.inst == 0 {
		return
	}
	if g.inst.loc == histories[9].inst.loc {
		return
	}

	for i := 9; i >= 1; i-- {
		histories[i-1] = histories[i]
	}

	histories[9] = History{
		inst: g.inst,
		reg:  g.Reg,
	}
}

// PrintHistory print out histories
func PrintHistory() {
	for i, h := range histories {
		fmt.Printf("%d: 0x%08x in 0x%08x\n", i, h.inst.inst, h.inst.loc)
	}
}
