package joypad

import (
	"github.com/pokemium/magia/pkg/util"

	"github.com/hajimehoshi/ebiten/v2"
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
	Input [4]byte
}

func (j *Joypad) Read() {
	j.Input[0] = util.SetBit8(j.Input[0], A, !btnA())
	j.Input[0] = util.SetBit8(j.Input[0], B, !btnB())
	j.Input[0] = util.SetBit8(j.Input[0], Select, !btnSelect())
	j.Input[0] = util.SetBit8(j.Input[0], Start, !btnStart())
	if btnRight() {
		j.Input[0] = j.Input[0] & ^byte((1 << Right))
		j.Input[0] = j.Input[0] | byte((1 << Left)) // off <-
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Right))
	}
	if btnLeft() {
		j.Input[0] = j.Input[0] & ^byte((1 << Left))
		j.Input[0] = j.Input[0] | byte((1 << Right)) // off ->
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Left))
	}
	if btnUp() {
		j.Input[0] = j.Input[0] & ^byte((1 << Up))
		j.Input[0] = j.Input[0] | byte((1 << Down)) // off ↓
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Up))
	}
	if btnDown() {
		j.Input[0] = j.Input[0] & ^byte((1 << Down))
		j.Input[0] = j.Input[0] | byte((1 << Up)) // off ↑
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Down))
	}
	j.Input[1] = util.SetBit8(j.Input[1], R, !btnR())
	j.Input[1] = util.SetBit8(j.Input[1], L, !btnL())
}

func btnA() bool      { return ebiten.IsKeyPressed(ebiten.KeyX) }
func btnB() bool      { return ebiten.IsKeyPressed(ebiten.KeyZ) }
func btnSelect() bool { return ebiten.IsKeyPressed(ebiten.KeyBackspace) }
func btnStart() bool  { return ebiten.IsKeyPressed(ebiten.KeyEnter) }
func btnRight() bool  { return ebiten.IsKeyPressed(ebiten.KeyRight) }
func btnLeft() bool   { return ebiten.IsKeyPressed(ebiten.KeyLeft) }
func btnUp() bool     { return ebiten.IsKeyPressed(ebiten.KeyUp) }
func btnDown() bool   { return ebiten.IsKeyPressed(ebiten.KeyDown) }
func btnR() bool      { return ebiten.IsKeyPressed(ebiten.KeyS) }
func btnL() bool      { return ebiten.IsKeyPressed(ebiten.KeyA) }

func Debug() bool { return ebiten.IsKeyPressed(ebiten.Key5) }
