package gba

import (
	"fmt"
	"mettaur/pkg/ram"
	"mettaur/pkg/util"
)

func (g *GBA) in(start, end uint32) bool {
	return g.PC >= start && g.PC <= end
}

func (g *GBA) thumbInst(inst uint16) {
	if inst != 0 {
		fmt.Printf("Thumb pc, inst: 0x%04x, 0x%02x\n", g.PC, inst)
	}
}
func (g *GBA) armInst(inst uint32) {
	if inst != 0 {
		fmt.Printf("ARM pc, inst: 0x%04x, 0x%04x\n", g.PC, inst)
	}
}
func (g *GBA) printExceptions() {
	flag := uint16(g.getRAM(ram.IE)) & uint16(g.getRAM(ram.IF))
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
