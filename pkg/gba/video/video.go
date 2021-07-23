package video

import (
	"github.com/pokemium/magia/pkg/gba/ram"
	"github.com/pokemium/magia/pkg/util"
)

const (
	HDRAW_LENGTH = 960
	HBLANK_FLIP  = 46
)

const DISPSTAT_MASK = 0xff38

func (v *Video) Load32(addr uint32) uint32 {
	switch addr {
	case ram.VCOUNT:
		return uint32(v.RenderPath.Vcount)
	}
	return util.LE32(v.IO[ram.IOOffset(addr):])
}

func (v *Video) Set8(addr uint32, val byte) {
	switch addr {
	case ram.WININ, ram.WININ + 1, ram.WINOUT, ram.WINOUT + 1:
		val &= 0x3f
	}

	ofs := ram.IOOffset(addr)
	val16 := uint16(0)
	if ofs&1 == 1 {
		val16 = uint16(val) << 8
		val16 |= uint16(v.IO[ofs-1])
	} else {
		val16 = uint16(val)
		val16 |= uint16(v.IO[ofs+1]) << 8
	}
	v.Set16(addr&0xffff_fffe, val16)
}

func (v *Video) Set16(addr uint32, val uint16) {
	switch addr {
	case ram.DISPCNT:
		v.RenderPath.writeDisplayControl(val)
	case ram.DISPSTAT:
		val &= DISPSTAT_MASK
	case ram.BG0CNT:
		val &= 0xdfff
		v.RenderPath.writeBackgroundControl(0, val)
	case ram.BG1CNT:
		val &= 0xdfff
		v.RenderPath.writeBackgroundControl(1, val)
	case ram.BG2CNT:
		v.RenderPath.writeBackgroundControl(2, val)
	case ram.BG3CNT:
		v.RenderPath.writeBackgroundControl(3, val)
	case ram.BG0HOFS:
		v.RenderPath.writeBackgroundHOffset(0, val)
	case ram.BG0VOFS:
		v.RenderPath.writeBackgroundVOffset(0, val)
	case ram.BG1HOFS:
		v.RenderPath.writeBackgroundHOffset(1, val)
	case ram.BG1VOFS:
		v.RenderPath.writeBackgroundVOffset(1, val)
	case ram.BG2HOFS:
		v.RenderPath.writeBackgroundHOffset(2, val)
	case ram.BG2VOFS:
		v.RenderPath.writeBackgroundVOffset(2, val)
	case ram.BG3HOFS:
		v.RenderPath.writeBackgroundHOffset(3, val)
	case ram.BG3VOFS:
		v.RenderPath.writeBackgroundVOffset(3, val)
	case ram.BG2X:
		upper := util.LE16(v.IO[ram.IOOffset(addr)+2:])
		v.RenderPath.writeBackgroundRefX(2, uint32(upper)<<16|uint32(val))
	case ram.BG2X + 2:
		lower := util.LE16(v.IO[ram.IOOffset(addr)-2:])
		v.RenderPath.writeBackgroundRefX(2, uint32(val)<<16|uint32(lower))
	case ram.BG2Y:
		upper := util.LE16(v.IO[ram.IOOffset(addr)+2:])
		v.RenderPath.writeBackgroundRefY(2, uint32(upper)<<16|uint32(val))
	case ram.BG2Y + 2:
		lower := util.LE16(v.IO[ram.IOOffset(addr)-2:])
		v.RenderPath.writeBackgroundRefY(2, uint32(val)<<16|uint32(lower))
	case ram.BG2PA:
		v.RenderPath.writeBackgroundParamA(2, val)
	case ram.BG2PB:
		v.RenderPath.writeBackgroundParamB(2, val)
	case ram.BG2PC:
		v.RenderPath.writeBackgroundParamC(2, val)
	case ram.BG2PD:
		v.RenderPath.writeBackgroundParamD(2, val)
	case ram.BG3X:
		upper := util.LE16(v.IO[ram.IOOffset(addr)+2:])
		v.RenderPath.writeBackgroundRefX(3, uint32(upper)<<16|uint32(val))
	case ram.BG3X + 2:
		lower := util.LE16(v.IO[ram.IOOffset(addr)-2:])
		v.RenderPath.writeBackgroundRefX(3, uint32(val)<<16|uint32(lower))
	case ram.BG3Y:
		upper := util.LE16(v.IO[ram.IOOffset(addr)+2:])
		v.RenderPath.writeBackgroundRefY(3, uint32(upper)<<16|uint32(val))
	case ram.BG3Y + 2:
		lower := util.LE16(v.IO[ram.IOOffset(addr)-2:])
		v.RenderPath.writeBackgroundRefY(3, uint32(val)<<16|uint32(lower))
	case ram.BG3PA:
		v.RenderPath.writeBackgroundParamA(3, val)
	case ram.BG3PB:
		v.RenderPath.writeBackgroundParamB(3, val)
	case ram.BG3PC:
		v.RenderPath.writeBackgroundParamC(3, val)
	case ram.BG3PD:
		v.RenderPath.writeBackgroundParamD(3, val)
	case ram.WIN0H:
		v.RenderPath.writeWin0H(val)
	case ram.WIN1H:
		v.RenderPath.writeWin1H(val)
	case ram.WIN0V:
		v.RenderPath.writeWin0V(val)
	case ram.WIN1V:
		v.RenderPath.writeWin1V(val)
	case ram.WININ:
		val &= 0x3f3f
		v.RenderPath.writeWinIn(val)
	case ram.WINOUT:
		val &= 0x3f3f
		v.RenderPath.writeWinOut(val)
	case ram.BLDCNT:
		val &= 0x7fff
		v.RenderPath.writeBlendControl(val)
	case ram.BLDALPHA:
		val &= 0x1f1f
		v.RenderPath.writeBlendAlpha(val)
	case ram.BLDY:
		val &= 0x001f
		v.RenderPath.writeBlendY(val)
	case ram.MOSAIC:
		v.RenderPath.writeMosaic(val)
	}

	ofs := ram.IOOffset(addr)
	v.IO[ofs] = byte(val)
	v.IO[ofs+1] = byte(val >> 8)
}

