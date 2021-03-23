package gpu

import (
	"fmt"
	"mettaur/pkg/util"
)

func (g *GPU) PrintBGMap0() {
	bgCnt := util.LE16(g.IO[BG0CNT:])
	mapBlockOfs := ((uint32(bgCnt) >> 8) & 0b11111) * 0x0800
	mapBlock := g.VRAM[mapBlockOfs : mapBlockOfs+2*uint32(kb)]
	fmt.Println(mapBlock)
}
