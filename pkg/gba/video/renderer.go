package video

import (
	"math"
	"sort"

	"github.com/pokemium/magia/pkg/util"
)

const LAYER_OBJ = 4
const LAYER_BACKDROP = 5

const BACKGROUND_MASK = 0b0001
const TARGET1_MASK = 0x10
const TARGET2_MASK = 0x08
const OBJWIN_MASK = 0x20
const WRITTEN_MASK = 0x80
const PRIORITY_MASK = 0b0111

// Screen pixel data
type Backing struct {
	color []uint16

	// Stencil format:
	//
	// - Bits 0-1: Layer
	//
	// - Bit 2: Is background
	//
	// - Bit 3: Is Target 2
	//
	// - Bit 4: Is Target 1
	//
	// - Bit 5: Is OBJ Window
	//
	// - Bit 6: Reserved
	//
	// - Bit 7: Has been written
	stencil []byte
}

// BG Map data
type SharedMap struct {
	tile         uint16 // tile index (0-1023)
	hflip, vflip bool
	palette      uint16 // palette index
}

type ImageData []byte

type WindowInfo struct {
	enabled [6]bool
	special bool
}

type BlendEffect uint16

const (
	NO_BLEND BlendEffect = iota
	ALPHA_BLEND
	WHITE_FADE
	BLACK_FADE
)

type SoftwareRenderer struct {
	drawBackdrop *Backdrop

	Palette     *Palette
	VRAM        *VRAM
	OAM         *OAM
	objLayers   [4]*ObjLayer
	objwinLayer *ObjLayer

	// current BG Mode (0-5)
	bgMode uint16

	// false: Frame 0, true: Frame 1 (for BG Modes 4,5 only)
	displayFrameSelect bool

	// true: Allow access to OAM during H-Blank
	hblankIntervalFree bool

	// false: Two dimensional, true: One dimensional
	objCharacterMapping bool
	forcedBlank         bool
	win0, win1, objwin  bool
	Vcount              uint16

	// Window
	win0Left, win0Right, win1Left, win1Right uint16
	win0Top, win0Bottom, win1Top, win1Bottom uint16
	windows                                  [4]*WindowInfo

	// 0b0000 or 0b10000 (BLDCNT's bit 0-5)
	// 0 -> BG0 is 1st blend target, 1 -> BG1, 2 -> BG2, ...
	target1 [6]byte

	// 0b0000 or 0b1000 (BLDCNT's bit 8-13)
	// 0 -> BG0 is 2nd blend target, 1 -> BG1, 2 -> BG2, ...
	target2 [6]byte

	// BLDCNT's bit 6-7
	blendMode BlendEffect

	blendA, blendB, blendY float64

	// BG Mosaic size (0 ~ 7)
	bgMosaicX, bgMosaicY uint16

	// Obj Mosaic size (0 ~ 7)
	objMosaicX, objMosaicY uint16

	bg           [4]*BGLayer
	bgModes      [6](func(backing *Backing, bg *BGLayer, start uint32, end uint32))
	objwinActive bool
	drawLayers   Layers
	alphaEnabled bool
	scanline     Backing

	sharedMap SharedMap
	pixelData ImageData
}

