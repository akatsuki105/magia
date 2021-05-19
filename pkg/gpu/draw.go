package gpu

import (
	"image"
	"image/color"

	"github.com/pokemium/magia/pkg/util"

	"github.com/anthonynsimon/bild/blend"
)

const (
	color16  = 0
	color256 = 1
)

const (
	BlendOff = iota
	BlendAlpha
	BlendWhite
	BlendBlack
)

func set(i *image.RGBA, x int, y int, c color.Color) {
	_, _, _, a := c.RGBA()
	if a == 0 {
		return
	}
	i.Set(x, y, c)
}

var draws = [6](func(g *GPU) *image.RGBA){draw0, draw1, draw2, draw3, draw4, draw5}

func (g *GPU) Draw() *image.RGBA {
	mode := g.IO[DISPCNT] & 0b111
	return (draws[mode])(g)
}

func draw0(g *GPU) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	util.FillImage(result, g.bgColor())
	dispcnt := util.LE16(g.IO[DISPCNT:])

	// priority: 0 > 1 > 2 > 3
	for p := 3; p >= 0; p-- {
		// if priority equals: BG0 > BG1 > BG2 > BG3
		for bg := 3; bg >= 0; bg-- {
			prio := int(g.IO[(BG0CNT+2*bg)] & 0b11)
			if prio == p {
				if util.Bit(dispcnt, 8+bg) {
					g.drawTextBG(result, bg)
				}
			}
		}

		if util.Bit(dispcnt, 12) {
			g.drawObjs(result, p)
		}
	}

	return result
}

func draw1(g *GPU) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	util.FillImage(result, g.bgColor())
	dispcnt := util.LE16(g.IO[DISPCNT:])

	for p := 3; p >= 0; p-- {
		for bg := 2; bg >= 0; bg-- {
			prio := int(g.IO[(BG0CNT+2*bg)] & 0b11)
			if prio == p {
				if util.Bit(dispcnt, 8+bg) {
					if bg == 2 {
						g.drawAffineBG(result, bg)
					} else {
						g.drawTextBG(result, bg)
					}
				}
			}
		}

		if util.Bit(dispcnt, 12) {
			g.drawObjs(result, p)
		}
	}

	return result
}

func draw2(g *GPU) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	util.FillImage(result, g.bgColor())
	dispcnt := util.LE16(g.IO[DISPCNT:])

	// blend
	bldCnt := g.IO[BLDCNT]
	blend := (bldCnt >> 6) & 0b11

	for p := 3; p >= 0; p-- {
		for bg := 3; bg >= 2; bg-- {
			prio := int(g.IO[(BG0CNT+2*bg)] & 0b11)
			if prio == p {
				if util.Bit(dispcnt, 8+bg) {
					if blend > 0 && util.Bit(bldCnt, bg) {
						layer := image.NewRGBA(image.Rect(0, 0, 240, 160))
						g.drawAffineBG(layer, bg)
						switch blend {
						case BlendAlpha:
							result = g.blendAlpha1(result, layer)
						case BlendWhite:
							layer = g.blendWhite(layer)
							g.override(result, layer)
						case BlendBlack:
							layer = g.blendBlack(layer)
							g.override(result, layer)
						}
					} else if blend == BlendAlpha && util.Bit(bldCnt, bg+8) {
						layer := image.NewRGBA(image.Rect(0, 0, 240, 160))
						g.drawAffineBG(layer, bg)
						result = g.blendAlpha2(result, layer)
					} else {
						g.drawAffineBG(result, bg)
					}
				}
			}
		}

		if util.Bit(dispcnt, 12) {
			g.drawObjs(result, p)
		}
	}

	return result
}

func draw3(g *GPU) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	util.FillImage(result, g.bgColor())

	frameBuffer := g.VRAM[:80*kb]
	for y := 0; y < 160; y++ {
		for x := 0; x < 480; x += 2 {
			c := util.LE16(frameBuffer[y*480+x:])
			r, g, b := byte((c&0b11111)*8), byte(((c>>5)&0b11111)*8), byte(((c>>10)&0b11111)*8)
			set(result, x/2, y, color.RGBA{r, g, b, 0xff})
		}
	}

	return result
}

func draw4(g *GPU) *image.RGBA {
	frame1 := util.Bit(g.IO[DISPCNT], 4)
	frameBuffer := g.VRAM[:0xa000]
	if frame1 {
		frameBuffer = g.VRAM[0xa000:]
	}

	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	util.FillImage(result, g.bgColor())
	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			c := g.paletteColor(-1, int(frameBuffer[y*240+x]))
			set(result, x, y, c)
		}
	}
	return result
}

func draw5(g *GPU) *image.RGBA {
	panic("unsupported BG mode 5")
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

func (g *GPU) override(src, layer *image.RGBA) {
	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			set(src, x, y, layer.At(x, y))
		}
	}
}

func (g *GPU) blendAlpha1(src, layer *image.RGBA) *image.RGBA {
	eva := uint32(g.IO[BLDALPHA] & 0b11111)
	if eva > 16 {
		eva = 16
	}
	return blend.Opacity(src, layer, float64(eva)/16/1.5)
}

func (g *GPU) blendAlpha2(src, layer *image.RGBA) *image.RGBA {
	eva := uint32(g.IO[BLDALPHA+1] & 0b11111)
	if eva > 16 {
		eva = 16
	}
	return blend.Opacity(src, layer, float64(eva)/16/1.5)
}

func (g *GPU) blendWhite(layer *image.RGBA) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	evy := uint32(g.IO[BLDY] & 0b11111)
	if evy > 16 {
		evy = 16
	}

	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			oldR, oldG, oldB, _ := layer.At(x, y).RGBA()
			oldR, oldG, oldB = oldR>>8, oldG>>8, oldB>>8
			r, g, b := byte(oldR+(248-oldR)*evy/16), byte(oldG+(248-oldG)*evy/16), byte(oldB+(248-oldB)*evy/16)
			c := color.RGBA{r, g, b, 0xff}
			set(result, x, y, c)
		}
	}
	return result
}

func (g *GPU) blendBlack(layer *image.RGBA) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	evy := uint32(g.IO[BLDY] & 0b11111)
	if evy > 16 {
		evy = 16
	}

	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			oldR, oldG, oldB, _ := layer.At(x, y).RGBA()
			oldR, oldG, oldB = oldR>>8, oldG>>8, oldB>>8
			r, g, b := byte(oldR-oldR*evy/16), byte(oldG-oldG*evy/16), byte(oldB-oldB*evy/16)
			c := color.RGBA{r, g, b, 0xff}
			set(result, x, y, c)
		}
	}
	return result
}
