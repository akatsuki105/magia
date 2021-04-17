package ram

import (
	_ "embed"
	"mettaur/pkg/util"
)

const (
	_       = iota
	kb uint = 1 << (10 * iota)
	mb
	gb
)

//go:embed bios.gba
var sBIOS []byte

// RAM struct
type RAM struct {
	BIOS  [16 * kb]byte
	EWRAM [256 * kb]byte
	IWRAM [32 * kb]byte
	IO    [2 * kb]byte
	GamePak
	ROMSize int
}

func New(src []byte) *RAM {
	bios := [16 * kb]byte{}
	for i, b := range sBIOS {
		bios[i] = b
	}

	gamePak0 := [32 * mb]byte{}
	for i, b := range src {
		gamePak0[i] = b
	}

	return &RAM{
		BIOS: bios,
		GamePak: GamePak{
			GamePak0: gamePak0,
		},
		ROMSize: len(src),
	}
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
		return util.LE32(r.BIOS[offset:])
	case EWRAM(addr):
		offset := EWRAMOffset(addr)
		return util.LE32(r.EWRAM[offset:])
	case IWRAM(addr):
		offset := IWRAMOffset(addr)
		return util.LE32(r.IWRAM[offset:])
	case IO(addr):
		offset := IOOffset(addr)
		return util.LE32(r.IO[offset:])
	case GamePak0(addr):
		offset := GamePak0Offset(addr)
		return util.LE32(r.GamePak0[offset:])
	case GamePak1(addr):
		offset := GamePak1Offset(addr)
		return util.LE32(r.GamePak0[offset:])
	case GamePak2(addr):
		offset := GamePak2Offset(addr)
		return util.LE32(r.GamePak0[offset:])
	case SRAM(addr):
		offset := SRAMOffset(addr)
		return util.LE32(r.SRAM[offset:])
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
