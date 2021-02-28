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
	if ok, offset := BIOS(addr); ok {
		return binary.LittleEndian.Uint32(r.BIOS[offset : offset+3])
	}
	if ok, offset := EWRAM(addr); ok {
		return binary.LittleEndian.Uint32(r.EWRAM[offset : offset+3])
	}
	if ok, offset := IWRAM(addr); ok {
		return binary.LittleEndian.Uint32(r.IWRAM[offset : offset+3])
	}
	if ok, offset := IO(addr); ok {
		return binary.LittleEndian.Uint32(r.IO[offset : offset+3])
	}
	if ok, offset := Palette(addr); ok {
		return binary.LittleEndian.Uint32(r.Palette[offset : offset+3])
	}
	if ok, offset := VRAM(addr); ok {
		return binary.LittleEndian.Uint32(r.VRAM[offset : offset+3])
	}
	if ok, offset := OAM(addr); ok {
		return binary.LittleEndian.Uint32(r.OAM[offset : offset+3])
	}
	if ok, offset := GamePak0(addr); ok {
		return binary.LittleEndian.Uint32(r.GamePak0[offset : offset+3])
	}
	if ok, offset := GamePak1(addr); ok {
		return binary.LittleEndian.Uint32(r.GamePak1[offset : offset+3])
	}
	if ok, offset := GamePak2(addr); ok {
		return binary.LittleEndian.Uint32(r.GamePak2[offset : offset+3])
	}
	if ok, offset := SRAM(addr); ok {
		return binary.LittleEndian.Uint32(r.SRAM[offset : offset+3])
	}
	return 0
}

// Set8 sets byte into addr
func (r *RAM) Set8(addr uint32, b byte) {
	if ok, offset := BIOS(addr); ok {
		r.BIOS[offset] = b
		return
	}
	if ok, offset := EWRAM(addr); ok {
		r.EWRAM[offset] = b
		return
	}
	if ok, offset := IWRAM(addr); ok {
		r.IWRAM[offset] = b
		return
	}
	if ok, offset := IO(addr); ok {
		r.IO[offset] = b
		return
	}
	if ok, offset := Palette(addr); ok {
		r.Palette[offset] = b
		return
	}
	if ok, offset := VRAM(addr); ok {
		r.VRAM[offset] = b
		return
	}
	if ok, offset := OAM(addr); ok {
		r.OAM[offset] = b
		return
	}
	if ok, offset := GamePak0(addr); ok {
		r.GamePak0[offset] = b
		return
	}
	if ok, offset := GamePak1(addr); ok {
		r.GamePak1[offset] = b
		return
	}
	if ok, offset := GamePak2(addr); ok {
		r.GamePak2[offset] = b
		return
	}
	if ok, offset := SRAM(addr); ok {
		r.SRAM[offset] = b
		return
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