func NewSoftwareRenderer() *SoftwareRenderer {
	s := &SoftwareRenderer{
		forcedBlank: true,
		win0Right:   240,
		win1Right:   240,
		win0Bottom:  160,
		win1Bottom:  160,
		bgMosaicX:   1,
		bgMosaicY:   1,
		objMosaicX:  1,
		objMosaicY:  1,
		Vcount:      0,
	}
	s.drawBackdrop = NewBackdrop(s)

	s.Palette = NewPalette()
	s.VRAM = NewVRAM(util.RegionSize["VRAM"])
	s.OAM = NewOAM(util.RegionSize["OAM"])
	s.OAM.video = s

	s.objLayers = [4]*ObjLayer{
		NewObjLayer(s, 0),
		NewObjLayer(s, 1),
		NewObjLayer(s, 2),
		NewObjLayer(s, 3),
	}
	s.objwinLayer = NewObjLayer(s, 4)
	s.objwinLayer.objwin = OBJWIN_MASK

	for i := 0; i < len(s.windows); i++ {
		s.windows[i] = &WindowInfo{
			enabled: [6]bool{false, false, false, false, false, true},
			special: false,
		}
	}

	for i := 0; i < len(s.bg); i++ {
		s.bg[i] = &BGLayer{
			bg:               true,
			index:            i,
			video:            s,
			vram:             s.VRAM,
			dx:               1,
			dmy:              1,
			drawScanlineFunc: drawScanlineBGMode0,
		}
	}

	s.bgModes = [6](func(backing *Backing, bg *BGLayer, start uint32, end uint32)){
		drawScanlineBGMode0,
		drawScanlineBGMode2,
		drawScanlineBGMode2,
		drawScanlineBGMode3,
		drawScanlineBGMode4,
		drawScanlineBGMode5,
	}

	s.drawLayers = [10]Layer{
		s.bg[0],
		s.bg[1],
		s.bg[2],
		s.bg[3],
		s.objLayers[0],
		s.objLayers[1],
		s.objLayers[2],
		s.objLayers[3],
		s.objwinLayer,
		s.drawBackdrop,
	}

	s.scanline = Backing{
		color:   make([]uint16, HORIZONTAL_PIXELS),
		stencil: make([]byte, HORIZONTAL_PIXELS),
	}

	s.setBacking(make([]byte, HORIZONTAL_PIXELS*VERTICAL_PIXELS*4))

	return s
}

func (s *SoftwareRenderer) setBacking(backing ImageData) {
	s.pixelData = backing

	// Clear backing first
	for i := 0; i < HORIZONTAL_PIXELS*VERTICAL_PIXELS*4; i += 4 {
		s.pixelData[i] = 0xff
		s.pixelData[i+1] = 0xff
		s.pixelData[i+2] = 0xff
		s.pixelData[i+3] = 0xff
	}
}

// DISPCNT
func (s *SoftwareRenderer) writeDisplayControl(value uint16) {
	s.bgMode = value & 0b0111
	s.displayFrameSelect = util.Bit(value, 4)
	s.hblankIntervalFree = util.Bit(value, 5)
	s.objCharacterMapping = util.Bit(value, 6)
	s.forcedBlank = util.Bit(value, 7)

	s.bg[0].enabled = util.Bit(value, 8)
	s.bg[1].enabled = util.Bit(value, 9)
	s.bg[2].enabled = util.Bit(value, 10)
	s.bg[3].enabled = util.Bit(value, 11)
	s.objLayers[0].enabled = util.Bit(value, 12)
	s.objLayers[1].enabled = util.Bit(value, 12)
	s.objLayers[2].enabled = util.Bit(value, 12)
	s.objLayers[3].enabled = util.Bit(value, 12)

	s.win0 = util.Bit(value, 13)
	s.win1 = util.Bit(value, 14)
	s.objwin = util.Bit(value, 15)
	if s.objwinLayer != nil {
		s.objwinLayer.enabled = util.Bit(value, 12) && util.Bit(value, 15)
	}

	// Total hack so we can store both things that would set it to 256-color mode in the same variable
	s.bg[2].color256 &= ^uint16(0x0001)
	s.bg[3].color256 &= ^uint16(0x0001)

	// BG2 is limited to 256-color when BG Mode is 1, 2, 3, 4, 5
	if s.bgMode > 0 {
		s.bg[2].color256 |= 0x0001
	}

	// BG3 is limited to 256-color in BG Mode2
	if s.bgMode == 2 {
		s.bg[3].color256 |= 0x0001
	}

	s.resetLayers()
}

// BGnCNT
func (s *SoftwareRenderer) writeBackgroundControl(bg, value uint16) {
	bgData := s.bg[bg]
	bgData.priority = int(value & 0x0003)
	bgData.charBase = (uint32(value) & 0x000c) << 12
	bgData.mosaic = util.Bit(value, 6)

	bgData.color256 &= ^uint16(0x0080)
	if bg < 2 || s.bgMode == 0 {
		bgData.color256 |= value & 0x0080
	}

	bgData.screenBase = (uint32(value) & 0x1f00) * 8
	bgData.overflow = util.Bit(value, 13)
	bgData.size = (uint32(value) & 0xc000) >> 14

	sort.Sort(&s.drawLayers)
}

