package joypad

import (
	"github.com/pokemium/magia/pkg/util"
)

const (
	A      = 0
	B      = 1
	Select = 2
	Start  = 3
	Right  = 4
	Left   = 5
	Up     = 6
	Down   = 7
	R      = 0
	L      = 1
)

type Joypad struct {
	Input   [4]byte
	handler [10](func() bool)
}

func New(joypadHandlers [10]func() bool) *Joypad {
	return &Joypad{
		handler: joypadHandlers,
	}
}

func (j *Joypad) Read() {
	j.Input[0] = util.SetBit8(j.Input[0], A, !wrapHandler(&j.handler[A]))
	j.Input[0] = util.SetBit8(j.Input[0], B, !wrapHandler(&j.handler[B]))
	j.Input[0] = util.SetBit8(j.Input[0], Select, !wrapHandler(&j.handler[Select]))
	j.Input[0] = util.SetBit8(j.Input[0], Start, !wrapHandler(&j.handler[Start]))
	if wrapHandler(&j.handler[Right]) {
		j.Input[0] = j.Input[0] & ^byte((1 << Right))
		j.Input[0] = j.Input[0] | byte((1 << Left)) // off <-
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Right))
	}
	if wrapHandler(&j.handler[Left]) {
		j.Input[0] = j.Input[0] & ^byte((1 << Left))
		j.Input[0] = j.Input[0] | byte((1 << Right)) // off ->
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Left))
	}
	if wrapHandler(&j.handler[Up]) {
		j.Input[0] = j.Input[0] & ^byte((1 << Up))
		j.Input[0] = j.Input[0] | byte((1 << Down)) // off ↓
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Up))
	}
	if wrapHandler(&j.handler[Down]) {
		j.Input[0] = j.Input[0] & ^byte((1 << Down))
		j.Input[0] = j.Input[0] | byte((1 << Up)) // off ↑
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Down))
	}
	j.Input[1] = util.SetBit8(j.Input[1], R, !wrapHandler(&j.handler[R+8]))
	j.Input[1] = util.SetBit8(j.Input[1], L, !wrapHandler(&j.handler[L+8]))
}

func wrapHandler(h *func() bool) bool {
	if h == nil {
		return false
	}
	if *h == nil {
		return false
	}
	return (*h)()
}
