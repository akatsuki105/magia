package ram

func BusWidth(addr uint32) int {
	switch {
	case BIOS(addr):
		return 32
	case EWRAM(addr):
		return 16
	case IWRAM(addr):
		return 32
	case IO(addr):
		return 32
	case Palette(addr):
		return 16
	case VRAM(addr):
		return 16
	case OAM(addr):
		return 32
	case GamePak0(addr) || GamePak1(addr) || GamePak2(addr):
		return 16
	case SRAM(addr):
		return 8
	}
	return 32
}
