package gba

// Cond represents condition
type Cond byte

// condition
const (
	EQ Cond = iota // ==
	NE             // !=
	CS             // carry
	CC             // not carry
	MI             // minus
	PL             // plus
	VS             // overflow
	VC             // not-overflow
	HI             // higher
	LS             // not carry
	GE             // >=
	LT             // <
	GT             // >
	LE             // <=
)

// Check returns if instruction condition is ok
func (g *GBA) Check(cond Cond) bool {
	switch cond {
	case EQ:
		return g.GetCPSRFlag(flagZ)
	case NE:
		return !g.GetCPSRFlag(flagZ)
	case CS:
		return g.GetCPSRFlag(flagC)
	case CC:
		return !g.GetCPSRFlag(flagC)
	case MI:
		return g.GetCPSRFlag(flagN)
	case PL:
		return !g.GetCPSRFlag(flagN)
	case VS:
		return g.GetCPSRFlag(flagV)
	case VC:
		return !g.GetCPSRFlag(flagV)
	case HI:
		return g.GetCPSRFlag(flagC) && !g.GetCPSRFlag(flagZ)
	case LS:
		return !g.GetCPSRFlag(flagC)
	case GE:
		return g.GetCPSRFlag(flagN) == g.GetCPSRFlag(flagV)
	case LT:
		return g.GetCPSRFlag(flagN) != g.GetCPSRFlag(flagV)
	case GT:
		return !g.GetCPSRFlag(flagZ) && (g.GetCPSRFlag(flagN) == g.GetCPSRFlag(flagV))
	case LE:
		return g.GetCPSRFlag(flagZ) || (g.GetCPSRFlag(flagN) != g.GetCPSRFlag(flagV))
	default:
		return false
	}
}
