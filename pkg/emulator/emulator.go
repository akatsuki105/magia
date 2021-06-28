package emulator

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pokemium/magia/pkg/emulator/audio"
	"github.com/pokemium/magia/pkg/gba"
)

type Emulator struct {
	GBA *gba.GBA
	Rom string
}

func New(g *gba.GBA, r string) *Emulator {
	e := &Emulator{
		GBA: g,
		Rom: r,
	}
	e.setupCloseHandler()

	// setup audio
	audio.Init()
	e.GBA.SetAudioBuffer(audio.Stream)

	return e
}

func (e *Emulator) Update() error {
	defer e.GBA.PanicHandler("core", true)
	e.GBA.Update()
	audio.Play()
	if e.GBA.DoSav && e.GBA.Frame%60 == 0 {
		e.WriteSav()
	}
	return nil
}

func (e *Emulator) Draw(screen *ebiten.Image) {
	defer e.GBA.PanicHandler("gpu", true)
	screen.ReplacePixels(e.GBA.Draw())
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

func (e *Emulator) WriteSav() {
	path := strings.ReplaceAll(e.Rom, ".gba", ".sav")
	if e.GBA.RAM.HasFlash {
		os.WriteFile(path, e.GBA.RAM.Flash[:], os.ModePerm)
	} else {
		os.WriteFile(path, e.GBA.RAM.SRAM[:], os.ModePerm)
	}
	e.GBA.DoSav = false
}

func (e *Emulator) LoadSav() {
	path := strings.ReplaceAll(e.Rom, ".gba", ".sav")
	if f, err := os.Stat(path); os.IsNotExist(err) || f.IsDir() {
		return
	} else if sav, err := os.ReadFile(path); err == nil {
		e.GBA.LoadSav(sav)
	}
}
