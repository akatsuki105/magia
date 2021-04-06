package gpu

var objWin [240][160]bool

func (g *GPU) inWindow0(x, y int) bool {
	x2, x1 := int(g.IO[WIN0H]), int(g.IO[WIN0H+1])
	if x2 > 240 || x1 > x2 {
		x2 = 240
	}

	y2, y1 := int(g.IO[WIN0V]), int(g.IO[WIN0V+1])
	if y2 > 160 || y1 > y2 {
		y2 = 160
	}

	condX := (x >= x1) && (x < x2)
	condY := (y >= y1) && (y < y2)
	return condX && condY
}

func (g *GPU) inWindow1(x, y int) bool {
	x2, x1 := int(g.IO[WIN1H]), int(g.IO[WIN1H+1])
	if x2 > 240 || x1 > x2 {
		x2 = 240
	}

	y2, y1 := int(g.IO[WIN1V]), int(g.IO[WIN1V+1])
	if y2 > 160 || y1 > y2 {
		y2 = 160
	}

	condX := (x >= x1) && (x < x2)
	condY := (y >= y1) && (y < y2)
	return condX && condY
}

func (g *GPU) inObjWindow(x, y int) bool {
	return objWin[x][y]
}
