package gba

const ()

func (g *GBA) step() {
	if g.GetCPSRFlag(flagT) {
		g.thumbStep()
	} else {
		g.armStep()
	}

	// flagT may toggle in armStep/thumbStep
	if g.GetCPSRFlag(flagT) {
		g.R[15] += 2
	} else {
		g.R[15] += 4
	}
}
