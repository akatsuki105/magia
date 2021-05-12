package gba

import (
	"fmt"
	"magia/pkg/ram"
	"magia/pkg/util"
	"os"
	"runtime"
)

const (
	ROM = 0x0800_0000
)

var debug = false

type History struct {
	inst Inst
	reg  Reg
}

type IRQHistory struct {
	irq      IRQID
	start    uint32
	returnTo uint32
	reg      Reg
}

const (
	historySize    = 10
	irqHistorySize = 10
)

// 0: oldest -> 9: newest
var histories [historySize]History = [historySize]History{}
var irqHistories [irqHistorySize]IRQHistory = [irqHistorySize]IRQHistory{}

var breakPoint []uint32 = []uint32{
	// 0x080006A8,
}

func (g *GBA) breakpoint() {
	fmt.Printf("Breakpoint: 0x%04x\n", g.inst.loc)
	printRegister(g.Reg)
	printPSR(g.Reg)
	g.printLCD()

	counter++
	// if counter == 1 {
	// 	g.Exit("")
	// }
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
	case util.Bit(flag, int(irqVBlank)):
		fmt.Println("exception occurred: IRQ VBlank")
	case util.Bit(flag, int(irqHBlank)):
		fmt.Println("exception occurred: IRQ HBlank")
	case util.Bit(flag, int(irqVCount)):
		fmt.Println("exception occurred: IRQ VCount")
	case util.Bit(flag, int(irqTimer0)):
		fmt.Println("exception occurred: IRQ Timer0")
	case util.Bit(flag, int(irqTimer1)):
		fmt.Println("exception occurred: IRQ Timer1")
	case util.Bit(flag, int(irqTimer2)):
		fmt.Println("exception occurred: IRQ Timer2")
	case util.Bit(flag, int(irqTimer3)):
		fmt.Println("exception occurred: IRQ Timer3")
	case util.Bit(flag, int(irqSerial)):
		fmt.Println("exception occurred: IRQ Serial")
	case util.Bit(flag, int(irqDMA0)):
		fmt.Println("exception occurred: IRQ DMA0")
	case util.Bit(flag, int(irqDMA1)):
		fmt.Println("exception occurred: IRQ DMA1")
	case util.Bit(flag, int(irqDMA2)):
		fmt.Println("exception occurred: IRQ DMA2")
	case util.Bit(flag, int(irqDMA3)):
		fmt.Println("exception occurred: IRQ DMA3")
	case util.Bit(flag, int(irqKEY)):
		fmt.Println("exception occurred: IRQ KEY")
	case util.Bit(flag, int(irqGamePak)):
		fmt.Println("exception occurred: IRQ GamePak")
	}
}

func outputCPSRFlag(r Reg) string {
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
	fmt.Printf(str, r.CPSR, outputCPSRFlag(r), r.SPSRBank[0], r.SPSRBank[1], r.SPSRBank[2], r.SPSRBank[3], r.SPSRBank[4])
}

func printR13Bank(r Reg) {
	str := ` R13_fiq: 0x%08x R13_svc: 0x%08x R13_abt: 0x%08x R13_irq: 0x%08x R13_und: 0x%08x R13_usr: 0x%08x
`
	fmt.Printf(str, r.R13Bank[0], r.R13Bank[1], r.R13Bank[2], r.R13Bank[3], r.R13Bank[4], r.R13Bank[5])
}

