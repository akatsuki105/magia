package video

import (
	"fmt"
	"math"

	"github.com/pokemium/magia/pkg/util"
)

type PaletteMode int

const (
	modeNormal PaletteMode = iota
	modeBright
	modeDark
)

type Palette struct {
	colors            [2][]uint16
	adjustedColors    [2][]uint16
	passthroughColors [6][]uint16
	blendY            float64

	mode PaletteMode
}

func (p *Palette) String() string {
	return fmt.Sprintf("%+v", p.colors)
}

func NewPalette() *Palette {
	colors := [2][]uint16{make([]uint16, 0x100), make([]uint16, 0x100)}
	adjustedColors := [2][]uint16{make([]uint16, 0x100), make([]uint16, 0x100)}
	passthroughColors := [6][]uint16{
		colors[0],
		colors[0],
		colors[0],
		colors[0],
		colors[1],
		colors[0],
	}
	return &Palette{
		colors:            colors,
		adjustedColors:    adjustedColors,
		passthroughColors: passthroughColors,
		blendY:            1,
	}
}

func (p *Palette) adjustColor(color uint16) uint16 {
	if p.mode == modeDark {
		return p.adjustColorDark(color)
	}
	return p.adjustColorBright(color)
}

func (p *Palette) Load8(offset uint32) byte {
	c16 := p.Load16(offset)
	return byte((c16 >> (8 * (offset & 1))) & 0xff)
}

func (p *Palette) Load16(offset uint32) uint16 {
	return p.colors[(offset&0x200)>>9][(offset&0x1ff)>>1]
}

func (p *Palette) Load32(offset uint32) uint32 {
	return (uint32(p.Load16(offset+2)) << 16) | uint32(p.Load16(offset))
}

func (p *Palette) Store16(offset uint32, value uint16) {
	colorType := (offset & 0x200) >> 9
	colorIdx := (offset & 0x1ff) >> 1
	p.colors[colorType][colorIdx] = value
	p.adjustedColors[colorType][colorIdx] = p.adjustColor(value)
}

func (p *Palette) Store32(offset uint32, value uint32) {
	p.Store16(offset, uint16(value))
	p.Store16(offset+2, uint16(value>>16))
}

func (p *Palette) invalidatePage(addr uint32) { return }

func (p *Palette) convert16To32(value uint16) [3]byte {
	r := (value & 0x001f) << 3
	g := (value & 0x03e0) >> 2
	b := (value & 0x7c00) >> 7

	return [3]byte{byte(r), byte(g), byte(b)}
}

func (p *Palette) mix(aWeight float64, aColor uint16, bWeight float64, bColor uint16) uint16 {
	ar := float64(aColor & 0x001f)
	ag := float64((aColor & 0x03e0) >> 5)
	ab := float64((aColor & 0x7c00) >> 10)

	br := float64(bColor & 0x001f)
	bg := float64((bColor & 0x03e0) >> 5)
	bb := float64((bColor & 0x7c00) >> 10)

	r := uint16(math.Min(aWeight*ar+bWeight*br, 0x1f))
	g := uint16(math.Min(aWeight*ag+bWeight*bg, 0x1f))
	b := uint16(math.Min(aWeight*ab+bWeight*bb, 0x1f))

	return r | (g << 5) | (b << 10)
}

func (p *Palette) makeDarkPalettes(layers uint16) {
	if p.mode != modeDark {
		p.mode = modeDark
		p.resetPalettes()
	}
	p.resetPaletteLayers(layers)
}

func (p *Palette) makeBrightPalettes(layers uint16) {
	if p.mode != modeBright {
		p.mode = modeBright
		p.resetPalettes()
	}
	p.resetPaletteLayers(layers)
}

func (p *Palette) makeNormalPalettes() {
	p.passthroughColors[0] = p.colors[0]
	p.passthroughColors[1] = p.colors[0]
	p.passthroughColors[2] = p.colors[0]
	p.passthroughColors[3] = p.colors[0]
	p.passthroughColors[4] = p.colors[1]
	p.passthroughColors[5] = p.colors[0]
}

func (p *Palette) makeSpecialPalette(layer int) {
	idx := map[int]int{4: 1}[layer]
	p.passthroughColors[layer] = p.adjustedColors[idx]
}

func (p *Palette) makeNormalPalette(layer int) {
	idx := map[int]int{4: 1}[layer]
	p.passthroughColors[layer] = p.colors[idx]
}

func (p *Palette) resetPaletteLayers(layers uint16) {
	if util.Bit(layers, 0) {
		p.passthroughColors[0] = p.adjustedColors[0]
	} else {
		p.passthroughColors[0] = p.colors[0]
	}

	if util.Bit(layers, 1) {
		p.passthroughColors[1] = p.adjustedColors[0]
	} else {
		p.passthroughColors[1] = p.colors[0]
	}

	if util.Bit(layers, 2) {
		p.passthroughColors[2] = p.adjustedColors[0]
	} else {
		p.passthroughColors[2] = p.colors[0]
	}

	if util.Bit(layers, 3) {
		p.passthroughColors[3] = p.adjustedColors[0]
	} else {
		p.passthroughColors[3] = p.colors[0]
	}

	if util.Bit(layers, 4) {
		p.passthroughColors[4] = p.adjustedColors[1]
	} else {
		p.passthroughColors[4] = p.colors[1]
	}

	if util.Bit(layers, 5) {
		p.passthroughColors[5] = p.adjustedColors[0]
	} else {
		p.passthroughColors[5] = p.colors[0]
	}
}

func (p *Palette) resetPalettes() {
	outPalette := p.adjustedColors[0]
	inPalette := p.colors[0]
	for i := 0; i < 256; i++ {
		outPalette[i] = p.adjustColor(inPalette[i])
	}

	outPalette = p.adjustedColors[1]
	inPalette = p.colors[1]
	for i := 0; i < 256; i++ {
		outPalette[i] = p.adjustColor(inPalette[i])
	}
}

func (p *Palette) accessColor(layer, index int) uint16 {
	return p.passthroughColors[layer][index]
}

func (p *Palette) adjustColorDark(color uint16) uint16 {
	r := float64(color & 0b0000_0000_0001_1111)
	g := float64((color & 0b0000_0011_1110_0000) >> 5)
	b := float64((color & 0b0111_1100_0000_0000) >> 10)

	r16 := uint16(r - r*p.blendY)
	g16 := uint16(g - g*p.blendY)
	b16 := uint16(b - b*p.blendY)

	return r16 | (g16 << 5) | (b16 << 10)
}

func (p *Palette) adjustColorBright(color uint16) uint16 {
	r := float64(color & 0b0000_0000_0001_1111)
	g := float64((color & 0b0000_0011_1110_0000) >> 5)
	b := float64((color & 0b0111_1100_0000_0000) >> 10)

	r16 := uint16(r + (31-r)*p.blendY)
	g16 := uint16(g + (31-g)*p.blendY)
	b16 := uint16(b + (31-b)*p.blendY)

	return r16 | (g16 << 5) | (b16 << 10)
}

func (p *Palette) setBlendY(y float64) {
	if p.blendY != y {
		p.blendY = y
		p.resetPalettes()
	}
}
