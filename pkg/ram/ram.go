package ram

import (
	_ "embed"
	"magia/pkg/util"
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
	GamePak0, GamePak1, GamePak2 [32 * mb]byte
	SRAM                         [64 * kb]byte
}

func (r *RAM) Get(addr uint32) uint32 {
	switch {
	case BIOS(addr):
		offset := BIOSOffset(addr)
		if offset > 0x3FFF {
			return 0
		}
		return util.LE32(r.BIOS[offset:])
	case EWRAM(addr):
		offset := EWRAMOffset(addr)
		return util.LE32(r.EWRAM[offset:])
	case IWRAM(addr):
		offset := IWRAMOffset(addr)
		return util.LE32(r.IWRAM[offset:])
	case IO(addr):
		offset := IOOffset(addr)
		if offset > 0x3fe {
			return 0
		}
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
	case BIOS(addr): // write only
		return
	case EWRAM(addr):
		r.EWRAM[EWRAMOffset(addr)] = b
	case IWRAM(addr):
		r.IWRAM[IWRAMOffset(addr)] = b
	case IO(addr):
		offset := IOOffset(addr)
		if offset > 0x3fe {
			return
		}
		r.IO[offset] = b
	case GamePak0(addr):
		return
	case GamePak1(addr):
		return
	case GamePak2(addr):
		return
	case SRAM(addr):
		r.SRAM[SRAMOffset(addr)] = b
	}
}

var busWidthMap = map[uint32]int{0x0: 32, 0x3: 32, 0x4: 32, 0x7: 32, 0x2: 16, 0x5: 16, 0x6: 16, 0x8: 16, 0x9: 16, 0xa: 16, 0xb: 16, 0xc: 16, 0xd: 16, 0xe: 8, 0xf: 8}

func BusWidth(addr uint32) int { return busWidthMap[addr>>24] }
