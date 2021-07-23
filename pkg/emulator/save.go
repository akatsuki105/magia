package emulator

import (
	"os"
	"strings"
)

func (e *Emulator) writeSav() {
	path := strings.ReplaceAll(e.RomDir, ".gba", ".sav")
	if e.GBA.RAM.HasFlash {
		os.WriteFile(path, e.GBA.RAM.Flash[:], os.ModePerm)
	} else {
		os.WriteFile(path, e.GBA.RAM.SRAM[:], os.ModePerm)
	}
	e.GBA.DoSav = false
}

func (e *Emulator) loadSav() {
	path := strings.ReplaceAll(e.RomDir, ".gba", ".sav")
	if f, err := os.Stat(path); os.IsNotExist(err) || f.IsDir() {
		return
	} else if sav, err := os.ReadFile(path); err == nil {
		if len(sav) > 65536*2 {
			return
		}
		for i, b := range sav {
			if i < 65536 {
				e.GBA.RAM.SRAM[i] = b
			}
			e.GBA.RAM.Flash[i] = b
		}
	}
}
