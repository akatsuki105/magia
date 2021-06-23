package util

type Region uint32

const (
	BIOS    Region = 0x0
	EWRAM   Region = 0x2
	IWRAM   Region = 0x3
	IO      Region = 0x4
	PALETTE Region = 0x5
	VRAM    Region = 0x6
	OAM     Region = 0x7
	CART0   Region = 0x8
	CART1   Region = 0xa
	CART2   Region = 0xc
	SRAM    Region = 0xe
)

var RegionSize = map[string]uint32{
	"BIOS":     0x00004000,
	"EWRAM":    0x00040000,
	"IWRAM":    0x00008000,
	"IO":       0x00000400,
	"PALETTE":  0x00000400,
	"VRAM":     0x00018000,
	"OAM":      0x00000400,
	"CART0":    0x02000000,
	"CART1":    0x02000000,
	"CART2":    0x02000000,
	"SRAM":     0x00008000,
	"FLASH512": 0x00010000,
	"FLASH1M":  0x00020000,
	"EEPROM":   0x00002000,
}
