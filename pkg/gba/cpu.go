package gba

const ()

func (g *GBA) step() {
	t := g.GetCPSRFlag(flagT)
	if t {
		g.thumbStep()
		return
	}
	g.armStep()
}
