package gba

const ()

func (g *GBA) step() {
	if g.GetCPSRFlag(flagT) {
		g.thumbStep()
	} else {
		g.armStep()
	}
}
