package gpu

import (
	"fmt"
	"mettaur/pkg/util"
)

func (g *GPU) PrintBGMap0() {
	bgCnt := util.LE16(g.IO[BG0CNT:])
	mapBlockOfs := ((uint32(bgCnt) >> 8) & 0b11111) * 0x0800
	_mapBlock := g.VRAM[mapBlockOfs : mapBlockOfs+2*uint32(kb)]

	mapBlock := [kb]uint16{}
	for i := uint(0); i < 2*kb; i += 2 {
		mapBlock[i/2] = util.LE16(_mapBlock[i:])
	}
	fmt.Printf("Map Addr: 0x%08x\n", 0x0600_0000+mapBlockOfs)

	fmt.Println("[")
	for i, data := range mapBlock {
		fmt.Printf("%02x ", byte(data))
		if i%32 == 31 && i > 0 {
			fmt.Println()
		}
	}
	fmt.Println("]")
}
