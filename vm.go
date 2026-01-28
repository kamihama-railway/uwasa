// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"math"
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
		return v.num != 0
	case valString:
		return v.ptr.(string)
	case valAny:
		return v.ptr
	default:
		return nil
	}
}

func FromAny(a any) Value {
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
	case nil:
		return Value{kind: valNil}
	default:
		return Value{kind: valAny, ptr: v}
	}
}

func (v Value) toFloat() (float64, bool) {
	if v.kind == valFloat {
		return math.Float64frombits(v.num), true
	}
	if v.kind == valInt {
		return float64(int64(v.num)), true
	}
	return 0, false
}

type vmInstruction struct {
	op   OpCode
	arg1 uint16
	arg2 uint16
	arg3 uint16
}

type RenderedBytecode struct {
	Instructions []vmInstruction
	Constants    []Value
}

func RunVM(rendered *RenderedBytecode, ctx Context) (any, error) {
	var staticStack [64]Value
	stack := staticStack[:]
	sp := 0
	ip := 0
	ins := rendered.Instructions
	consts := rendered.Constants

	var mapCtx *MapContext
	if m, ok := ctx.(*MapContext); ok {
		mapCtx = m
	}

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
			} else {
				stack[sp-1] = valArithmetic("+", l, r)
			}
		case OpSub:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) - int64(r.num))}
			} else {
				stack[sp-1] = valArithmetic("-", l, r)
			}
		case OpMul:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) * int64(r.num))}
			} else {
				stack[sp-1] = valArithmetic("*", l, r)
			}
		case OpDiv:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			stack[sp-1] = valArithmetic("/", l, r)
		case OpMod:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			stack[sp-1] = valArithmetic("%", l, r)
		case OpEqual:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			if l.kind == r.kind && l.num == r.num && l.ptr == r.ptr {
				stack[sp-1] = Value{kind: valBool, num: 1}
			} else {
				stack[sp-1] = valCompare("==", l, r)
			}
		case OpEqualConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			if l.kind == r.kind && l.num == r.num && l.ptr == r.ptr {
				stack[sp-1] = Value{kind: valBool, num: 1}
			} else {
				stack[sp-1] = valCompare("==", l, r)
			}
		case OpGreater:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			stack[sp-1] = valCompare(">", l, r)
		case OpGreaterConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			stack[sp-1] = valCompare(">", l, r)
		case OpLess:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			stack[sp-1] = valCompare("<", l, r)
		case OpLessConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			stack[sp-1] = valCompare("<", l, r)
		case OpGreaterEqual:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			stack[sp-1] = valCompare(">=", l, r)
		case OpGreaterEqualConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			stack[sp-1] = valCompare(">=", l, r)
		case OpLessEqual:
			r := stack[sp-1]; l := stack[sp-2]; sp--
			stack[sp-1] = valCompare("<=", l, r)
		case OpLessEqualConst:
			l := stack[sp-1]; r := consts[inst.arg1]
			stack[sp-1] = valCompare("<=", l, r)
		case OpMinus:
			r := stack[sp-1]
			if r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(-int64(r.num))}
			} else if r.kind == valFloat {
				stack[sp-1] = Value{kind: valFloat, num: math.Float64bits(-math.Float64frombits(r.num))}
			} else {
				res, _ := evalPrefixExpression("-", r.ToAny())
				stack[sp-1] = FromAny(res)
			}
		case OpGetGlobal:
			name := consts[inst.arg1].ptr.(string)
			var val any
			if mapCtx != nil { val = mapCtx.vars[name] } else { val, _ = ctx.Get(name) }
			stack[sp] = FromAny(val); sp++
		case OpSetGlobal:
			name := consts[inst.arg1].ptr.(string)
			val := stack[sp-1]
			_ = ctx.Set(name, val.ToAny())
		case OpJump:
			ip = int(inst.arg1) - 1
		case OpJumpIfFalse:
			cond := stack[sp-1]
			isTruthy := true
			if cond.kind == valBool { isTruthy = cond.num != 0 } else if cond.kind == valNil { isTruthy = false }
			if !isTruthy { ip = int(inst.arg1) - 1 }
		case OpJumpIfTrue:
			cond := stack[sp-1]
			isTruthy := true
			if cond.kind == valBool { isTruthy = cond.num != 0 } else if cond.kind == valNil { isTruthy = false }
			if isTruthy { ip = int(inst.arg1) - 1 }
		case OpToBool:
			if isTruthyValue(stack[sp-1]) { stack[sp-1] = Value{kind: valBool, num: 1} } else { stack[sp-1] = Value{kind: valBool, num: 0} }
		case OpCall:
			numArgs := int(inst.arg1)
			funcName := stack[sp-1].ptr.(string); sp--
			var staticArgs [8]any
			var args []any
			if numArgs <= 8 { args = staticArgs[:numArgs] } else { args = make([]any, numArgs) }
			for i := numArgs - 1; i >= 0; i-- { args[i] = stack[sp-1].ToAny(); sp-- }
			builtin, _ := builtins[funcName]
			res, _ := builtin(args...)
			stack[sp] = FromAny(res); sp++
		case OpFusedCompareGlobalConstJumpIfFalse:
			name := consts[inst.arg1].ptr.(string)
			var val any
			if mapCtx != nil { val = mapCtx.vars[name] } else { val, _ = ctx.Get(name) }
			l := FromAny(val); r := consts[inst.arg2]
			if valCompare("==", l, r).num == 0 { ip = int(inst.arg3) - 1 }
		case OpAddGlobal:
			l := stack[sp-1]
			name := consts[inst.arg1].ptr.(string)
			var val any
			if mapCtx != nil { val = mapCtx.vars[name] } else { val, _ = ctx.Get(name) }
			r := FromAny(val)
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) + int64(r.num))}
			} else {
				stack[sp-1] = valArithmetic("+", l, r)
			}
		}
		ip++
	}
	if sp == 0 { return nil, nil }
	return stack[sp-1].ToAny(), nil
}

