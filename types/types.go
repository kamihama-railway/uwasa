// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package types

import (
	"fmt"
	"math"
	"sync"
	"bytes"
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
	case int32:
		return Value{Type: ValInt, Num: uint64(val)}
	case int16:
		return Value{Type: ValInt, Num: uint64(val)}
	case int8:
		return Value{Type: ValInt, Num: uint64(val)}
	case float64:
		return Value{Type: ValFloat, Num: math.Float64bits(val)}
	case float32:
		return Value{Type: ValFloat, Num: math.Float64bits(float64(val))}
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

func ValToFloat64(v Value) (float64, bool) {
	switch v.Type {
	case ValFloat:
		return math.Float64frombits(v.Num), true
	case ValInt:
		return float64(int64(v.Num)), true
	}
	return 0, false
}

type Context interface {
	Get(name string) (val any, exists bool)
	Set(name string, value any) error
}

type MapContext struct {
	Vars map[string]any
}

var MapContextPool = sync.Pool{
	New: func() any {
		return &MapContext{}
	},
}

func NewMapContext(vars map[string]any) *MapContext {
	ctx := MapContextPool.Get().(*MapContext)
	if vars == nil {
		vars = make(map[string]any)
	}
	ctx.Vars = vars
	return ctx
}

func (c *MapContext) Get(name string) (any, bool) {
	val, ok := c.Vars[name]
	return val, ok
}

func (c *MapContext) Set(name string, value any) error {
	c.Vars[name] = value
	return nil
}

type BuiltinFunc func(args ...any) (any, error)

var Builtins = map[string]BuiltinFunc{
	"concat": func(args ...any) (any, error) {
		totalLen := 0
		argStrings := make([]string, len(args))
		for i, arg := range args {
			switch v := arg.(type) {
			case string:
				argStrings[i] = v
			case int64:
				argStrings[i] = fmt.Sprintf("%d", v)
			case float64:
				argStrings[i] = fmt.Sprintf("%g", v)
			case bool:
				argStrings[i] = fmt.Sprintf("%v", v)
			default:
				argStrings[i] = fmt.Sprintf("%v", v)
			}
			totalLen += len(argStrings[i])
		}

		buf := BufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		buf.Grow(totalLen)
		for _, s := range argStrings {
			buf.WriteString(s)
		}
		res := buf.String()
		BufferPool.Put(buf)
		return res, nil
	},
}

var BufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}
