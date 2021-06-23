package video

import "fmt"

type VRAM struct {
	*MemoryAligned16
	vram []uint16
}

func NewVRAM(size uint32) *VRAM {
	mem := NewMemoryAligned16(size)
	return &VRAM{
		MemoryAligned16: mem,
		vram:            mem.buffer,
	}
}

func (v *VRAM) String() string {
	return fmt.Sprintf("%+v", v.vram)
}