func valArithmetic(op string, l, r Value) Value {
	if l.kind == valInt && r.kind == valInt {
		lv, rv := int64(l.num), int64(r.num)
		switch op {
		case "+": return Value{kind: valInt, num: uint64(lv + rv)}
		case "-": return Value{kind: valInt, num: uint64(lv - rv)}
		case "*": return Value{kind: valInt, num: uint64(lv * rv)}
		case "/": if rv == 0 { return Value{kind: valNil} }; return Value{kind: valInt, num: uint64(lv / rv)}
		case "%": if rv == 0 { return Value{kind: valNil} }; return Value{kind: valInt, num: uint64(lv % rv)}
		}
	}
	if l.kind == valString && r.kind == valString && op == "+" {
		return Value{kind: valString, ptr: l.ptr.(string) + r.ptr.(string)}
	}
	res, _ := evalArithmetic(op, l.ToAny(), r.ToAny())
	return FromAny(res)
}

func valCompare(op string, l, r Value) Value {
	if l.kind == valInt && r.kind == valInt {
		lv, rv := int64(l.num), int64(r.num)
		var res bool
		switch op {
		case "==": res = lv == rv
		case ">":  res = lv > rv
		case "<":  res = lv < rv
		case ">=": res = lv >= rv
		case "<=": res = lv <= rv
		}
		if res { return Value{kind: valBool, num: 1} }
		return Value{kind: valBool, num: 0}
	}
	if fl, okL := l.toFloat(); okL {
		if fr, okR := r.toFloat(); okR {
			var res bool
			switch op {
			case "==": res = fl == fr
			case ">":  res = fl > fr
			case "<":  res = fl < fr
			case ">=": res = fl >= fr
			case "<=": res = fl <= fr
			}
			if res { return Value{kind: valBool, num: 1} }
			return Value{kind: valBool, num: 0}
		}
	}
	res, _ := evalComparison(op, l.ToAny(), r.ToAny())
	return FromAny(res)
}

func isTruthyValue(v Value) bool {
	if v.kind == valBool { return v.num != 0 }
	if v.kind == valNil { return false }
	return true
}
