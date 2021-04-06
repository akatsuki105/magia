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
			tileIdx, paletteIdx, flipX, flipY := uint32(mapData&0b0011_1111_1111), int((mapData>>12)&0b1111), util.Bit(mapData, 10), util.Bit(mapData, 11)
			if colorMode == color16 {
				tileData := g.VRAM[tileBlock+32*tileIdx:]
				for y := uint32(0); y < 8; y++ {
					for x := uint32(0); x < 8; x += 2 {
						c := tileData[y*4+x/2]

						xCoord0, xCoord1, yCoord := int(xTile*8+x), int(xTile*8+x+1), int(yTile*8+y)
						if flipX {
							xCoord0 = int(xTile*8 + (7 - x))
							xCoord1 = int(xTile*8 + (7 - x - 1))
						}
						if flipY {
							yCoord = int(yTile*8 + (7 - y))
						}
						set(vScreen, xCoord0, yCoord, g.paletteColor(paletteIdx, int(c&0b1111)))
						set(vScreen, xCoord1, yCoord, g.paletteColor(paletteIdx, int((c>>4)&0b1111)))
					}
				}
			} else {
				tileData := g.VRAM[tileBlock+64*tileIdx:]
				for y := uint32(0); y < 8; y++ {
					for x := uint32(0); x < 8; x++ {
						xCoord, yCoord := int(xTile*8+x), int(yTile*8+y)
						if flipX {
							xCoord = int(xTile*8 + (7 - x))
						}
						if flipY {
							yCoord = int(yTile*8 + (7 - y))
						}
						set(vScreen, xCoord, yCoord, g.paletteColor(-1, int(tileData[y*8+x])))
					}
				}
			}
		}
	}

	mask := uint16(0b0000_0001_1111_1111)
	scrollX, scrollY := int(util.LE16(g.IO[BG0HOFS+idx*4:])&mask), int(util.LE16(g.IO[BG0VOFS+idx*4:])&mask)
	win0Enable, win1Enable, objWinEnable := util.Bit(g.IO[DISPCNT+1], 5), util.Bit(g.IO[DISPCNT+1], 6), util.Bit(g.IO[DISPCNT+1], 7)
	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			if win0Enable && g.inWindow0(x, y) {
				if g.IO[WININ]>>idx&0b1 == 0b1 {
					set(screen, x, y, vScreen.At((x+scrollX)%width, (y+scrollY)%height))
				}
			} else if win1Enable && g.inWindow1(x, y) {
				if g.IO[WININ+1]>>idx&0b1 == 0b1 {
					set(screen, x, y, vScreen.At((x+scrollX)%width, (y+scrollY)%height))
				}
			} else if objWinEnable && g.inObjWindow(x, y) {
				if g.IO[WINOUT+1]>>idx&0b1 == 0b1 {
					set(screen, x, y, vScreen.At((x+scrollX)%width, (y+scrollY)%height))
				}
			} else {
				set(screen, x, y, vScreen.At((x+scrollX)%width, (y+scrollY)%height))
			}
		}
	}
}

func (g *GPU) drawAffineBG(screen *image.RGBA, idx int) {
	bgCnt := util.LE16(g.IO[BG0CNT+2*idx:])

	tileBlock, mapBlock := int(((uint32(bgCnt)>>2)&0b11)*0x4000), int(((uint32(bgCnt)>>8)&0b11111)*0x0800)

	delta := (idx - 2) * 0x10
	pa, pb, pc, pd := int16(util.LE16(g.IO[BG2PA+delta:])), int16(util.LE16(g.IO[BG2PA+delta+2:])), int16(util.LE16(g.IO[BG2PA+delta+4:])), int16(util.LE16(g.IO[BG2PA+delta+6:]))
	refxi, refyi := util.LE32(g.IO[BG2X+delta:]), util.LE32(g.IO[BG2Y+delta:])
	ox, oy := int(int32(refxi<<4)>>4), int(int32((refyi<<4))>>4)

	win0Enable, win1Enable, objWinEnable := util.Bit(g.IO[DISPCNT+1], 5), util.Bit(g.IO[DISPCNT+1], 6), util.Bit(g.IO[DISPCNT+1], 7)
	tpr := int((byte(128/8) << (bgCnt >> 14))) // tiles per row(= tiles per column)
	tmsk := tpr - 1
	wrap := util.Bit(bgCnt, 13)
	for y := 0; y < 160; y++ {
		oldOx, oldOy := ox, oy
		for x := 0; x < 240; x++ {
			tileX, tileY := ox>>11, oy>>11
			if wrap {
				tileX &= tmsk
				tileY &= tmsk
			} else {
				if tileX < 0 || tileX >= tpr || tileY < 0 || tileY >= tpr {
					ox += int(pa)
					oy += int(pc)
					continue
				}
			}

			chrX, chrY := (ox>>8)&7, (oy>>8)&7
			mapAddr := mapBlock + tileY*tpr + tileX
			tileAddr := tileBlock + int(uint32(g.VRAM[mapAddr]))*64 + chrY*8 + chrX

			if win0Enable && g.inWindow0(x, y) {
				if g.IO[WININ]>>idx&0b1 == 0b1 {
					set(screen, x, y, g.paletteColor(-1, int(g.VRAM[tileAddr])))
				}
			} else if win1Enable && g.inWindow1(x, y) {
				if g.IO[WININ+1]>>idx&0b1 == 0b1 {
					set(screen, x, y, g.paletteColor(-1, int(g.VRAM[tileAddr])))
				}
			} else if objWinEnable && g.inObjWindow(x, y) {
				if g.IO[WINOUT+1]>>idx&0b1 == 0b1 {
					set(screen, x, y, g.paletteColor(-1, int(g.VRAM[tileAddr])))
				}
			} else {
				set(screen, x, y, g.paletteColor(-1, int(g.VRAM[tileAddr])))
			}

			ox += int(pa) // pa * x
			oy += int(pc) // pc * x
		}

		ox, oy = oldOx, oldOy
		ox += int(pb) // pb * y -> pa * x + pb * y + refx
		oy += int(pd) // pd * y -> pc * x + pd * y + refy
	}
}
