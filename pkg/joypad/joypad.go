package joypad

import "github.com/hajimehoshi/ebiten/v2"

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
	if btnA() {
		j.Input[0] = j.Input[0] & ^byte(1)
	} else {
		j.Input[0] = j.Input[0] | byte(1)
	}

	if btnB() {
		j.Input[0] = j.Input[0] & ^byte((1 << B))
	} else {
		j.Input[0] = j.Input[0] | byte((1 << B))
	}

	if btnSelect() {
		j.Input[0] = j.Input[0] & ^byte((1 << Select))
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Select))
	}

	if btnStart() {
		j.Input[0] = j.Input[0] & ^byte((1 << Start))
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Start))
	}
	if btnRight() {
		j.Input[0] = j.Input[0] & ^byte((1 << Right))
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Right))
	}
	if btnLeft() {
		j.Input[0] = j.Input[0] & ^byte((1 << Left))
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Left))
	}
	if btnUp() {
		j.Input[0] = j.Input[0] & ^byte((1 << Up))
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Up))
	}
	if btnDown() {
		j.Input[0] = j.Input[0] & ^byte((1 << Down))
	} else {
		j.Input[0] = j.Input[0] | byte((1 << Down))
	}
	if btnR() {
		j.Input[1] = j.Input[1] & ^byte((1 << (R)))
	} else {
		j.Input[1] = j.Input[1] | byte((1 << (R)))
	}
	if btnL() {
		j.Input[1] = j.Input[1] & ^byte((1 << (L)))
	} else {
		j.Input[1] = j.Input[1] | byte((1 << (L)))
	}
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