func (v *Video) Set32(addr uint32, val uint32) {
	switch addr {
	case ram.BG2X:
		val &= 0x0fff_ffff
		v.RenderPath.writeBackgroundRefX(2, val)
	case ram.BG2Y:
		val &= 0x0fff_ffff
		v.RenderPath.writeBackgroundRefY(2, val)
	case ram.BG3X:
		val &= 0x0fff_ffff
		v.RenderPath.writeBackgroundRefX(3, val)
	case ram.BG3Y:
		val &= 0x0fff_ffff
		v.RenderPath.writeBackgroundRefY(3, val)
	default:
		v.Set16(addr, uint16(val))
		v.Set16(addr+2, uint16(val>>16))
	}

	ofs := ram.IOOffset(addr)
	v.IO[ofs] = byte(val)
	v.IO[ofs+1] = byte(val >> 8)
	v.IO[ofs+2] = byte(val >> 16)
	v.IO[ofs+3] = byte(val >> 24)
}

type Video struct {
	IO         [96]byte
	RenderPath *SoftwareRenderer
}

func NewVideo() *Video {
	return &Video{
		RenderPath: NewSoftwareRenderer(),
	}
}

// VBlank returns true if in VBlank
func (v *Video) VBlank() bool {
	return util.Bit(uint16(v.IO[ram.IOOffset(ram.DISPSTAT)]), 0)
}

func (v *Video) SetVBlank(b bool) {
	if b {
		v.IO[ram.IOOffset(ram.DISPSTAT)] = v.IO[ram.IOOffset(ram.DISPSTAT)] | 0b0000_0001
		return
	}
	v.IO[ram.IOOffset(ram.DISPSTAT)] = v.IO[ram.IOOffset(ram.DISPSTAT)] & 0b1111_1110
}
func (v *Video) SetHBlank(b bool) {
	if b {
		v.IO[ram.IOOffset(ram.DISPSTAT)] = v.IO[ram.IOOffset(ram.DISPSTAT)] | 0b0000_0010
		return
	}
	v.IO[ram.IOOffset(ram.DISPSTAT)] = v.IO[ram.IOOffset(ram.DISPSTAT)] & 0b1111_1101
}
func (v *Video) SetVCounter(b bool) {
	if b {
		v.IO[ram.IOOffset(ram.DISPSTAT)] = v.IO[ram.IOOffset(ram.DISPSTAT)] | 0b0000_0100
		return
	}
	v.IO[ram.IOOffset(ram.DISPSTAT)] = v.IO[ram.IOOffset(ram.DISPSTAT)] & 0b1111_1011
}