// BGnHOFS
func (s *SoftwareRenderer) writeBackgroundHOffset(bg, value uint16) { s.bg[bg].x = value & 0x1ff }

// BGnVOFS
func (s *SoftwareRenderer) writeBackgroundVOffset(bg, value uint16) { s.bg[bg].y = value & 0x1ff }

func (s *SoftwareRenderer) writeBackgroundRefX(bg, value uint32) {
	s.bg[bg].refx = float64(value<<4) / 0x1000
	s.bg[bg].sx = s.bg[bg].refx
}
func (s *SoftwareRenderer) writeBackgroundRefY(bg, value uint32) {
	s.bg[bg].refy = float64(value<<4) / 0x1000
	s.bg[bg].sy = s.bg[bg].refy
}

func (s *SoftwareRenderer) writeBackgroundParamA(bg, value uint16) {
	s.bg[bg].dx = float64(uint32(value)<<16) / 0x0100_0000
}

func (s *SoftwareRenderer) writeBackgroundParamB(bg, value uint16) {
	s.bg[bg].dmx = float64(uint32(value)<<16) / 0x0100_0000
}

func (s *SoftwareRenderer) writeBackgroundParamC(bg, value uint16) {
	s.bg[bg].dy = float64(uint32(value)<<16) / 0x0100_0000
}

func (s *SoftwareRenderer) writeBackgroundParamD(bg, value uint16) {
	s.bg[bg].dmy = float64(uint32(value)<<16) / 0x0100_0000
}

func (s *SoftwareRenderer) writeWin0H(value uint16) {
	s.win0Left = (value & 0xff00) >> 8
	s.win0Right = uint16(math.Min(HORIZONTAL_PIXELS, float64(value&0x00ff)))
	if s.win0Left > s.win0Right {
		s.win0Right = HORIZONTAL_PIXELS
	}
}

func (s *SoftwareRenderer) writeWin1H(value uint16) {
	s.win1Left = (value & 0xff00) >> 8
	s.win1Right = uint16(math.Min(HORIZONTAL_PIXELS, float64(value&0x00ff)))
	if s.win1Left > s.win1Right {
		s.win1Right = HORIZONTAL_PIXELS
	}
}

func (s *SoftwareRenderer) writeWin0V(value uint16) {
	s.win0Top = (value & 0xff00) >> 8
	s.win0Bottom = uint16(math.Min(VERTICAL_PIXELS, float64(value&0x00ff)))
	if s.win0Top > s.win0Bottom {
		s.win0Bottom = VERTICAL_PIXELS
	}
}

func (s *SoftwareRenderer) writeWin1V(value uint16) {
	s.win1Top = (value & 0xff00) >> 8
	s.win1Bottom = uint16(math.Min(VERTICAL_PIXELS, float64(value&0x00ff)))
	if s.win1Top > s.win1Bottom {
		s.win1Bottom = VERTICAL_PIXELS
	}
}

func (s *SoftwareRenderer) writeWindow(index, value byte) {
	window := s.windows[index]
	window.enabled[0] = util.Bit(value, 0)
	window.enabled[1] = util.Bit(value, 1)
	window.enabled[2] = util.Bit(value, 2)
	window.enabled[3] = util.Bit(value, 3)
	window.enabled[4] = util.Bit(value, 4)
	window.special = util.Bit(value, 5)
}

func (s *SoftwareRenderer) writeWinIn(value uint16) {
	s.writeWindow(0, byte(value))
	s.writeWindow(1, byte(value>>8))
}

func (s *SoftwareRenderer) writeWinOut(value uint16) {
	s.writeWindow(2, byte(value))
	s.writeWindow(3, byte(value>>8))
}

