package ram

func BusWidth(addr uint32) int {
	switch {
	case BIOS(addr) || IWRAM(addr) || IO(addr) || OAM(addr):
		return 32
	case EWRAM(addr) || Palette(addr) || VRAM(addr) || GamePak0(addr) || GamePak1(addr) || GamePak2(addr):
		return 16
	case SRAM(addr):
		return 8
	}
	return 32
}
