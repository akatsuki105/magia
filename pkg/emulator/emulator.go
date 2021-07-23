package emulator

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pokemium/magia/pkg/emulator/audio"
	"github.com/pokemium/magia/pkg/gba"
)

var (
	second = time.NewTicker(time.Second)
)

type Emulator struct {
	GBA    *gba.GBA
	Rom    []byte
	RomDir string
}

func New(romData []byte, romDir string) *Emulator {
	g := gba.New(romData)
	e := &Emulator{
		GBA:    g,
		Rom:    romData,
		RomDir: romDir,
	}

	// setup audio
	audio.Reset(&g.Sound.Enable)
	e.GBA.SetAudioBuffer(audio.Stream)

	e.setupCloseHandler()
	e.loadSav()
	return e
}

func (e *Emulator) Update() error {
	defer e.GBA.PanicHandler("core", true)
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
