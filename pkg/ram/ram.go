package ram

import (
	"encoding/binary"
)

const (
	_       = iota
	kb uint = 1 << (10 * iota)
	mb
	gb
)

// RAM struct
type RAM struct {
	BIOS  [16 * kb]byte
	EWRAM [256 * kb]byte
	IWRAM [32 * kb]byte
	IO    [1 * kb]byte
	VideoRAM
	GamePak
}

type VideoRAM struct {
	Palette [kb]byte
	VRAM    [96 * kb]byte
	OAM     [kb]byte
}

type GamePak struct {
	GamePak0 [32 * mb]byte
	GamePak1 [32 * mb]byte
	GamePak2 [32 * mb]byte
	SRAM     [64 * kb]byte
}

func (r *RAM) Get(addr uint32) uint32 {
	switch {
	case BIOS(addr):
		offset := BIOSOffset(addr)
		return binary.LittleEndian.Uint32(r.BIOS[offset : offset+3])
	case EWRAM(addr):
		offset := EWRAMOffset(addr)
		return binary.LittleEndian.Uint32(r.EWRAM[offset : offset+3])
	case IWRAM(addr):
		offset := IWRAMOffset(addr)
		return binary.LittleEndian.Uint32(r.IWRAM[offset : offset+3])
	case IO(addr):
		offset := IOOffset(addr)
		return binary.LittleEndian.Uint32(r.IO[offset : offset+3])
	case Palette(addr):
		offset := PaletteOffset(addr)
		return binary.LittleEndian.Uint32(r.Palette[offset : offset+3])
	case VRAM(addr):
		offset := VRAMOffset(addr)
		return binary.LittleEndian.Uint32(r.VRAM[offset : offset+3])
	case OAM(addr):
		offset := OAMOffset(addr)
		return binary.LittleEndian.Uint32(r.OAM[offset : offset+3])
	case GamePak0(addr):
		offset := GamePak0Offset(addr)
		return binary.LittleEndian.Uint32(r.GamePak0[offset : offset+3])
	case GamePak1(addr):
		offset := GamePak1Offset(addr)
		return binary.LittleEndian.Uint32(r.GamePak1[offset : offset+3])
	case GamePak2(addr):
		offset := GamePak2Offset(addr)
		return binary.LittleEndian.Uint32(r.GamePak2[offset : offset+3])
	case SRAM(addr):
		offset := SRAMOffset(addr)
		return binary.LittleEndian.Uint32(r.SRAM[offset : offset+3])
	}
	return 0
}

// Set8 sets byte into addr
func (r *RAM) Set8(addr uint32, b byte) {
	switch {
	case BIOS(addr):
		r.BIOS[BIOSOffset(addr)] = b
	case EWRAM(addr):
		r.EWRAM[EWRAMOffset(addr)] = b
	case IWRAM(addr):
		r.IWRAM[IWRAMOffset(addr)] = b
	case IO(addr):
		r.IO[IOOffset(addr)] = b
	case Palette(addr):
		r.Palette[PaletteOffset(addr)] = b
	case VRAM(addr):
		r.VRAM[VRAMOffset(addr)] = b
	case OAM(addr):
		r.OAM[OAMOffset(addr)] = b
	case GamePak0(addr):
		r.GamePak0[GamePak0Offset(addr)] = b
	case GamePak1(addr):
		r.GamePak1[GamePak1Offset(addr)] = b
	case GamePak2(addr):
		r.GamePak2[GamePak2Offset(addr)] = b
	case SRAM(addr):
		r.SRAM[SRAMOffset(addr)] = b
	}
}

// Set16 sets half-word into addr
func (r *RAM) Set16(addr uint32, value uint16) {
	b0, b1 := value&0xff, (value>>8)&0xff
	r.Set8(addr, byte(b0))
	r.Set8(addr+1, byte(b1))
}

// Set32 sets word into addr
func (r *RAM) Set32(addr uint32, value uint32) {
	b0, b1, b2, b3 := value&0xff, (value>>8)&0xff, (value>>16)&0xff, (value>>24)&0xff
	r.Set8(addr, byte(b0))
	r.Set8(addr+1, byte(b1))
	r.Set8(addr+2, byte(b2))
	r.Set8(addr+3, byte(b3))
}
