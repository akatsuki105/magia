package emulator

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pokemium/magia/pkg/emulator/audio"
	"github.com/pokemium/magia/pkg/emulator/debug"
	"github.com/pokemium/magia/pkg/emulator/joypad"
	"github.com/pokemium/magia/pkg/gba"
)

var (
	second = time.NewTicker(time.Second)
	cache  []byte
)

type Emulator struct {
	GBA      *gba.GBA
	Rom      []byte
	RomDir   string
	debugger *debug.Debugger
	pause    bool
	reset    bool
}

func New(romData []byte, romDir string) *Emulator {
	g := gba.New(romData, joypad.Handler, audio.Stream)
	audio.Reset(&g.Sound.Enable)

	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle(g.CartHeader.Title)
	ebiten.SetWindowSize(240*2, 160*2)

	e := &Emulator{
		GBA:    g,
		Rom:    romData,
		RomDir: romDir,
	}
	e.debugger = debug.New(g, &e.pause)
	e.setupCloseHandler()

	e.loadSav()
	return e
}

func (e *Emulator) ResetGBA() {
	e.writeSav()
	e.debugger.Reset(e.GBA)
	e.loadSav()

	e.reset = false
}

func (e *Emulator) Update() error {
	if e.pause {
		return nil
	}
	if e.reset {
		e.ResetGBA()
		return nil
	}

	defer e.GBA.PanicHandler("update", true)
	e.GBA.Update()
	audio.Play()

	select {
	case <-second.C:
		if e.GBA.DoSav {
			e.writeSav()
		}
	default:
	}
	return nil
}

func (e *Emulator) Draw(screen *ebiten.Image) {
	if e.pause {
		screen.ReplacePixels(cache)
		return
	}

	defer e.GBA.PanicHandler("gpu", true)
	cache = e.GBA.Draw()
	screen.ReplacePixels(cache)
}

func (e *Emulator) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 240, 160
}

func (e *Emulator) setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		e.GBA.Exit("Ctrl+C pressed in Terminal")
		os.Exit(0)
	}()
}
