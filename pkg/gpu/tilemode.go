package gpu

import (
	"image"
	"mettaur/pkg/util"
)

func (g *GPU) drawTextBG(screen *image.RGBA, idx int) {
	bgCnt := util.LE16(g.IO[BG0CNT+2*idx:])
	width, height := 256, 256
	x := [2]uint32{1, 1}
	switch (bgCnt >> 14) & 0b11 {
	case 1:
		x[0] = 2
		width, height = 512, 256
	case 2:
		x[1] = 2
		width, height = 256, 512
	case 3:
		x[0], x[1] = 2, 2
		width, height = 512, 512
	}
	vScreen := image.NewRGBA(image.Rect(0, 0, width, height))

	tileBlock, colorMode, mapBlock := ((uint32(bgCnt)>>2)&0b11)*0x4000, (bgCnt>>7)&0b1, ((uint32(bgCnt)>>8)&0b11111)*0x0800

	xTiles, yTiles := 32*x[0], 32*x[1]
	for yTile := uint32(0); yTile < yTiles; yTile++ {
		for xTile := uint32(0); xTile < xTiles; xTile++ {
			mapIdx := yTile*yTiles + xTile
			mapData := util.LE16(g.VRAM[(mapBlock + 2*(mapIdx)):])
			tileIdx, paletteIdx := uint32(mapData&0b0011_1111_1111), int((mapData>>12)&0b1111)
			if colorMode == color16 {
				tileData := g.VRAM[tileBlock+32*tileIdx:]
				for y := uint32(0); y < 8; y++ {
					for x := uint32(0); x < 8; x += 2 {
						c := tileData[y*4+x/2]
						set(vScreen, int(xTile*8+x), int(yTile*8+y), g.paletteColor(paletteIdx, int(c&0b1111)))
						set(vScreen, int(xTile*8+x+1), int(yTile*8+y), g.paletteColor(paletteIdx, int((c>>4)&0b1111)))
					}
				}
			} else {
				tileData := g.VRAM[tileBlock+64*tileIdx:]
				for y := uint32(0); y < 8; y++ {
					for x := uint32(0); x < 8; x++ {
						set(vScreen, int(xTile*8+x), int(yTile*8+y), g.paletteColor(-1, int(tileData[y*8+x])))
					}
				}
			}
		}
	}

	scrollX, scrollY := int(util.LE16(g.IO[BG0HOFS+idx*4:])), int(util.LE16(g.IO[BG0VOFS+idx*4:]))
	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			set(screen, x, y, vScreen.At(x+scrollX, y+scrollY))
		}
	}
}

func (g *GPU) drawAffineBG(screen *image.RGBA, idx int) {
	bgCnt := util.LE16(g.IO[BG0CNT+2*idx:])
	width, height := 128, 128
	mag := [2]uint32{1, 1}
	switch (bgCnt >> 14) & 0b11 {
	case 1:
		mag[0], mag[1] = 2, 2
		width, height = 256, 256
	case 2:
		mag[0], mag[1] = 4, 4
		width, height = 512, 512
	case 3:
		mag[0], mag[1] = 8, 8
		width, height = 1024, 1024
	}
	vScreen := image.NewRGBA(image.Rect(0, 0, width, height))

	tileBlock, mapBlock := ((uint32(bgCnt)>>2)&0b11)*0x4000, ((uint32(bgCnt)>>8)&0b11111)*0x0800

	delta := (idx - 2) * 0x10
	pa, pb, pc, pd := int16(util.LE16(g.IO[BG2PA+delta:])), int16(util.LE16(g.IO[BG2PA+delta+2:])), int16(util.LE16(g.IO[BG2PA+delta+4:])), int16(util.LE16(g.IO[BG2PA+delta+6:]))
	xTiles, yTiles := 16*mag[0], 16*mag[1]
	for yTile := uint32(0); yTile < yTiles; yTile++ {
		for xTile := uint32(0); xTile < xTiles; xTile++ {
			mapIdx := yTile*yTiles + xTile
			tileIdx := uint32(g.VRAM[mapBlock+mapIdx])
			tileData := g.VRAM[tileBlock+64*tileIdx : tileBlock+64*tileIdx+64]
			for y := uint32(0); y < 8; y++ {
				for x := uint32(0); x < 8; x++ {
					px, py := int(xTile*8+x), int(yTile*8+y)
					px = (int(pa)*px + int(pb)*py) / 256
					py = (int(pc)*px + int(pd)*py) / 256
					set(vScreen, px, py, g.paletteColor(-1, int(tileData[y*8+x])))
				}
			}
		}
	}

	scrollX, scrollY := 0, 0 // TODO
	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			set(screen, x, y, vScreen.At(x+scrollX, y+scrollY))
		}
	}
}
