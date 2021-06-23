package video

import (
	"github.com/pokemium/magia/pkg/util"
)

type ObjShape int

const (
	square ObjShape = iota
	horizontal
	vertical
)

type Obj struct {
	oam   *OAM
	index int

	// xCoord(Atr1 bit0-8), yCoord(Atr0 bit0-7)
	x, y uint16

	scalerot      bool
	doublesize    bool
	disable       bool
	mode          uint16
	mosaic        bool
	color256      bool
	shape         ObjShape
	scalerotParam uint16
	hflip, vflip  bool
	priority      int
	tileBase      uint32
	palette       uint16

	// ? [0-512]
	cachedWidth uint16

	// ? [0-255]
	cachedHeight uint16

	scalerotOam Scalerot
	size        int
	isAffine    bool
}

func newObj(oam *OAM, index int) *Obj {
	return &Obj{
		oam:          oam,
		index:        index,
		disable:      true,
		cachedWidth:  8,
		cachedHeight: 8,
	}
}

func (o *Obj) drawScanline(backing *Backing, y uint32, yOff int32, start, end uint32) {
	if o.isAffine {
		o.drawScanlineAffine(backing, uint16(y), yOff, uint16(start), uint16(end))
		return
	}
	o.drawScanlineNormal(backing, y, yOff, uint16(start), uint16(end))
}

const TILE_OFFSET = 0x10000

func (o *Obj) drawScanlineNormal(backing *Backing, y uint32, yOff int32, start, end uint16) {
	video := o.oam.video
	x := uint16(0)

	mask := byte(o.mode) | video.target2[LAYER_OBJ] | byte(o.priority)<<1
	if o.mode == 0x10 {
		mask |= TARGET1_MASK
	}
	if video.blendMode == ALPHA_BLEND && video.alphaEnabled {
		mask |= video.target1[LAYER_OBJ]
	}

	totalWidth := o.cachedWidth
	underflow, offset := uint16(0), uint16(0)
	if o.x < HORIZONTAL_PIXELS {
		underflow, offset = 0, o.x
		if o.x < start {
			underflow = start - o.x
			offset = start
		}

		if end < o.cachedWidth+o.x {
			totalWidth = end - o.x
		}
	} else {
		underflow = start + 512 - o.x
		offset = start
		if end < o.cachedWidth-underflow && o.cachedWidth > underflow {
			totalWidth = end
		}
	}

	localY := uint32(int32(y) - yOff)
	if o.vflip {
		localY = uint32(int32(o.cachedHeight) - int32(y) + yOff - 1)
	}

	localYLo := localY & 0x7
	paletteShift := util.BoolToU32(o.color256)

	tileOffset := (localY & 0x01f8) << (2 - paletteShift)
	if video.objCharacterMapping {
		tileOffset = ((localY & 0x01f8) * uint32(o.cachedWidth)) >> 6
	}

	mosaicX := uint16(0)
	if o.mosaic {
		mosaicX = video.objMosaicX - 1 - ((video.objMosaicX + offset - 1) % video.objMosaicX)
		offset += mosaicX
		underflow += mosaicX
	}

	localX := underflow
	if o.hflip {
		localX = o.cachedWidth - underflow - 1
	}

	tileRow := video.accessTile(
		TILE_OFFSET+(uint32(x)&0x4)*paletteShift,
		o.tileBase+(tileOffset<<paletteShift)+(uint32(localX&0x01f8)>>(3-paletteShift)),
		localYLo<<paletteShift,
	)

	for x = underflow; x < totalWidth; x++ {
		mosaicX = 0
		if o.mosaic {
			mosaicX = offset % video.objMosaicX
		}

		localX = x - mosaicX
		if o.hflip {
			localX = o.cachedWidth - (x - mosaicX) - 1
		}

		if paletteShift == 0 {
			if (x&0x7 == 0) || (o.mosaic && mosaicX == 0) {
				tileRow = video.accessTile(
					TILE_OFFSET,
					o.tileBase+tileOffset+uint32(localX>>3),
					localYLo,
				)
			}
		} else {
			if (x&0x3 == 0) || (o.mosaic && mosaicX == 0) {
				tileRow = video.accessTile(
					TILE_OFFSET+uint32(localX&0x4),
					o.tileBase+(tileOffset<<1)+uint32((localX&0x01f8)>>2),
					localYLo<<1,
				)
			}
		}
		o.pushPixel(LAYER_OBJ, o, video, tileRow, uint32(localX&0x7), uint32(offset), backing, mask)
		offset++
	}
}

