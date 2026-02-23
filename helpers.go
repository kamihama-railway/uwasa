package uwasa

import (
	"math"
)

func isValTruthyAny(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

func boolToUint64(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

func valToFloat64(v Value) (float64, bool) {
	switch v.Type {
	case ValInt:
		return float64(int64(v.Num)), true
	case ValFloat:
		return math.Float64frombits(v.Num), true
	case ValBool:
		if v.Num != 0 {
			return 1.0, true
		}
		return 0.0, true
	}
	return 0.0, false
}
