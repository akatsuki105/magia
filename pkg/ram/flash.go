package ram

func (r *RAM) FlashWrite(ofs uint32, b byte) {
	r.SRAM[ofs] = b
}
