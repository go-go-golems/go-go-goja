package pbconv

import "math"

func uint32FromInt(v int) uint32 {
	if v <= 0 {
		return 0
	}
	if v > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(v) // #nosec G115 -- value is clamped to uint32 range above.
}
