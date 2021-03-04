package gba

import (
	"mettaur/pkg/cart"
	"mettaur/pkg/ram"
)

// GBA is core object
type GBA struct {
	Reg
	CartHeader *cart.Header
	RAM        ram.RAM
	lastAddr   uint32
	cycle      uint
	PC         uint32
}

// New GBA
func New(src []byte) *GBA {
	return &GBA{
		CartHeader: cart.New(src),
		RAM:        ram.RAM{},
	}
}
