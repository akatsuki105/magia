package gpu

import (
	"image"
	"image/color"
	"mettaur/pkg/util"
)

var objDim = [4][3][2]int{
	{{8, 8}, {16, 8}, {8, 16}},
	{{16, 16}, {32, 8}, {8, 32}},
	{{32, 32}, {32, 16}, {16, 32}},
	{{64, 64}, {64, 32}, {32, 64}},
}

func (g *GPU) drawObjs(screen *image.RGBA) {
	for i := 0; i < 128; i++ {
		g.drawObj(screen, i)
	}
}

func (g *GPU) drawObj(screen *image.RGBA, idx int) {
	atr0, atr1, atr2 := util.LE16(g.OAM[8*idx:]), util.LE16(g.OAM[8*idx+2:]), util.LE16(g.OAM[8*idx+4:])
	yCoord, xCoord := int(atr0&0xff), int(atr1&0x1ff)
	tileBaseIdx, colorMode, paletteIdx := int(atr2&0x3ff), (atr0 >> 13 & 0b1), int((atr2>>12)&0b1111)

	d1 := util.Bit(g.IO[DISPCNT], 6)
	shape, size := (atr0>>14)&0b11, (atr1>>14)&0b11
	xTiles, yTiles := objDim[size][shape][0]/8, objDim[size][shape][1]/8
	affine := util.Bit(atr0, 8)
	if affine {
		return
	} else {
		if util.Bit(atr0, 9) {
			return
		}

		// mapping delta
		delta := 1
		if colorMode == color256 {
			delta = 2
		}

		for yTile := 0; yTile < yTiles; yTile++ {
			for xTile := 0; xTile < xTiles; xTile++ {
				tileIdx := tileBaseIdx + xTile*delta + yTile*0x20
				if d1 {
					tileIdx = tileBaseIdx + (yTile*xTiles+xTile)*delta
				}

				if colorMode == color16 {
					tileData := g.VRAM[0x10000+32*tileIdx:]
					for y := 0; y < 8; y++ {
						for x := 0; x < 8; x += 2 {
							c := tileData[y*4+x/2]
							set(screen, int(xCoord+xTile*8+x), int(yCoord+yTile*8+y), g.paletteObjColor(paletteIdx, int(c&0b1111)))
							set(screen, int(xCoord+xTile*8+x+1), int(yCoord+yTile*8+y), g.paletteObjColor(paletteIdx, int((c>>4)&0b1111)))
						}
					}
				} else {
					tileData := g.VRAM[0x10000+32*tileIdx:]
					for y := 0; y < 8; y++ {
						for x := 0; x < 8; x++ {
							set(screen, int(xCoord+xTile*8+x), int(yCoord+yTile*8+y), g.paletteObjColor(-1, int(tileData[y*8+x])))
						}
					}
				}
			}
		}
	}
}

func (g *GPU) paletteObjColor(palette, index int) color.RGBA {
	var c uint16
	if index == 0 {
		return color.RGBA{0x00, 0x00, 0x00, 0x00}
	}
	if palette >= 0 && palette < 16 {
		c = util.LE16(g.Palette[0x200+palette*32+index*2:])
	} else {
		c = util.LE16(g.Palette[0x200+index*2:])
	}
	red, blue, green := byte((c&0b11111)*8), byte(((c>>5)&0b11111)*8), byte(((c>>10)&0b11111)*8)
	return color.RGBA{red, blue, green, 0xff}
}
