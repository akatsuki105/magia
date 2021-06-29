package util

import "math"

func Div(num, denom int32) (uint32, uint32, uint32) {
	const I32_MIN = -2147483647 - 1
	if denom == 0 {
		// If abs(num) > 1, this should hang, but that would be painful to
		// emulate in HLE, and no game will get into a state under normal
		// operation where it hangs...
		if num < 0 {
			return uint32(0xffff_ffff), uint32(num), 1
		}
		return 1, uint32(num), 1
	} else if denom == -1 && num == I32_MIN {
		return 0x8000_0000, 0, 0x8000_0000
	} else {
		result := num / denom
		mod := num % denom

		return uint32(result), uint32(mod), uint32(math.Abs(float64(result)))
	}
}

func Sqrt(x uint32) uint32 {
	if x == 0 {
		return 0
	}

	lower, upper, bound := uint32(0), x, uint32(1)

	for bound < upper {
		upper >>= 1
		bound <<= 1
	}

	for {
		upper = x
		accum := uint32(0)
		lower = bound

		for {
			oldLower := lower
			if lower <= upper>>1 {
				lower <<= 1
			}
			if oldLower >= upper>>1 {
				break
			}
		}

		for {
			accum <<= 1
			if upper >= lower {
				accum++
				upper -= lower
			}
			if lower == bound {
				break
			}
			lower >>= 1
		}

		oldBound := bound
		bound += accum
		bound >>= 1
		if bound >= oldBound {
			bound = oldBound
			break
		}
	}
	return bound
}

func ArcTan(i int32) (uint32, uint32, uint32) {
	a := -(int32(i*i) >> 14)
	b := (int32(0xa9*a) >> 14) + 0x390
	b = ((b * a) >> 14) + 0x91c
	b = ((b * a) >> 14) + 0xfb6
	b = ((b * a) >> 14) + 0x16aa
	b = ((b * a) >> 14) + 0x2081
	b = ((b * a) >> 14) + 0x3651
	b = ((b * a) >> 14) + 0xa2f9

	r0, r1, r3 := int32(i*b)>>16, a, b
	return uint32(r0), uint32(r1), uint32(r3)
}

func ArcTan2(x, y int32) (uint32, uint32) {
	if y == 0 {
		if x >= 0 {
			return 0, 0
		}

		return 0x8000, 0
	}

	if x == 0 {
		if y >= 0 {
			return 0x4000, uint32(y)
		}

		return 0xc000, uint32(y)
	}

	if y >= 0 {
		if x >= 0 {
			if x >= y {
				r0, r1, _ := ArcTan((y << 14) / x)

				return r0, r1
			}
		} else if -x >= y {
			r0, r1, _ := ArcTan((y << 14) / x)

			return r0 + 0x8000, r1
		}

		r0, r1, _ := ArcTan((x << 14) / y)

		return 0x4000 - r0, r1
	} else {
		if x <= 0 {
			if -x > -y {
				r0, r1, _ := ArcTan((y << 14) / x)

				return r0 + 0x8000, r1
			}
		} else if x >= -y {
			r0, r1, _ := ArcTan((y << 14) / x)

			return r0 + 0x10000, r1
		}

		r0, r1, _ := ArcTan((x << 14) / y)

		return 0xc000 - r0, r1
	}
}