func (s *SoftwareRenderer) writeBlendControl(value uint16) {
	s.target1[0] = util.BoolToU8(util.Bit(value, 0)) * TARGET1_MASK
	s.target1[1] = util.BoolToU8(util.Bit(value, 1)) * TARGET1_MASK
	s.target1[2] = util.BoolToU8(util.Bit(value, 2)) * TARGET1_MASK
	s.target1[3] = util.BoolToU8(util.Bit(value, 3)) * TARGET1_MASK
	s.target1[4] = util.BoolToU8(util.Bit(value, 4)) * TARGET1_MASK
	s.target1[5] = util.BoolToU8(util.Bit(value, 5)) * TARGET1_MASK

	s.target2[0] = util.BoolToU8(util.Bit(value, 8)) * TARGET2_MASK
	s.target2[1] = util.BoolToU8(util.Bit(value, 9)) * TARGET2_MASK
	s.target2[2] = util.BoolToU8(util.Bit(value, 10)) * TARGET2_MASK
	s.target2[3] = util.BoolToU8(util.Bit(value, 11)) * TARGET2_MASK
	s.target2[4] = util.BoolToU8(util.Bit(value, 12)) * TARGET2_MASK
	s.target2[5] = util.BoolToU8(util.Bit(value, 13)) * TARGET2_MASK

	s.blendMode = BlendEffect((value & 0x00c0) >> 6)

	switch s.blendMode {
	case 0, 1:
		s.Palette.makeNormalPalettes()
	case 2:
		s.Palette.makeBrightPalettes(value & 0x3f)
	case 3:
		s.Palette.makeDarkPalettes(value & 0x3f)
	}
}

func (s *SoftwareRenderer) setBlendEnabled(layer int, enabled bool, override BlendEffect) {
	s.alphaEnabled = enabled && override == ALPHA_BLEND
	if enabled {
		switch override {
		case 0, 1:
			s.Palette.makeNormalPalette(layer)
		case 2, 3:
			s.Palette.makeSpecialPalette(layer)
		}
	} else {
		s.Palette.makeNormalPalette(layer)
	}
}

// BLDALPHA
func (s *SoftwareRenderer) writeBlendAlpha(value uint16) {
	s.blendA = float64(value&0x001f) / 16
	if s.blendA > 1 {
		s.blendA = 1
	}

	s.blendB = float64((value&0x1f00)>>8) / 16
	if s.blendB > 1 {
		s.blendB = 1
	}
}

// BLDY
func (s *SoftwareRenderer) writeBlendY(value uint16) {
	s.blendY = float64(value)

	y := float64(value)
	if y >= 16 {
		y = 1
	} else {
		y /= 16
	}
	s.Palette.setBlendY(y)
}

func (s *SoftwareRenderer) writeMosaic(value uint16) {
	s.bgMosaicX = (value & 0xf) + 1
	s.bgMosaicY = ((value >> 4) & 0xf) + 1
	s.objMosaicX = ((value >> 8) & 0xf) + 1
	s.objMosaicY = ((value >> 12) & 0xf) + 1
}

func (s *SoftwareRenderer) resetLayers() {
	if s.bgMode > 1 {
		s.bg[0].enabled = false
		s.bg[1].enabled = false
	}

	if s.bg[2].enabled {
		s.bg[2].drawScanlineFunc = s.bgModes[s.bgMode]
	}

	if s.bgMode == 0 || s.bgMode == 2 {
		if s.bg[3].enabled {
			s.bg[3].drawScanlineFunc = s.bgModes[s.bgMode]
		}
	} else {
		s.bg[3].enabled = false
	}

	sort.Sort(&s.drawLayers)
}

// accessMapMode0 fetch tile info from BG map
func (s *SoftwareRenderer) accessMapMode0(base, size, x, yBase uint32, out *SharedMap) {
	offset := base + ((x / 4) & 0x3e) + yBase // bg map offset in VRAM

	// += 2KB * n (2KB is a single BG Map size)
	if size&1 == 1 {
		offset += (x & 0x100) * 8
	}

	mem := s.VRAM.Load16(offset)
	out.tile = mem & 1023
	out.hflip = util.Bit(mem, 10)
	out.vflip = util.Bit(mem, 11)
	out.palette = (mem & 0xf000) >> 8 // This is shifted up 4 to make pushPixel faster
}

