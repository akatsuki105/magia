package main

import (
	"errors"
	"flag"
	"fmt"
	"mettaur/pkg/gba"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

var version string

const (
	title   = "Mettaur"
	exeName = "mettaur"
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
		showVersion = flag.Bool("v", false, "show version")
	)

	flag.Parse()
	if *showVersion {
		printVersion()
		return ExitCodeOK
	}

	path := flag.Arg(0)
	data, err := readROM(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read ROM data: %s\n", err)
		return ExitCodeError
	}

	emu := &Emulator{
		gba: gba.New(data),
	}
	emu.gba.Reset()

	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowSize(240, 160)
	if err := ebiten.RunGame(emu); err != nil {
		fmt.Fprintf(os.Stderr, "crash in emulation: %s\n", err)
	}
	return ExitCodeOK
}

func printVersion() {
	fmt.Println(title+":", version)
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
}

func (e *Emulator) Update() error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "crash in emulation: %s in 0x%08x\n", err, e.gba.PC())
			e.gba.Exit("")
			panic("")
		}
	}()
	e.gba.Update()
	return nil
}
func (e *Emulator) Draw(screen *ebiten.Image) {
	screen.DrawImage(ebiten.NewImageFromImage(e.gba.Draw()), nil)
}
func (e *Emulator) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 240, 160
}
