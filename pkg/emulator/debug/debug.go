package debug

import "github.com/pokemium/magia/pkg/gba"

type Debugger struct {
	g     *gba.GBA
	pause *bool
}

func New(g *gba.GBA, pause *bool) *Debugger {
	return &Debugger{
		g:     g,
		pause: pause,
	}
}
