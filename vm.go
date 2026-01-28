// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"bytes"
	"fmt"
	"math"
)

var (
	vmTrue    any = true
	vmFalse   any = false
	vmNil     any = nil
	smallInts [256]any
)

func init() {
	for i := range smallInts {
		smallInts[i] = int64(i)
	}
}

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
		val := int64(v.num)
		if val >= 0 && val < 256 {
			return smallInts[val]
		}
		return val
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
		name := "UNKNOWN"
		if def, ok := definitions[ins.op]; ok {
			name = def.Name
		}
		out += fmt.Sprintf("%04d %v arg1:%d arg2:%d arg3:%d\n", i, name, ins.arg1, ins.arg2, ins.arg3)
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

	numIns := len(ins)

	for ip < numIns {
		inst := &ins[ip]
		ip++

		switch inst.op {
		case OpConstant:
			stack[sp] = consts[inst.arg1]
			sp++
		case OpPop:
			if sp > 0 { sp-- }

		case OpAdd:
			sp--
			l := &stack[sp-1]
			r := &stack[sp]
			if l.kind == valInt && r.kind == valInt {
				l.num = uint64(int64(l.num) + int64(r.num))
			} else if l.kind == valFloat && r.kind == valFloat {
				l.num = math.Float64bits(math.Float64frombits(l.num) + math.Float64frombits(r.num))
			} else if l.kind == valString && r.kind == valString {
				l.ptr = l.ptr.(string) + r.ptr.(string)
			} else {
				res, err := evalArithmetic("+", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}
		case OpAddGlobal:
			l := &stack[sp-1]
			r := &vars[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				l.num = uint64(int64(l.num) + int64(r.num))
			} else {
				res, err := evalArithmetic("+", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}
		case OpAddConst:
			l := &stack[sp-1]
			r := &consts[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				l.num = uint64(int64(l.num) + int64(r.num))
			} else {
				res, err := evalArithmetic("+", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}

		case OpSub:
			sp--
			l := &stack[sp-1]
			r := &stack[sp]
			if l.kind == valInt && r.kind == valInt {
				l.num = uint64(int64(l.num) - int64(r.num))
			} else {
				res, err := evalArithmetic("-", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}
		case OpSubGlobal:
			l := &stack[sp-1]
			r := &vars[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				l.num = uint64(int64(l.num) - int64(r.num))
			} else {
				res, err := evalArithmetic("-", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}
		case OpSubConst:
			l := &stack[sp-1]
			r := &consts[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				l.num = uint64(int64(l.num) - int64(r.num))
			} else {
				res, err := evalArithmetic("-", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}

		case OpMul:
			sp--
			l := &stack[sp-1]
			r := &stack[sp]
			if l.kind == valInt && r.kind == valInt {
				l.num = uint64(int64(l.num) * int64(r.num))
			} else {
				res, err := evalArithmetic("*", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}
		case OpMulGlobal:
			l := &stack[sp-1]
			r := &vars[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				l.num = uint64(int64(l.num) * int64(r.num))
			} else {
				res, err := evalArithmetic("*", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}
		case OpMulConst:
			l := &stack[sp-1]
			r := &consts[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				l.num = uint64(int64(l.num) * int64(r.num))
			} else {
				res, err := evalArithmetic("*", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}

		case OpDiv:
			sp--
			l := &stack[sp-1]
			r := &stack[sp]
			if r.kind == valInt && l.kind == valInt {
				if r.num == 0 { return nil, fmt.Errorf("division by zero") }
				l.num = uint64(int64(l.num) / int64(r.num))
			} else {
				res, err := evalArithmetic("/", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}
		case OpDivGlobal:
			l := &stack[sp-1]
			r := &vars[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				if r.num == 0 { return nil, fmt.Errorf("division by zero") }
				l.num = uint64(int64(l.num) / int64(r.num))
			} else {
				res, err := evalArithmetic("/", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}
		case OpDivConst:
			l := &stack[sp-1]
			r := &consts[inst.arg1]
			if l.kind == valInt && r.kind == valInt {
				if r.num == 0 { return nil, fmt.Errorf("division by zero") }
				l.num = uint64(int64(l.num) / int64(r.num))
			} else {
				res, err := evalArithmetic("/", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}

		case OpMod:
			sp--
			l := &stack[sp-1]
			r := &stack[sp]
			res, err := evalArithmetic("%", l.ToAny(), r.ToAny())
			if err != nil { return nil, err }
			*l = FromAny(res)

		case OpMinus:
			l := &stack[sp-1]
			if l.kind == valInt {
				l.num = uint64(-int64(l.num))
			} else if l.kind == valFloat {
				l.num = math.Float64bits(-math.Float64frombits(l.num))
			} else {
				res, err := evalPrefixExpression("-", l.ToAny())
				if err != nil { return nil, err }
				*l = FromAny(res)
			}

		case OpEqual:
			sp--
			l := &stack[sp-1]
			r := &stack[sp]
			res := false
			if l.kind == r.kind && l.num == r.num && l.ptr == r.ptr {
				res = true
			} else {
				val, _ := evalComparison("==", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { *l = Value{kind: valBool, num: 1} } else { *l = Value{kind: valBool, num: 0} }

		case OpEqualConst:
			l := &stack[sp-1]
			r := &consts[inst.arg1]
			res := false
			if l.kind == r.kind && l.num == r.num && l.ptr == r.ptr {
				res = true
			} else {
				val, _ := evalComparison("==", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { *l = Value{kind: valBool, num: 1} } else { *l = Value{kind: valBool, num: 0} }

		case OpGreater:
			sp--
			l := &stack[sp-1]
			r := &stack[sp]
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) > int64(r.num)
			} else {
				val, _ := evalComparison(">", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { *l = Value{kind: valBool, num: 1} } else { *l = Value{kind: valBool, num: 0} }

		case OpGreaterConst:
			l := &stack[sp-1]
			r := &consts[inst.arg1]
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) > int64(r.num)
			} else {
				val, _ := evalComparison(">", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { *l = Value{kind: valBool, num: 1} } else { *l = Value{kind: valBool, num: 0} }

		case OpLess:
			sp--
			l := &stack[sp-1]
			r := &stack[sp]
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) < int64(r.num)
			} else {
				val, _ := evalComparison("<", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { *l = Value{kind: valBool, num: 1} } else { *l = Value{kind: valBool, num: 0} }

		case OpLessConst:
			l := &stack[sp-1]
			r := &consts[inst.arg1]
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) < int64(r.num)
			} else {
				val, _ := evalComparison("<", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if res { *l = Value{kind: valBool, num: 1} } else { *l = Value{kind: valBool, num: 0} }

		case OpGreaterEqual:
			sp--
			l := &stack[sp-1]
			r := &stack[sp]
			val, _ := evalComparison(">=", l.ToAny(), r.ToAny())
			if val.(bool) { *l = Value{kind: valBool, num: 1} } else { *l = Value{kind: valBool, num: 0} }

		case OpLessEqual:
			sp--
			l := &stack[sp-1]
			r := &stack[sp]
			val, _ := evalComparison("<=", l.ToAny(), r.ToAny())
			if val.(bool) { *l = Value{kind: valBool, num: 1} } else { *l = Value{kind: valBool, num: 0} }

		case OpGreaterEqualConst:
			l := &stack[sp-1]
			r := &consts[inst.arg1]
			val, _ := evalComparison(">=", l.ToAny(), r.ToAny())
			if val.(bool) { *l = Value{kind: valBool, num: 1} } else { *l = Value{kind: valBool, num: 0} }

		case OpLessEqualConst:
			l := &stack[sp-1]
			r := &consts[inst.arg1]
			val, _ := evalComparison("<=", l.ToAny(), r.ToAny())
			if val.(bool) { *l = Value{kind: valBool, num: 1} } else { *l = Value{kind: valBool, num: 0} }

		case OpGetGlobal:
			stack[sp] = vars[inst.arg1]
			sp++
		case OpSetGlobal:
			val := stack[sp-1]
			vars[inst.arg1] = val
			_ = ctx.Set(rendered.VariableNames[inst.arg1], val.ToAny())

		case OpJump:
			ip = int(inst.arg1)
		case OpJumpIfFalse:
			cond := &stack[sp-1]
			isTruthy := true
			if cond.kind == valBool { isTruthy = cond.num != 0 } else if cond.kind == valNil { isTruthy = false }
			if !isTruthy { ip = int(inst.arg1) }
		case OpJumpIfFalsePop:
			sp--
			cond := &stack[sp]
			isTruthy := true
			if cond.kind == valBool { isTruthy = cond.num != 0 } else if cond.kind == valNil { isTruthy = false }
			if !isTruthy { ip = int(inst.arg1) }
		case OpJumpIfTrue:
			cond := &stack[sp-1]
			isTruthy := true
			if cond.kind == valBool { isTruthy = cond.num != 0 } else if cond.kind == valNil { isTruthy = false }
			if isTruthy { ip = int(inst.arg1) }
		case OpToBool:
			cond := &stack[sp-1]
			isTruthy := true
			if cond.kind == valBool { isTruthy = cond.num != 0 } else if cond.kind == valNil { isTruthy = false }
			if isTruthy { *cond = Value{kind: valBool, num: 1} } else { *cond = Value{kind: valBool, num: 0} }

		case OpCall:
			numArgs := int(inst.arg1)
			sp--
			funcName := stack[sp].ptr.(string)
			var staticArgs [8]any
			var args []any
			if numArgs <= 8 { args = staticArgs[:numArgs] } else { args = make([]any, numArgs) }
			for i := numArgs - 1; i >= 0; i-- { sp--; args[i] = stack[sp].ToAny() }
			builtin, ok := builtins[funcName]
			if !ok { return nil, fmt.Errorf("builtin function %s not found", funcName) }
			res, err := builtin(args...)
			if err != nil { return nil, err }
			stack[sp] = FromAny(res)
			sp++

		case OpCallResolved:
			numArgs := int(inst.arg1)
			builtin := rendered.Builtins[inst.arg2]
			var staticArgs [8]any
			var args []any
			if numArgs <= 8 { args = staticArgs[:numArgs] } else { args = make([]any, numArgs) }
			for i := numArgs - 1; i >= 0; i-- { sp--; args[i] = stack[sp].ToAny() }
			res, err := builtin(args...)
			if err != nil { return nil, err }
			stack[sp] = FromAny(res)
			sp++

		case OpConcat:
			numArgs := int(inst.arg1)
			var argStrings [16]string
			var ss []string
			if numArgs <= 16 { ss = argStrings[:numArgs] } else { ss = make([]string, numArgs) }
			totalLen := 0
			for i := numArgs - 1; i >= 0; i-- {
				sp--
				v := &stack[sp]
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
			stack[sp] = Value{kind: valString, ptr: res}
			sp++

		case OpFusedCompareGlobalConstJumpIfFalse:
			l := &vars[inst.arg1]
			r := &consts[inst.arg2]
			res := false
			if l.kind == r.kind && l.num == r.num && l.ptr == r.ptr {
				res = true
			} else {
				val, _ := evalComparison("==", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if !res { ip = int(inst.arg3) }

		case OpFusedGreaterGlobalConstJumpIfFalse:
			l := &vars[inst.arg1]
			r := &consts[inst.arg2]
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) > int64(r.num)
			} else {
				val, _ := evalComparison(">", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if !res { ip = int(inst.arg3) }

		case OpFusedLessGlobalConstJumpIfFalse:
			l := &vars[inst.arg1]
			r := &consts[inst.arg2]
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) < int64(r.num)
			} else {
				val, _ := evalComparison("<", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if !res { ip = int(inst.arg3) }

		case OpFusedGreaterEqualGlobalConstJumpIfFalse:
			l := &vars[inst.arg1]
			r := &consts[inst.arg2]
			val, _ := evalComparison(">=", l.ToAny(), r.ToAny())
			if !val.(bool) { ip = int(inst.arg3) }

		case OpFusedLessEqualGlobalConstJumpIfFalse:
			l := &vars[inst.arg1]
			r := &consts[inst.arg2]
			val, _ := evalComparison("<=", l.ToAny(), r.ToAny())
			if !val.(bool) { ip = int(inst.arg3) }

		case OpFusedGreaterConstJumpIfFalsePop:
			sp--
			l := &stack[sp]
			r := &consts[inst.arg1]
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) > int64(r.num)
			} else {
				val, _ := evalComparison(">", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if !res { ip = int(inst.arg2) }

		case OpFusedLessConstJumpIfFalsePop:
			sp--
			l := &stack[sp]
			r := &consts[inst.arg1]
			res := false
			if l.kind == valInt && r.kind == valInt {
				res = int64(l.num) < int64(r.num)
			} else {
				val, _ := evalComparison("<", l.ToAny(), r.ToAny())
				res = val.(bool)
			}
			if !res { ip = int(inst.arg2) }

		case OpFusedGreaterEqualConstJumpIfFalsePop:
			sp--
			l := &stack[sp]
			r := &consts[inst.arg1]
			val, _ := evalComparison(">=", l.ToAny(), r.ToAny())
			if !val.(bool) { ip = int(inst.arg2) }

		case OpFusedLessEqualConstJumpIfFalsePop:
			sp--
			l := &stack[sp]
			r := &consts[inst.arg1]
			val, _ := evalComparison("<=", l.ToAny(), r.ToAny())
			if !val.(bool) { ip = int(inst.arg2) }

		default:
		}
	}


	if sp == 0 { return nil, nil }
	return stack[sp-1].ToAny(), nil
}
