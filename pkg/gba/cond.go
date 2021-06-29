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
	AL
	NV
)

var condLut = [16]uint16{
	0xF0F0, 0x0F0F, 0xCCCC, 0x3333, 0xFF00, 0x00FF, 0xAAAA, 0x5555, 0x0C0C, 0xF3F3, 0xAA55, 0x55AA, 0x0A05, 0xF5FA, 0xFFFF, 0x0000,
}

var cond2str = map[Cond]string{EQ: "eq", NE: "ne", CS: "cs", CC: "cc", MI: "mi", PL: "pl", VS: "vs", VC: "vc", HI: "hi", LS: "ls", GE: "ge", LT: "lt", GT: "gt", LE: "le", AL: "al", NV: "nv"}

func (c Cond) String() string {
	if s, ok := cond2str[c]; ok {
		return s
	}
	return ""
}

// Check returns if instruction condition is ok
func (g *GBA) Check(cond Cond) bool {
	flags := g.Reg.CPSR >> 28
	return (condLut[cond] & (1 << flags)) != 0
}