func (o *Obj) drawScanlineAffine(backing *Backing, y uint16, yOff int32, start, end uint16) {
	video := o.oam.video
	mask := byte(o.mode) | video.target2[LAYER_OBJ] | byte(o.priority<<1)
	if o.mode == 0x10 {
		mask |= TARGET1_MASK
	}
	if video.blendMode == ALPHA_BLEND && video.alphaEnabled {
		mask |= video.target1[LAYER_OBJ]
	}

	yDiff := uint16(int32(y) - yOff)

	paletteShift := util.BoolToU32(o.color256)
	totalWidth := o.cachedWidth << util.BoolToU16(o.doublesize)
	totalHeight := o.cachedHeight << util.BoolToU16(o.doublesize)
	drawWidth := totalWidth
	if drawWidth > HORIZONTAL_PIXELS {
		totalWidth = HORIZONTAL_PIXELS
	}

	underflow, offset := uint16(0), uint16(0)
	if o.x < HORIZONTAL_PIXELS {
		if o.x < start {
			underflow = start - o.x
			offset = start
		} else {
			underflow = 0
			offset = o.x
		}
		if end < drawWidth+o.x {
			drawWidth = end - o.x
		}
	} else {
		underflow = start + 512 - o.x
		offset = start
		if end < drawWidth-underflow {
			drawWidth = end
		}
	}

	tileOffset := uint16(0)
	localX, localY := float64(0), float64(0)
	for x := underflow; x < drawWidth; x++ {
		localX = o.scalerotOam[0]*float64((x-(totalWidth>>1))) + o.scalerotOam[1]*float64(yDiff-(totalHeight>>1)) + float64(o.cachedWidth>>1)
		localY = o.scalerotOam[2]*float64(x-(totalWidth>>1)) + o.scalerotOam[3]*float64(yDiff-(totalHeight>>1)) + float64(o.cachedHeight>>1)

		if o.mosaic {
			localX -= float64(x%video.objMosaicX)*o.scalerotOam[0] + float64(y%video.objMosaicY)*o.scalerotOam[1]
			localY -= float64(x%video.objMosaicX)*o.scalerotOam[2] + float64(y%video.objMosaicY)*o.scalerotOam[3]
		}

		if localX < 0 || uint16(localX) >= o.cachedWidth || localY < 0 || uint16(localY) >= o.cachedHeight {
			offset++
			continue
		}

		tileOffset = (uint16(localY) & 0x01f8) << (2 - paletteShift)
		if video.objCharacterMapping {
			tileOffset = ((uint16(localY) & 0x01f8) * o.cachedWidth) >> 6
		}

		tileRow := video.accessTile(
			TILE_OFFSET+(uint32(localX)&0x4)*paletteShift,
			o.tileBase+(uint32(tileOffset)<<paletteShift)+((uint32(localX)&0x01f8)>>(3-paletteShift)),
			(uint32(localY)&0x7)<<paletteShift,
		)

		o.pushPixel(LAYER_OBJ, o, video, tileRow, (uint32(localX) & 0x7), uint32(offset), backing, mask)
		offset++
	}
}

func (o *Obj) recalcSize() {
	switch o.shape {
	case square:
		o.cachedHeight = 8 << o.size
		o.cachedWidth = 8 << o.size
	case horizontal:
		hw := [4][2]uint16{{8, 16}, {8, 32}, {16, 32}, {32, 64}}
		o.cachedHeight = hw[o.size][0]
		o.cachedWidth = hw[o.size][1]
	case vertical:
		hw := [4][2]uint16{{16, 8}, {32, 8}, {32, 16}, {64, 32}}
		o.cachedHeight = hw[o.size][0]
		o.cachedWidth = hw[o.size][1]
	}
}

func (o *Obj) pushPixel(layer int, objMap *Obj, video *SoftwareRenderer, row, x, offset uint32, backing *Backing, mask byte) {
	// palette index
	index := uint32(0)
	if o.color256 {
		// 256 color
		if x >= 4 {
			x -= 4
		}
		index = (row >> (x << 3)) & 0xff
	} else {
		// 16 color
		index = (row >> (x << 2)) & 0xf
	}

	// Index 0 is transparent
	if index == 0 {
		return
	} else if !o.color256 {
		index |= uint32(objMap.palette)
	}

	stencil := byte(WRITTEN_MASK)
	oldStencil := backing.stencil[offset]
	blend := video.blendMode
	if video.objwinActive {
		if oldStencil&OBJWIN_MASK > 0 {
			if video.windows[3].enabled[layer] {
				video.setBlendEnabled(layer, video.windows[3].special && video.target1[layer] > 0, blend)
				if video.windows[3].special && video.alphaEnabled {
					mask |= video.target1[layer]
				}
				stencil |= OBJWIN_MASK
			} else {
				return
			}
		} else if video.windows[2].enabled[layer] {
			video.setBlendEnabled(layer, video.windows[2].special && video.target1[layer] > 0, blend)
			if video.windows[2].special && video.alphaEnabled {
				mask |= video.target1[layer]
			}
		} else {
			return
		}
	}

	if (mask&TARGET1_MASK > 0) && (oldStencil&TARGET2_MASK > 0) {
		video.setBlendEnabled(layer, true, 1)
	}

	pixel := video.Palette.accessColor(layer, int(index))

	if mask&TARGET1_MASK > 0 {
		video.setBlendEnabled(layer, blend != 0, blend)
	}

	highPriority := (mask & PRIORITY_MASK) < (oldStencil & PRIORITY_MASK)
	// Backgrounds can draw over each other, too.
	if (mask & PRIORITY_MASK) == (oldStencil & PRIORITY_MASK) {
		highPriority = (mask & BACKGROUND_MASK) > 0
	}

	if (oldStencil & WRITTEN_MASK) == 0 {
		// Nothing here yet, just continue
		stencil |= mask
	} else if highPriority {
		// We are higher priority
		if (mask&TARGET1_MASK != 0) && (oldStencil&TARGET2_MASK != 0) {
			pixel = video.Palette.mix(video.blendA, pixel, video.blendB, backing.color[offset])
		}
		// We just drew over something, so it doesn't make sense for us to be a TARGET1 anymore...
		stencil |= mask & ^byte(TARGET1_MASK)
	} else if (mask & PRIORITY_MASK) > (oldStencil & PRIORITY_MASK) {
		// We're below another layer, but might be the blend target for it
		stencil = oldStencil & ^byte(TARGET1_MASK|TARGET2_MASK)
		if (mask&TARGET2_MASK != 0) && (oldStencil&TARGET1_MASK != 0) {
			pixel = video.Palette.mix(video.blendB, pixel, video.blendA, backing.color[offset])
		} else {
			return
		}
	} else {
		return
	}

	if (mask & OBJWIN_MASK) > 0 {
		// We ARE the object window, don't draw pixels!
		backing.stencil[offset] |= OBJWIN_MASK

		return
	}
	backing.color[offset] = pixel
	backing.stencil[offset] = stencil
}