func printR14Bank(r Reg) {
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

func (g *GBA) printSWI(nn byte) {
	state := "ARM"
	if g.GetCPSRFlag(flagT) {
		state = "THUMB"
	}

	switch nn {
	case 0x05:
		// fmt.Printf("%s.VBlankIntrWait() in 0x%08x\n", state, g.inst.loc)
	case 0x06:
		fmt.Printf("%s.Div(0x%x, 0x%x, 0x%x) in 0x%08x\n", state, g.R[0], g.R[1], g.R[3], g.inst.loc)
	case 0x07:
		fmt.Printf("%s.DivArm(0x%x, 0x%x, 0x%x) in 0x%08x\n", state, g.R[0], g.R[1], g.R[3], g.inst.loc)
	case 0x08:
		fmt.Printf("%s.Sqrt(0x%x) in 0x%08x\n", state, g.R[0], g.inst.loc)
	case 0x0b:
		// fmt.Printf("%s.CPUSet(0x%x, 0x%x, 0x%x) in 0x%08x\n", state, g.R[0], g.R[1], g.R[2], g.inst.loc)
		fmt.Printf("%s.%s\n", state, g.outputCPUSet())
	case 0x0c:
		fmt.Printf("%s.CPUFastSet(0x%x, 0x%x, 0x%x) in 0x%08x\n", state, g.R[0], g.R[1], g.R[2], g.inst.loc)
	case 0x0e:
		fmt.Printf("%s.BgAffineSet(0x%x, 0x%x, 0x%x) in 0x%08x\n", state, g.R[0], g.R[1], g.R[2], g.inst.loc)
	case 0x0f:
		fmt.Printf("%s.ObjAffineSet(0x%x, 0x%x, 0x%x, 0x%x) in 0x%08x\n", state, g.R[0], g.R[1], g.R[2], g.R[3], g.inst.loc)
	default:
		fmt.Printf("%s.SWI(0x%x) in 0x%08x\n", state, nn, g.inst.loc)
	}
}

func (g *GBA) outputCPUSet() string {
	size := g.R[2] & 0b1_1111_1111_1111_1111_1111
	if util.Bit(g.R[2], 26) {
		size *= 4
	} else {
		size *= 2
	}

	fill := util.Bit(g.R[2], 24)
	if fill {
		return fmt.Sprintf("Memfill 0x%08x(0x%x) -> 0x%08x-%08x", g.R[0], g._getRAM(g.R[0]), g.R[1], g.R[1]+size)
	} else {
		return fmt.Sprintf("Memcpy 0x%08x-%08x -> 0x%08x-%08x", g.R[0], g.R[0]+size, g.R[1], g.R[1]+size)
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

func (g *GBA) printRAM32(addr uint32) {
	value := g._getRAM(addr)
	fmt.Printf("Word[0x%08x] => 0x%08x\n", addr, value)
}
func (g *GBA) printRAM8(addr uint32) {
	value := g._getRAM(addr)
	fmt.Printf("Word[0x%08x] => 0x%02x\n", addr, byte(value))
}

func (g *GBA) pushHistory() {
	if g.inst.inst == 0 {
		return
	}
	if g.halt {
		return
	}
	if g.inst.loc == histories[historySize-1].inst.loc {
		return
	}

	for i := 0; i < historySize-1; i++ {
		histories[i] = histories[i+1]
	}

	histories[historySize-1] = History{
		inst: g.inst,
		reg:  g.Reg,
	}
}
func (g *GBA) pushIRQHistory(i IRQHistory) {
	for i := 0; i < irqHistorySize-1; i++ {
		irqHistories[i] = irqHistories[i+1]
	}
	irqHistories[irqHistorySize-1] = i
}

// PrintHistory print out histories
func (g *GBA) PrintHistory() {
	for i, h := range histories {
		if h.reg.GetCPSRFlag(flagT) {
			fmt.Printf("%d: 0x%08x in 0x%08x\n", i, h.inst.inst, h.inst.loc)
		} else {
			fmt.Printf("%d: %s(0x%08x) in 0x%08x\n", i, armDecode(h.inst.loc, h.inst.inst), h.inst.inst, h.inst.loc)
		}
		// printRegister(h.reg)
	}
}

var irq2str = map[IRQID]string{irqVBlank: "Vblank", irqHBlank: "Hblank", irqVCount: "VCount", irqTimer0: "Timer0", irqTimer1: "Timer1", irqTimer2: "Timer2", irqTimer3: "Timer3", irqSerial: "Serial", irqDMA0: "DMA0", irqDMA1: "DMA1", irqDMA2: "DMA2", irqDMA3: "DMA3", irqKEY: "KEY", irqGamePak: "GamePak"}

func (i IRQID) String() string { return irq2str[i] }

func (ih IRQHistory) String() string {
	mode := "ARM"
	if ih.reg.GetCPSRFlag(flagT) {
		mode = "THUMB"
	}
	return fmt.Sprintf("IRQ(%s): 0x%08x -> 0x%08x on %s", ih.irq, ih.start, ih.returnTo, mode)
}

func (g *GBA) PanicHandler(place string, stack bool) {
	if err := recover(); err != nil {
		fmt.Fprintf(os.Stderr, "%s emulation error: %s in 0x%08x\n", place, err, g.PC())
		for depth := 0; ; depth++ {
			_, file, line, ok := runtime.Caller(depth)
			if !ok {
				break
			}
			fmt.Printf("======> %d: %v:%d\n", depth, file, line)
		}
		g.Exit("")
	}
}
