package joypad

import (
	"github.com/hajimehoshi/ebiten/v2"
)

var Handler = [10](func() bool){
	btnA, btnB, btnSelect, btnStart, btnRight, btnLeft, btnUp, btnDown, btnR, btnL,
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
