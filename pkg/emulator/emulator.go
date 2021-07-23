package emulator

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pokemium/magia/pkg/emulator/audio"
	"github.com/pokemium/magia/pkg/gba"
)

type Emulator struct {
	GBA *gba.GBA
	Rom string
}

func New(romData []byte, romDir string) *Emulator {
	g := gba.New(romData)
	e := &Emulator{
		GBA: g,
		Rom: romDir,
	}
	e.setupCloseHandler()

	// setup audio
	audio.Reset(&g.Sound.Enable)
	e.GBA.SetAudioBuffer(audio.Stream)

	e.loadSav()
	return e
}

func (e *Emulator) Update() error {
	defer e.GBA.PanicHandler("core", true)
	e.GBA.Update()
	audio.Play()
	if e.GBA.DoSav && e.GBA.Frame%60 == 0 {
		e.writeSav()
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
