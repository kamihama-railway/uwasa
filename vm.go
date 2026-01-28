// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"bytes"
	"fmt"
	"math"
)

var (
	vmTrue  any = true
	vmFalse any = false
	vmNil   any = nil
)

type valKind byte

const (
	valNil valKind = iota
	valInt
	valFloat
	valBool
	valString
	valAny
)

type Value struct {
	kind valKind
	num  uint64
	ptr  any
}

func (v Value) ToAny() any {
	switch v.kind {
	case valInt:
		return int64(v.num)
	case valFloat:
		return math.Float64frombits(v.num)
	case valBool:
		if v.num != 0 {
			return vmTrue
		}
		return vmFalse
	case valString:
		return v.ptr.(string)
	case valAny:
		return v.ptr
	default:
		return vmNil
	}
}

func FromAny(a any) Value {
	if a == nil {
		return Value{kind: valNil}
	}
	switch v := a.(type) {
	case int64:
		return Value{kind: valInt, num: uint64(v)}
	case int:
		return Value{kind: valInt, num: uint64(int64(v))}
	case float64:
		return Value{kind: valFloat, num: math.Float64bits(v)}
	case bool:
		if v {
			return Value{kind: valBool, num: 1}
		}
		return Value{kind: valBool, num: 0}
	case string:
		return Value{kind: valString, ptr: v}
	default:
		return Value{kind: valAny, ptr: v}
	}
}

type vmInstruction struct {
	op   OpCode
	arg1 uint16
	arg2 uint16
	arg3 uint16
}

type RenderedBytecode struct {
	Instructions  []vmInstruction
	Constants     []Value
	VariableNames []string
	Builtins      []BuiltinFunc
}

func (rb *RenderedBytecode) String() string {
	var out string
	for i, ins := range rb.Instructions {
		out += fmt.Sprintf("%04d %v arg1:%d arg2:%d arg3:%d\n", i, definitions[ins.op].Name, ins.arg1, ins.arg2, ins.arg3)
	}
	return out
}

