package main

import (
	"errors"
	"flag"
	"fmt"
	"magia/pkg/gba"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/hajimehoshi/ebiten/v2"
)

var version string

const (
	title   = "Magia"
	exeName = "magia"
)

// ExitCode represents program's status code
type ExitCode int

// exit code
const (
	ExitCodeOK ExitCode = iota
	ExitCodeError
)

func init() {
	if version == "" {
		version = "Develop"
	}

	flag.Usage = func() {
		usage := fmt.Sprintf(`Usage:
    %s [arg] [input]
input: a filepath
Arguments: 
`, exeName)

		fmt.Fprintf(os.Stderr, usage)
		flag.PrintDefaults()
	}
}

func main() {
	os.Exit(int(Run()))
}

// Run program
func Run() ExitCode {
	var (
		showVersion   = flag.Bool("v", false, "show version")
		debug         = flag.Bool("d", false, "exec in debug mode")
		showBIOSIntro = flag.Bool("b", false, "show BIOS intro")
		showCartInfo  = flag.Bool("c", false, "show cartridge info")
		mute          = flag.Bool("m", false, "mute sound")
	)

	flag.Parse()
	if *showVersion {
		fmt.Println(title+":", version)
		return ExitCodeOK
	}

	path := flag.Arg(0)
	data, err := readROM(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read ROM data: %s\n", err)
		return ExitCodeError
	}

	emu := &Emulator{
		gba: gba.New(data, *debug, *mute),
		rom: path,
	}
	if *showCartInfo {
		fmt.Println(emu.gba.CartInfo())
		return ExitCodeOK
	}

	emu.SetupCloseHandler()
	emu.loadSav()
	if *showBIOSIntro {
		emu.gba.Reset()
	} else {
		emu.gba.SoftReset()
	}

	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle(emu.gba.CartHeader.Title)
	ebiten.SetWindowSize(240*2, 160*2)
	if err := ebiten.RunGame(emu); err != nil {
		fmt.Fprintf(os.Stderr, "crash in emulation: %s\n", err)
	}
	return ExitCodeOK
}

func readROM(path string) ([]byte, error) {
	if path == "" {
		return []byte{}, errors.New("please select gba file path")
	}
	if filepath.Ext(path) != ".gba" {
		return []byte{}, errors.New("please select .gba file")
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return []byte{}, errors.New("fail to read file")
	}
	return bytes, nil
}

type Emulator struct {
	gba *gba.GBA
	rom string
}

func (e *Emulator) Update() error {
	defer e.gba.PanicHandler("core", true)
	e.gba.Update()
	if e.gba.DoSav && e.gba.Frame%60 == 0 {
		e.writeSav()
	}
	return nil
}

func (e *Emulator) Draw(screen *ebiten.Image) {
	defer e.gba.PanicHandler("gpu", true)
	screen.ReplacePixels(e.gba.Draw().Pix)
}

func (e *Emulator) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 240, 160
}

func (e *Emulator) SetupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		e.gba.Exit("Ctrl+C pressed in Terminal")
		os.Exit(0)
	}()
}

func (e *Emulator) writeSav() {
	path := strings.ReplaceAll(e.rom, ".gba", ".sav")
	if e.gba.RAM.HasFlash {
		os.WriteFile(path, e.gba.RAM.Flash[:], os.ModePerm)
	} else {
		os.WriteFile(path, e.gba.RAM.SRAM[:], os.ModePerm)
	}
	e.gba.DoSav = false
}

func (e *Emulator) loadSav() {
	path := strings.ReplaceAll(e.rom, ".gba", ".sav")
	if f, err := os.Stat(path); os.IsNotExist(err) || f.IsDir() {
		return
	} else if sav, err := os.ReadFile(path); err == nil {
		e.gba.LoadSav(sav)
	}
}
