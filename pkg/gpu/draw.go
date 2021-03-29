package gpu

import (
	"fmt"
	"image"
	"image/color"
	"mettaur/pkg/util"
	"os"
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
	}

	if util.Bit(dispcnt, 12) {
		g.drawObjs(result)
	}

	return result
}

func (g *GPU) draw1() *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	util.FillImage(result, g.bgColor())
	dispcnt := util.LE16(g.IO[DISPCNT:])

	for p := 3; p >= 0; p-- {
		for bg := 1; bg >= 0; bg-- {
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
	}

	if util.Bit(dispcnt, 12) {
		g.drawObjs(result)
	}

	return result
}

func (g *GPU) draw2() *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	util.FillImage(result, g.bgColor())
	dispcnt := util.LE16(g.IO[DISPCNT:])

	for p := 3; p >= 0; p-- {
		for bg := 3; bg >= 2; bg-- {
			prio := int(g.IO[(BG0CNT+2*bg)] & 0b11)
			if prio == p {
				if util.Bit(dispcnt, 8+bg) {
					g.drawAffineBG(result, bg)
				}
			}
		}
	}

	if util.Bit(dispcnt, 12) {
		g.drawObjs(result)
	}

	return result
}

func (g *GPU) draw3() *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 240, 160))

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

func (g *GPU) draw4() *image.RGBA {
	frame := (g.IO[DISPCNT] >> 4) & 0b1
	frameBuffer := g.VRAM[:0xa000]
	if frame == 1 {
		frameBuffer = g.VRAM[0xa000:]
	}

	result := image.NewRGBA(image.Rect(0, 0, 240, 160))
	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			c := g.paletteColor(-1, int(frameBuffer[y*240+x]))
			set(result, x, y, c)
		}
	}
	return result
}

func (g *GPU) draw5() *image.RGBA {
	fmt.Fprintf(os.Stderr, "unsupported BG mode 5\n")
	panic("")

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