func RunVM(rendered *RenderedBytecode, ctx Context) (any, error) {
	var staticStack [128]Value
	stack := staticStack[:]
	sp := 0
	ip := 0
	ins := rendered.Instructions
	consts := rendered.Constants

	var mapCtx *MapContext
	if m, ok := ctx.(*MapContext); ok {
		mapCtx = m
	}

	var staticVars [16]Value
	var vars []Value
	if len(rendered.VariableNames) <= 16 {
		vars = staticVars[:len(rendered.VariableNames)]
	} else {
		vars = make([]Value, len(rendered.VariableNames))
	}

	for i, name := range rendered.VariableNames {
		var val any
		if mapCtx != nil {
			val = mapCtx.vars[name]
		} else {
			val, _ = ctx.Get(name)
		}
		vars[i] = FromAny(val)
	}

	hasSideEffects := false

	for ip < len(ins) {
		inst := ins[ip]
		op := inst.op

		switch op {
		case OpConstant:
			stack[sp] = consts[inst.arg1]; sp++
		case OpPop:
			if sp > 0 { sp-- }

		case OpAdd:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) + int64(r.num))}
			} else if l.kind == valFloat && r.kind == valFloat {
				stack[sp-1] = Value{kind: valFloat, num: math.Float64bits(math.Float64frombits(l.num) + math.Float64frombits(r.num))}
			} else if l.kind == valString && r.kind == valString {
				stack[sp-1] = Value{kind: valString, ptr: l.ptr.(string) + r.ptr.(string)}
			} else {
				res, err := evalArithmetic("+", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}
		case OpAddGlobal:
			l := stack[sp-1]; r := vars[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) + int64(r.num))}
			} else {
				res, err := evalArithmetic("+", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}
		case OpAddConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) + int64(r.num))}
			} else {
				res, err := evalArithmetic("+", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpSub:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) - int64(r.num))}
			} else {
				res, err := evalArithmetic("-", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}
		case OpSubGlobal:
			l := stack[sp-1]; r := vars[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) - int64(r.num))}
			} else {
				res, err := evalArithmetic("-", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}
		case OpSubConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) - int64(r.num))}
			} else {
				res, err := evalArithmetic("-", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpMul:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) * int64(r.num))}
			} else {
				res, err := evalArithmetic("*", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}
		case OpMulGlobal:
			l := stack[sp-1]; r := vars[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) * int64(r.num))}
			} else {
				res, err := evalArithmetic("*", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}
		case OpMulConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) * int64(r.num))}
			} else {
				res, err := evalArithmetic("*", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpDiv:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			if r.kind == valInt && l.kind == valInt {
				if r.num == 0 { return nil, fmt.Errorf("division by zero") }
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) / int64(r.num))}
			} else {
				res, err := evalArithmetic("/", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}
		case OpDivGlobal:
			l := stack[sp-1]; r := vars[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				if r.num == 0 { return nil, fmt.Errorf("division by zero") }
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) / int64(r.num))}
			} else {
				res, err := evalArithmetic("/", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}
		case OpDivConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				if r.num == 0 { return nil, fmt.Errorf("division by zero") }
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) / int64(r.num))}
			} else {
				res, err := evalArithmetic("/", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpMod:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			res, err := evalArithmetic("%", l.ToAny(), r.ToAny())
			if err != nil { return nil, err }
			stack[sp-1] = FromAny(res)

		case OpMinus:
			l := stack[sp-1]
			if l.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(-int64(l.num))}
			} else if l.kind == valFloat {
				stack[sp-1] = Value{kind: valFloat, num: math.Float64bits(-math.Float64frombits(l.num))}
			} else {
				res, err := evalPrefixExpression("-", l.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpEqual:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			res := false
			if l.kind == r.kind && l.num == r.num && l.ptr == r.ptr {
				res = true
			} else {
				val, _ := evalComparison("==", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpEqualConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			res := false
			if l.kind == r.kind && l.num == r.num && l.ptr == r.ptr {
				res = true
			} else {
				val, _ := evalComparison("==", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpGreater:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) > int64(r.num)
			} else {
				val, _ := evalComparison(">", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpGreaterConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) > int64(r.num)
			} else {
				val, _ := evalComparison(">", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpLess:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) < int64(r.num)
			} else {
				val, _ := evalComparison("<", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpLessConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) < int64(r.num)
			} else {
				val, _ := evalComparison("<", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpGreaterEqual:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			val, _ := evalComparison(">=", l.ToAny(), r.ToAny())
			if val.(bool) { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpLessEqual:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			val, _ := evalComparison("<=", l.ToAny(), r.ToAny())
			if val.(bool) { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpGreaterEqualConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			val, _ := evalComparison(">=", l.ToAny(), r.ToAny())
			if val.(bool) { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpLessEqualConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			val, _ := evalComparison("<=", l.ToAny(), r.ToAny())
			if val.(bool) { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpGetGlobal:
			stack[sp] = vars[inst.arg1]; sp++
		case OpSetGlobal:
			vars[inst.arg1] = stack[sp-1]
			hasSideEffects = true

		case OpJump:
			ip = int(inst.arg1) - 1
		case OpJumpIfFalse:
			cond := stack[sp-1]
			isTruthy := true
			if cond.kind == valBool { isTruthy = cond.num != 0 } else if cond.kind == valNil { isTruthy = false }
			if !isTruthy { ip = int(inst.arg1) - 1 }
		case OpJumpIfFalsePop:
			cond := stack[sp-1]; sp--
			isTruthy := true
			if cond.kind == valBool { isTruthy = cond.num != 0 } else if cond.kind == valNil { isTruthy = false }
			if !isTruthy { ip = int(inst.arg1) - 1 }
		case OpJumpIfTrue:
			cond := stack[sp-1]
			isTruthy := true
			if cond.kind == valBool { isTruthy = cond.num != 0 } else if cond.kind == valNil { isTruthy = false }
			if isTruthy { ip = int(inst.arg1) - 1 }
		case OpToBool:
			cond := stack[sp-1]
			isTruthy := true
			if cond.kind == valBool { isTruthy = cond.num != 0 } else if cond.kind == valNil { isTruthy = false }
			if isTruthy { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }

		case OpCall:
			numArgs := int(inst.arg1)
			funcName := stack[sp-1].ptr.(string); sp--
			var staticArgs [8]any
			var args []any
			if numArgs <= 8 { args = staticArgs[:numArgs] } else { args = make([]any, numArgs) }
			for i := numArgs - 1; i >= 0; i-- { args[i] = stack[sp-1].ToAny(); sp-- }
			builtin, ok := builtins[funcName]
			if !ok { return nil, fmt.Errorf("builtin function %s not found", funcName) }
			res, err := builtin(args...)
			if err != nil { return nil, err }
			stack[sp] = FromAny(res); sp++

		case OpCallResolved:
			numArgs := int(inst.arg1)
			builtin := rendered.Builtins[inst.arg2]
			var staticArgs [8]any
			var args []any
			if numArgs <= 8 { args = staticArgs[:numArgs] } else { args = make([]any, numArgs) }
			for i := numArgs - 1; i >= 0; i-- { args[i] = stack[sp-1].ToAny(); sp-- }
			res, err := builtin(args...)
			if err != nil { return nil, err }
			stack[sp] = FromAny(res); sp++

		case OpConcat:
			numArgs := int(inst.arg1)
			var argStrings [16]string
			var ss []string
			if numArgs <= 16 { ss = argStrings[:numArgs] } else { ss = make([]string, numArgs) }
			totalLen := 0
			for i := numArgs - 1; i >= 0; i-- {
				v := stack[sp-1]; sp--
				var s string
				switch v.kind {
				case valString: s = v.ptr.(string)
				case valInt: s = fmt.Sprintf("%d", int64(v.num))
				case valFloat: s = fmt.Sprintf("%g", math.Float64frombits(v.num))
				case valBool: s = fmt.Sprintf("%v", v.num != 0)
				case valNil: s = "nil"
				default: s = fmt.Sprintf("%v", v.ptr)
				}
				ss[i] = s
				totalLen += len(s)
			}
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.Grow(totalLen)
			for _, s := range ss { buf.WriteString(s) }
			res := buf.String()
			bufferPool.Put(buf)
			stack[sp] = Value{kind: valString, ptr: res}; sp++

		case OpFusedCompareGlobalConstJumpIfFalse:
			l := vars[inst.arg1]; r := consts[inst.arg2]
			res := false
			if l.kind == r.kind && l.num == r.num && l.ptr == r.ptr {
				res = true
			} else {
				val, _ := evalComparison("==", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if !res { ip = int(inst.arg3) - 1 }

		default:
		}
		ip++
	}

	if hasSideEffects {
		for i, name := range rendered.VariableNames {
			_ = ctx.Set(name, vars[i].ToAny())
		}
	}

	if sp == 0 { return nil, nil }
	return stack[sp-1].ToAny(), nil
}
