package gba

func (g *GBA) thumbStep() {
	inst := g.thumbFetch()
	g.thumbExec(inst)
}

func (g *GBA) thumbFetch() uint16 {
	pc := g.R[15]
	if g.lastAddr+2 == pc {
		// sequential
	} else {
		// non-sequential
	}
	return uint16(g.RAM.Get(pc))
}

func (g *GBA) thumbExec(inst uint16) {}