func (s *SoftwareRenderer) accessMapMode1(base, size, x, yBase uint32, out *SharedMap) {
	offset := base + x*8 + yBase
	out.tile = uint16(s.VRAM.LoadU8(offset))
}

// fetch tile data (by 4bytes)
//
// 4bytes = 8px(16/16) or 4px(256/1)
//
// tileIdx is 0-1023 (in 256-color, 0-2046)
//
// y is 0-7 (in 256-color, 0-14)
func (s *SoftwareRenderer) accessTile(base, tileIdx, y uint32) uint32 {
	offset := base + tileIdx*32
	offset |= y * 4

	return s.VRAM.LoadU32(offset)
}

func (s *SoftwareRenderer) drawScanlineBlank(backing *Backing) {
	for x := 0; x < HORIZONTAL_PIXELS; x++ {
		backing.color[x] = 0xffff
		backing.stencil[x] = 0
	}
}

func (s *SoftwareRenderer) prepareScanline(backing *Backing) {
	stencil := s.target2[LAYER_BACKDROP]
	for x := 0; x < HORIZONTAL_PIXELS; x++ {
		backing.stencil[x] = stencil
	}
}

func applyVflip(y uint32, vflip bool) uint32 {
	if vflip {
		return (7 - y) & 0x7
	}
	return y & 0x7
}

func drawScanlineBGMode0(backing *Backing, bg *BGLayer, start, end uint32) {
	s := bg.video
	y := uint32(s.Vcount)

	// backing offset (0-256). Used in pushPixel
	offset := start

	// 0-511
	scx, scy := uint32(bg.x), bg.y

	localY := y + uint32(scy)
	if bg.mosaic {
		localY -= y % uint32(s.bgMosaicY)
	}
	localYLo := localY & 0b0111
	screenBase, charBase := bg.screenBase, bg.charBase
	size := bg.size
	index := bg.index
	sharedMap := &s.sharedMap

	paletteShift := 0
	if bg.color256 > 0 {
		paletteShift = 1
	}

	mask := s.target2[index] | (byte(bg.priority) << 1) | BACKGROUND_MASK
	if s.blendMode == ALPHA_BLEND && s.alphaEnabled {
		mask |= s.target1[index]
	}

	yBase := (localY * 8) & 0b0111_1100_0000
	switch size {
	case 2:
		yBase += (localY * 8) & 0b1000_0000_0000
	case 3:
		yBase += (localY * 16) & 0b0001_0000_0000_0000
	}

	// if x length in BG is 512, use 511 as Mask
	xMask := uint32(0xff)
	if size&1 == 1 {
		xMask = 0x1ff
	}

	s.accessMapMode0(screenBase, size, (start+scx)&xMask, yBase, sharedMap)
	tileRow := s.accessTile(
		charBase,
		uint32(sharedMap.tile)<<paletteShift,
		applyVflip(localYLo, sharedMap.vflip)<<paletteShift,
	)

	for x := start; x < end; x++ {
		localX := (x + scx) & xMask // x coordinate (start ~ end)

		mosaicX := uint32(0)
		if bg.mosaic {
			mosaicX = offset % uint32(s.bgMosaicX)
		}
		localX -= mosaicX

		// x offset in tile
		// if this value is zero, target tile is changed
		localXLo := localX & 0x7

		if bg.color256 == 0 {
			// 16 color (1px == 4bit == 0.5byte)
			// Process 8 pixels.

			if localXLo == 0 || (bg.mosaic && (mosaicX == 0)) {
				s.accessMapMode0(screenBase, size, localX, yBase, sharedMap)
				tileRow = s.accessTile(charBase, uint32(sharedMap.tile), applyVflip(localYLo, sharedMap.vflip))
				if tileRow == 0 && localXLo == 0 {
					x += 7
					offset += 8
					continue
				}
			}
		} else {
			// 256 color (1px == 8bit == 1byte)
			// Process 4 pixels on each loop.

			if localXLo == 0 || (bg.mosaic && (mosaicX == 0)) {
				// target is new tile, update map property
				s.accessMapMode0(screenBase, size, localX, yBase, sharedMap)
			}

			// localXLo&0b11 == 0 -> 0 or 4
			if (localXLo&0b11 == 0) || (bg.mosaic && (mosaicX == 0)) {
				tmp := uint32(0)
				if localX&0x4 > 0 == !sharedMap.hflip {
					tmp = 4
				}

				// load 4px(32bit)
				tileRow = s.accessTile(
					charBase+tmp,
					uint32(sharedMap.tile)<<1,
					applyVflip(localYLo, sharedMap.vflip)<<1,
				)

				if tileRow == 0 && (localXLo&0x3) == 0 {
					// if tileRow(4px) is transparent, skip 4 pixel
					x += 3
					offset += 4
					continue
				}
			}
		}

		if sharedMap.hflip {
			localXLo = 7 - localXLo
		}
		bg.pushPixel(index, *sharedMap, s, tileRow, localXLo, offset, backing, mask, true)
		offset++
	}
}

