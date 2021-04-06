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

func (g *GPU) drawObjs(screen *image.RGBA, prio int) {
	objWin = [240][160]bool{} // init

	// prio: obj0 > obj1 > ... > obj127
	for i := 127; i >= 0; i-- {
		for j := 0; j < 160; j++ {
			g.drawObj(screen, i, prio, j)
		}
	}
}

func (g *GPU) drawObj(screen *image.RGBA, idx, prio, vcount int) {
	atr0, atr1, atr2 := util.LE16(g.OAM[8*idx:]), util.LE16(g.OAM[8*idx+2:]), util.LE16(g.OAM[8*idx+4:])
	if int((atr2>>10)&0b11) != prio {
		return
	}

	objY := int(atr0 & 0xff)

	d1 := util.Bit(g.IO[DISPCNT], 6)
	shape, size := (atr0>>14)&0b11, (atr1>>14)&0b11
	xTiles, yTiles := objDim[size][shape][0]/8, objDim[size][shape][1]/8

	affine := util.Bit(atr0, 8)
	if !affine && util.Bit(atr0, 9) {
		return
	}

	pa, pb, pc, pd := 0x100, 0x0, 0x0, 0x100
	if affine {
		pidx := (atr1 >> 9) & 0b11111
		pa, pb, pc, pd = int(int16(util.LE16(g.OAM[0x06+0x20*pidx:]))), int(int16(util.LE16(g.OAM[0x0e+0x20*pidx:]))), int(int16(util.LE16(g.OAM[0x16+0x20*pidx:]))), int(int16(util.LE16(g.OAM[0x1e+0x20*pidx:])))
	}

	rcx, rcy := xTiles*4, yTiles*4 // rotate center for affine
	double := util.Bit(atr0, 9)
	if affine && double {
		rcx *= 2
		rcy *= 2
	}

	// wrap
	if objY+rcy*2 > 0xff {
		objY -= 0x100
	}

	if objY > vcount || objY+rcy*2 <= vcount {
		return
	}

	objX := int(atr1 & 0x1ff)
	tileBaseIdx, colorMode, paletteIdx := int(atr2&0x3ff), (atr0 >> 13 & 0b1), int((atr2>>12)&0b1111)

	y := vcount - objY
	flipY := util.Bit(atr1, 13)
	if !affine && flipY {
		y = int(int32(y) ^ ((int32(yTiles) * 8) - 1))
	}

	ox, oy := int32(pa*(-rcx)+pb*(y-rcy)+(xTiles*256*4)), int32(pc*(-rcx)+pd*(y-rcy)+(yTiles*256*4))

	flipX := util.Bit(atr1, 12)
	if !affine && flipX {
		ox = (int32(xTiles)*8 - 1) << 8
		pa = -0x100
	}

	delta := 1
	if colorMode == color256 {
		delta = 2
	}

	isObjWin := atr0>>10&0b11 == 2
	for x := 0; x < rcx*2; x++ {
		if objX+x < 0 {
			ox += int32(pa)
			oy += int32(pc)
			continue
		}
		if objX+x >= 240 {
			break
		}

		xTile, yTile := ox>>11, oy>>11
		if (ox < 0 || xTile >= int32(xTiles)) || (oy < 0 || yTile >= int32(yTiles)) {
			ox += int32(pa)
			oy += int32(pc)
			continue
		}

		chrX, chrY := (ox>>8)&7, (oy>>8)&7
		tileIdx := tileBaseIdx + int(xTile)*delta + int(yTile)*0x20
		if d1 {
			tileIdx = tileBaseIdx + (int(yTile)*xTiles+int(xTile))*delta
		}

		var c color.RGBA
		if colorMode == color16 {
			tileData := g.VRAM[0x10000+32*tileIdx:]
			cidx := int((tileData[chrY*4+chrX/2] >> (4 * (chrX % 2))) & 0b1111)
			c = g.paletteObjColor(paletteIdx, cidx)
		} else {
			tileData := g.VRAM[0x10000+64*(tileIdx/2):]
			cidx := int(tileData[chrY*8+chrX])
			c = g.paletteObjColor(-1, cidx)
		}

		win0Enable, win1Enable, objWinEnable := util.Bit(g.IO[DISPCNT+1], 5), util.Bit(g.IO[DISPCNT+1], 6), util.Bit(g.IO[DISPCNT+1], 7)
		if isObjWin && c.R > 0 {
			objWin[objX+x][vcount] = true
		} else if win0Enable && g.inWindow0(objX+x, vcount) {
			if g.IO[WININ]>>4&0b1 == 0b1 {
				set(screen, objX+x, vcount, c)
			}
		} else if win1Enable && g.inWindow1(objX+x, vcount) {
			if g.IO[WININ+1]>>4&0b1 == 0b1 {
				set(screen, objX+x, vcount, c)
			}
		} else if objWinEnable && g.inObjWindow(objX+x, vcount) {
			if g.IO[WINOUT+1]>>4&0b1 == 0b1 {
				set(screen, objX+x, vcount, c)
			}
		} else if !isObjWin {
			set(screen, objX+x, vcount, c)
		}

		ox += int32(pa)
		oy += int32(pc)
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
