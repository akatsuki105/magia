package cart

// Header represents GBA Cartridge header
type Header struct {
	Entry     [4]byte
	Title     string
	GameCode  string
	MakerCode string
}

// New cartridge header
func New(src []byte) *Header {
	return &Header{
		Entry:     [4]byte{src[0], src[1], src[2], src[3]},
		Title:     string(src[0xa0 : 0xa0+12]),
		GameCode:  string(src[0xac : 0xac+4]),
		MakerCode: string(src[0xb0 : 0xb0+2]),
	}
}