func drawScanlineBGMode2(backing *Backing, bg *BGLayer, start, end uint32) {
	s := bg.video
	y := int(s.Vcount)
	offset := start
	screenBase, charBase := bg.screenBase, bg.charBase
	size := bg.size
	sizeAdjusted := 128 << size
	index := bg.index
	sharedMap := &s.sharedMap

	mask := s.target2[index] | byte(bg.priority)<<1 | BACKGROUND_MASK
	if s.blendMode == ALPHA_BLEND && s.alphaEnabled {
		mask |= s.target1[index]
	}

	yBase, color := uint32(0), byte(0)
	for x := int(start); x < int(end); x++ {
		localXF64 := bg.dx*float64(x) + bg.sx
		localYF64 := bg.dy*float64(x) + bg.sy
		if bg.mosaic {
			localXF64 -= float64(x%int(s.bgMosaicX))*bg.dx + float64(y%int(s.bgMosaicY))*bg.dmx
			localYF64 -= float64(x%int(s.bgMosaicX))*bg.dy + float64(y%int(s.bgMosaicY))*bg.dmy
		}
		localX, localY := int(localXF64), int(localYF64)

		if bg.overflow {
			localX &= sizeAdjusted - 1
			if localX < 0 {
				localX += sizeAdjusted
			}
			localY &= sizeAdjusted - 1
			if localY < 0 {
				localY += sizeAdjusted
			}
		} else if localX < 0 || localY < 0 || localX >= sizeAdjusted || localY >= sizeAdjusted {
			offset++
			continue
		}

		yBase = uint32((localY<<1)&0b0111_1111_0000) << size
		s.accessMapMode1(screenBase, size, uint32(localX), yBase, sharedMap)
		color = s.VRAM.LoadU8(charBase + (uint32(sharedMap.tile) * 64) + (uint32(localY&0b0111) * 8) + uint32(localX&0x7))
		bg.pushPixel(index, *sharedMap, s, uint32(color), 0, offset, backing, mask, true)
		offset++
	}
}

func drawScanlineBGMode3(backing *Backing, bg *BGLayer, start, end uint32) {
	s := bg.video
	y := int(s.Vcount)
	offset := start
	index := bg.index
	sharedMap := s.sharedMap

	mask := s.target2[index] | byte(bg.priority)<<1 | BACKGROUND_MASK
	if s.blendMode == ALPHA_BLEND && s.alphaEnabled {
		mask |= s.target1[index]
	}

	color := uint16(0)
	for x := int(start); x < int(end); x++ {
		localX := int(bg.dx*float64(x) + bg.sx)
		localY := int(bg.dy*float64(x) + bg.sy)
		if bg.mosaic {
			localX -= int(float64(x%int(s.bgMosaicX))*bg.dx + float64(y%int(s.bgMosaicY))*bg.dmx)
			localY -= int(float64(x%int(s.bgMosaicX))*bg.dy + float64(y%int(s.bgMosaicY))*bg.dmy)
		}

		if localX < 0 || localY < 0 || localX >= HORIZONTAL_PIXELS || localY >= VERTICAL_PIXELS {
			offset++
			continue
		}

		color = s.VRAM.Load16(uint32(localY*HORIZONTAL_PIXELS+localX) << 1)
		bg.pushPixel(index, sharedMap, s, uint32(color), 0, offset, backing, mask, false)
		offset++
	}
}

