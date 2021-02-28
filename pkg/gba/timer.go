package gba

import "mettaur/pkg/ram"

var (
	wsN  = [4]int{4, 3, 2, 8}
	wsS0 = [2]int{2, 1}
	wsS1 = [2]int{4, 1}
	wsS2 = [2]int{8, 1}
)

func (g *GBA) cycleN(addr uint32) int {
	if ok, _ := ram.GamePak0(addr); ok {
		idx := g.RAM.Get(addr) >> 2 & 0b11
		return wsN[idx]
	}
	if ok, _ := ram.GamePak1(addr); ok {
		idx := g.RAM.Get(addr) >> 5 & 0b11
		return wsN[idx]
	}
	if ok, _ := ram.GamePak2(addr); ok {
		idx := g.RAM.Get(addr) >> 8 & 0b11
		return wsN[idx]
	}
	if ok, _ := ram.SRAM(addr); ok {
		idx := g.RAM.Get(addr) & 0b11
		return wsN[idx]
	}
	return 1
}

func (g *GBA) cycleS(addr uint32) int {
	if ok, _ := ram.GamePak0(addr); ok {
		idx := g.RAM.Get(addr) >> 4 & 0b1
		return wsS0[idx]
	}
	if ok, _ := ram.GamePak1(addr); ok {
		idx := g.RAM.Get(addr) >> 7 & 0b1
		return wsS1[idx]
	}
	if ok, _ := ram.GamePak2(addr); ok {
		idx := g.RAM.Get(addr) >> 10 & 0b1
		return wsS2[idx]
	}
	if ok, _ := ram.SRAM(addr); ok {
		idx := g.RAM.Get(addr) & 0b11
		return wsN[idx]
	}
	return 1
}

func (g *GBA) timer(cycle int) {}
