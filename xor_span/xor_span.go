package xor_span

import "math/bits"

type XorSpan [20]int

func (x *XorSpan) Add(v int) bool {
	for v != 0 {
		i := 31 - bits.LeadingZeros32(uint32(v))
		if x[i] != 0 {
			v ^= x[i]
		} else {
			x[i] = v
			return true
		}
	}
	return false
}