func drawScanlineBGMode4(backing *Backing, bg *BGLayer, start, end uint32) {
	s := bg.video
	y := int(s.Vcount)
	offset := start
	charBase := uint32(0)
	if s.displayFrameSelect {
		charBase = 0xa000
	}
	index := bg.index
	sharedMap := s.sharedMap

	mask := s.target2[index] | byte(bg.priority)<<1 | BACKGROUND_MASK
	if s.blendMode == ALPHA_BLEND && s.alphaEnabled {
		mask |= s.target1[index]
	}

	for x := int(start); x < int(end); x++ {
		localX := int(bg.dx*float64(x) + bg.sx)
		localY := int(bg.dy*float64(x) + bg.sy)
		if bg.mosaic {
			localX -= int(float64(x%int(s.bgMosaicX))*bg.dx + float64(y%int(s.bgMosaicY))*bg.dmx)
			localY -= int(float64(x%int(s.bgMosaicX))*bg.dy + float64(y%int(s.bgMosaicY))*bg.dmy)
		}

		if localX < 0 || localY < 0 || localX >= HORIZONTAL_PIXELS || localY >= VERTICAL_PIXELS {
			offset++
			continue
		}

		color := s.VRAM.LoadU8(charBase + uint32(localY*HORIZONTAL_PIXELS+localX))
		bg.pushPixel(index, sharedMap, s, uint32(color), 0, offset, backing, mask, false) // TODO raw is false?
		offset++
	}
}

func drawScanlineBGMode5(backing *Backing, bg *BGLayer, start, end uint32) {
	s := bg.video
	y := int(s.Vcount)

	offset := start
	charBase := uint32(0)
	if s.displayFrameSelect {
		charBase = 0xa000
	}
	index := bg.index
	sharedMap := s.sharedMap

	mask := s.target2[index] | byte(bg.priority)<<1 | BACKGROUND_MASK
	if s.blendMode == ALPHA_BLEND && s.alphaEnabled {
		mask |= s.target1[index]
	}

	for x := int(start); x < int(end); x++ {
		localX := int(bg.dx*float64(x) + bg.sx)
		localY := int(bg.dy*float64(x) + bg.sy)
		if bg.mosaic {
			localX -= int(float64(x%int(s.bgMosaicX))*bg.dx + float64(y%int(s.bgMosaicY))*bg.dmx)
			localY -= int(float64(x%int(s.bgMosaicX))*bg.dy + float64(y%int(s.bgMosaicY))*bg.dmy)
		}

		if localX < 0 || localY < 0 || localX >= 160 || localY >= 128 {
			offset++
			continue
		}

		color := s.VRAM.Load16((charBase + uint32(localY*160+localX)) << 1)
		bg.pushPixel(index, sharedMap, s, uint32(color), 0, offset, backing, mask, false)
		offset++
	}
}

