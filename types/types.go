package types

import (
	"math"
)

type ValueType byte

const (
	ValNil ValueType = iota
	ValInt
	ValFloat
	ValBool
	ValString
)

type Value struct {
	Type ValueType
	Num  uint64
	Str  string
}

func (v Value) ToInterface() any {
	switch v.Type {
	case ValInt:
		return int64(v.Num)
	case ValFloat:
		return math.Float64frombits(v.Num)
	case ValBool:
		return v.Num != 0
	case ValString:
		return v.Str
	default:
		return nil
	}
}

func FromInterface(v any) Value {
	switch val := v.(type) {
	case int64:
		return Value{Type: ValInt, Num: uint64(val)}
	case int:
		return Value{Type: ValInt, Num: uint64(val)}
	case float64:
		return Value{Type: ValFloat, Num: math.Float64bits(val)}
	case bool:
		if val {
			return Value{Type: ValBool, Num: 1}
		}
		return Value{Type: ValBool, Num: 0}
	case string:
		return Value{Type: ValString, Str: val}
	default:
		return Value{Type: ValNil}
	}
}

type Context interface {
	Get(name string) (any, bool)
	Set(name string, value any) error
}

func ValToFloat64(v Value) (float64, bool) {
	switch v.Type {
	case ValFloat:
		return math.Float64frombits(v.Num), true
	case ValInt:
		return float64(int64(v.Num)), true
	}
	return 0, false
}

func IsValTruthy(v Value) bool {
	switch v.Type {
	case ValBool:
		return v.Num != 0
	case ValNil:
		return false
	default:
		return true
	}
}

func BoolToUint64(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

type MapContext struct {
	Vars map[string]any
}

func NewMapContext(vars map[string]any) *MapContext {
	return &MapContext{Vars: vars}
}

func (m *MapContext) Get(name string) (any, bool) {
	val, ok := m.Vars[name]
	return val, ok
}

func (m *MapContext) Set(name string, value any) error {
	m.Vars[name] = value
	return nil
}

func (m *MapContext) Reset(vars map[string]any) {
	m.Vars = vars
}
