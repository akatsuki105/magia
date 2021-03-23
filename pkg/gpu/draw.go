package gpu

import (
	"image"
	"image/color"
	"mettaur/pkg/util"
)

const (
	color16  = 0
	color256 = 1
)

func set(i *image.RGBA, x int, y int, c color.Color) {
	_, _, _, a := c.RGBA()
	if a == 0 {
		return
	}
	if x < 0 || x >= 240 {
		return
	}
	if y < 0 || y >= 160 {
		return
	}
	i.Set(x, y, c)
}

// Draw screen
func (g *GPU) Draw() *image.RGBA {
	mode := g.IO[DISPCNT] & 0b111
	switch mode {
	case 0:
		return g.draw0()
	case 1:
		return g.draw1()
	case 2:
		return g.draw2()
	case 3:
		return g.draw3()
	case 4:
		return g.draw4()
	case 5:
		return g.draw5()
	}

	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	return result
}

func (g *GPU) draw0() *image.RGBA {
	bgValid := 8
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	bgColor := g.bgColor()
	util.FillImage(result, bgColor)
	dispcnt := util.LE16(g.IO[DISPCNT:])

	// TODO: BGPriority
	if util.Bit(dispcnt, bgValid+3) {
		g.drawLayer(result, util.LE16(g.IO[BG3CNT:]))
	}
	if util.Bit(dispcnt, bgValid+2) {
		g.drawLayer(result, util.LE16(g.IO[BG2CNT:]))
	}
	if util.Bit(dispcnt, bgValid+1) {
		g.drawLayer(result, util.LE16(g.IO[BG1CNT:]))
	}
	if util.Bit(dispcnt, bgValid) {
		g.drawLayer(result, util.LE16(g.IO[BG0CNT:]))
	}
	return result
}

func (g *GPU) drawLayer(screen *image.RGBA, bgCnt uint16) {
	tileBlock, colorMode, mapBlock := ((uint32(bgCnt)>>2)&0b11)*0x4000, (bgCnt>>7)&0b1, ((uint32(bgCnt)>>8)&0b11111)*0x0800

	for yTile := uint32(0); yTile < 32; yTile++ {
		for xTile := uint32(0); xTile < 32; xTile++ {
			mapIdx := yTile*32 + xTile
			mapData := util.LE16(g.VRAM[(mapBlock + 2*(mapIdx)):])
			tileIdx, paletteIdx := uint32(mapData&0b0011_1111_1111), int((mapData>>12)&0b1111)
			if colorMode == color16 {
				tileData := g.VRAM[tileBlock+32*tileIdx:]
				for y := uint32(0); y < 8; y++ {
					for x := uint32(0); x < 8; x += 2 {
						c := tileData[y*4+x/2]
						set(screen, int(xTile*8+x), int(yTile*8+y), g.paletteColor(paletteIdx, int(c&0b1111)))
						set(screen, int(xTile*8+x+1), int(yTile*8+y), g.paletteColor(paletteIdx, int((c>>4)&0b1111)))
					}
				}
			} else {
				tileData := g.VRAM[tileBlock+64*tileIdx:]
				for y := uint32(0); y < 8; y++ {
					for x := uint32(0); x < 8; x++ {
						set(screen, int(xTile*8+x), int(yTile*8+y), g.paletteColor(-1, int(tileData[y*8+x])))
					}
				}
			}
		}
	}
}

func (g *GPU) draw1() *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	return result
}
func (g *GPU) draw2() *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	return result
}

func (g *GPU) draw3() *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))

	frameBuffer := g.VRAM[:80*kb]
	for y := 0; y < 160; y++ {
		for x := 0; x < 480; x += 2 {
			c := util.LE16(frameBuffer[x : x+2])
			r, g, b := byte((c&0b11111)*8), byte(((c>>5)&0b11111)*8), byte(((c>>10)&0b11111)*8)
			set(result, x/2, y, color.RGBA{r, g, b, 0xff})
		}
	}

	return result
}

func (g *GPU) draw4() *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	return result
}
func (g *GPU) draw5() *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	return result
}

func (g *GPU) paletteColor(palette, index int) color.RGBA {
	var c uint16
	if index == 0 {
		return color.RGBA{0x00, 0x00, 0x00, 0x00}
	}
	if palette >= 0 && palette < 16 {
		c = util.LE16(g.Palette[palette*32+index*2:])
	} else {
		c = util.LE16(g.Palette[index*2:])
	}
	red, blue, green := byte((c&0b11111)*8), byte(((c>>5)&0b11111)*8), byte(((c>>10)&0b11111)*8)
	return color.RGBA{red, blue, green, 0xff}
}
func (g *GPU) bgColor() color.RGBA {
	c := util.LE16(g.Palette[:])
	red, blue, green := byte((c&0b11111)*8), byte(((c>>5)&0b11111)*8), byte(((c>>10)&0b11111)*8)
	return color.RGBA{red, blue, green, 0xff}
}