// DrawScanline is root function
// This calls drawScanline internally
func (s *SoftwareRenderer) DrawScanline(y uint16) {
	backing := &s.scanline
	if s.forcedBlank {
		s.drawScanlineBlank(backing)
		return
	}

	s.prepareScanline(backing)
	s.Vcount = y

	for i := 0; i < len(s.drawLayers); i++ {
		layer := s.drawLayers[i]
		idx := layer.Index()
		if !layer.Enabled() {
			continue
		}

		s.objwinActive = false
		if !s.win0 && !s.win1 && !s.objwin {
			// no window
			s.setBlendEnabled(idx, s.target1[idx] > 0, s.blendMode)
			layer.drawScanline(backing, 0, HORIZONTAL_PIXELS)
		} else {
			// use window
			firstStart, firstEnd := uint16(0), uint16(HORIZONTAL_PIXELS)
			lastStart, lastEnd := uint16(0), uint16(HORIZONTAL_PIXELS)

			if s.win0 && (y >= s.win0Top && y < s.win0Bottom) {
				// inner window0
				if s.windows[0].enabled[idx] {
					s.setBlendEnabled(idx, s.windows[0].special && s.target1[idx] > 0, s.blendMode)
					layer.drawScanline(backing, uint32(s.win0Left), uint32(s.win0Right))
				}

				firstStart = uint16(math.Max(float64(firstStart), float64(s.win0Left)))
				firstEnd = uint16(math.Min(float64(firstEnd), float64(s.win0Left)))
				lastStart = uint16(math.Max(float64(lastStart), float64(s.win0Right)))
				lastEnd = uint16(math.Min(float64(lastEnd), float64(s.win0Right)))
			}

			if s.win1 && (y >= s.win1Top && y < s.win1Bottom) {
				// inner window1
				if s.windows[1].enabled[idx] {
					s.setBlendEnabled(idx, s.windows[1].special && s.target1[idx] > 0, s.blendMode)

					if !s.windows[0].enabled[idx] && (s.win1Left < firstStart || s.win1Right < lastStart) {
						// We've been cut in two by window 0!
						layer.drawScanline(backing, uint32(s.win1Left), uint32(firstStart))
						layer.drawScanline(backing, uint32(lastEnd), uint32(s.win1Right))
					} else {
						layer.drawScanline(backing, uint32(s.win1Left), uint32(s.win1Right))
					}
				}

				firstStart = uint16(math.Max(float64(firstStart), float64(s.win1Left)))
				firstEnd = uint16(math.Min(float64(firstEnd), float64(s.win1Left)))
				lastStart = uint16(math.Max(float64(lastStart), float64(s.win1Right)))
				lastEnd = uint16(math.Min(float64(lastEnd), float64(s.win1Right)))
			}

			// Do last two
			if s.windows[2].enabled[idx] || (s.objwin && s.windows[3].enabled[idx]) {
				// WINOUT/OBJWIN
				s.objwinActive = s.objwin
				s.setBlendEnabled(idx, s.windows[2].special && s.target1[idx] > 0, s.blendMode) // Window 3 handled in pushPixel

				if firstEnd > lastStart {
					layer.drawScanline(backing, 0, HORIZONTAL_PIXELS)
				} else {
					if firstEnd != 0 {
						layer.drawScanline(backing, 0, uint32(firstEnd))
					}
					if lastStart < HORIZONTAL_PIXELS {
						layer.drawScanline(backing, uint32(lastStart), HORIZONTAL_PIXELS)
					}
					if lastEnd < firstStart {
						layer.drawScanline(backing, uint32(lastEnd), uint32(firstStart))
					}
				}
			}

			s.setBlendEnabled(LAYER_BACKDROP, (s.target1[LAYER_BACKDROP] > 0 && s.windows[2].special), s.blendMode)
		}

		if l, ok := layer.(*BGLayer); ok {
			if l.bg {
				l.sx += l.dmx
				l.sy += l.dmy
			}
		}
	}

	s.finishScanline(backing)
}

// Push backing data into Screen buffer
func (s *SoftwareRenderer) finishScanline(backing *Backing) {
	bd := s.Palette.accessColor(LAYER_BACKDROP, 0)
	xx := uint32(s.Vcount) * HORIZONTAL_PIXELS * 4
	isTarget2 := s.target2[LAYER_BACKDROP] > 0

	for x := 0; x < HORIZONTAL_PIXELS; x++ {
		sharedColor := [3]byte{}
		if (backing.stencil[x] & WRITTEN_MASK) > 0 {
			color := backing.color[x]
			if isTarget2 && (backing.stencil[x]&TARGET1_MASK) > 0 {
				color = s.Palette.mix(s.blendA, color, s.blendB, bd)
			}
			sharedColor = s.Palette.convert16To32(color)
		} else {
			sharedColor = s.Palette.convert16To32(bd)
		}

		s.pixelData[xx] = sharedColor[0]
		s.pixelData[xx+1] = sharedColor[1]
		s.pixelData[xx+2] = sharedColor[2]
		xx += 4
	}
}

func (s *SoftwareRenderer) FinishDraw() ImageData {
	s.bg[2].sx = s.bg[2].refx
	s.bg[2].sy = s.bg[2].refy
	s.bg[3].sx = s.bg[3].refx
	s.bg[3].sy = s.bg[3].refy
	return s.pixelData
}
