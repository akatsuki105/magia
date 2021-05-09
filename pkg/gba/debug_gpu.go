package gba

import (
	"fmt"
	"magia/pkg/ram"
)

func (g *GBA) printBGMap(bg int) {
	g.GPU.PrintBGMap(bg)
}
func (g *GBA) printPalette() {
	g.GPU.PrintPalette()
}
func (g *GBA) printLCD() {
	str := ` dispcnt: 0x%04x dispstat: 0x%04x LY: %d
`
	fmt.Printf(str, uint16(g._getRAM(ram.DISPCNT)), uint16(g._getRAM(ram.DISPSTAT)), byte(g._getRAM(ram.VCOUNT)))
}
func (g *GBA) printBGCnt() {
	str := "BG0CNT: 0x%04x BG1CNT: 0x%04x BG2CNT: 0x%04x BG3CNT: 0x%04x\n"
	fmt.Printf(str, uint16(g._getRAM(ram.BG0CNT)), uint16(g._getRAM(ram.BG1CNT)), uint16(g._getRAM(ram.BG2CNT)), uint16(g._getRAM(ram.BG3CNT)))
}
