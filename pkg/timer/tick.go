package timer

func (ts *Timers) Tick() [4]bool {
	overflow, irq := false, [4]bool{}
	if ts[0].enable() {
		ts[0].Next--
		if ts[0].Next == 0 {
			overflow = ts[0].increment()
			if overflow && ts[0].overflow() {
				irq[0] = true
			}
		}
	}

	for i := 1; i < 4; i++ {
		if !ts[i].enable() {
			continue
		}

		countUp := false
		if ts[i].cascade() {
			countUp = overflow
			overflow = false
		} else {
			ts[i].Next--
			countUp = ts[1].Next == 0
		}

		if countUp {
			overflow = ts[i].increment()
			if overflow && ts[i].overflow() {
				irq[i] = true
			}
		}
	}

	return irq
}
