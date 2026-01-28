// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
)

func RunVM(bc *RenderedBytecode, ctx Context) (any, error) {
	if bc == nil || len(bc.Instructions) == 0 {
		return nil, nil
	}

	var stack [64]Value
	sp := -1
	pc := 0
	insts := bc.Instructions
	consts := bc.Constants
	nInsts := len(insts)

	// Optimization: check if ctx is MapContext to avoid interface calls
	mapCtx, isMapCtx := ctx.(*MapContext)

	for pc < nInsts {
		inst := insts[pc]
		pc++

		switch inst.Op {
		case OpPush:
			sp++
			stack[sp] = consts[inst.Arg]

		case OpPop:
			sp--

		case OpAdd:
			r := stack[sp]
			sp--
			l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Int: l.Int + r.Int}
			} else if l.Type == ValString && r.Type == ValString {
				stack[sp] = Value{Type: ValString, String: l.String + r.String}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Float: lf + rf}
			}

		case OpSub:
			r := stack[sp]
			sp--
			l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Int: l.Int - r.Int}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Float: lf - rf}
			}

		case OpMul:
			r := stack[sp]
			sp--
			l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Int: l.Int * r.Int}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Float: lf * rf}
			}

		case OpDiv:
			r := stack[sp]
			sp--
			l := stack[sp]
			if r.Type == ValInt && r.Int == 0 { return nil, fmt.Errorf("division by zero") }
			if r.Type == ValFloat && r.Float == 0 { return nil, fmt.Errorf("division by zero") }
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Int: l.Int / r.Int}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Float: lf / rf}
			}

		case OpMod:
			r := stack[sp]
			sp--
			l := stack[sp]
			if r.Type != ValInt || l.Type != ValInt {
				return nil, fmt.Errorf("modulo operator supports only integers")
			}
			if r.Int == 0 { return nil, fmt.Errorf("division by zero") }
			stack[sp] = Value{Type: ValInt, Int: l.Int % r.Int}

		case OpEqual:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == r.Type {
				switch l.Type {
				case ValInt: res = l.Int == r.Int
				case ValFloat: res = l.Float == r.Float
				case ValBool: res = l.Bool == r.Bool
				case ValString: res = l.String == r.String
				case ValNil: res = true
				}
			} else {
				lf, okL := valToFloat64(l)
				rf, okR := valToFloat64(r)
				if okL && okR {
					res = lf == rf
				}
			}
			stack[sp] = Value{Type: ValBool, Bool: res}

		case OpGreater:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = l.Int > r.Int
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf > rf
			}
			stack[sp] = Value{Type: ValBool, Bool: res}

		case OpLess:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = l.Int < r.Int
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf < rf
			}
			stack[sp] = Value{Type: ValBool, Bool: res}

		case OpGreaterEqual:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = l.Int >= r.Int
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf >= rf
			}
			stack[sp] = Value{Type: ValBool, Bool: res}

		case OpLessEqual:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = l.Int <= r.Int
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf <= rf
			}
			stack[sp] = Value{Type: ValBool, Bool: res}

		case OpAnd:
			r := stack[sp]
			sp--
			l := stack[sp]
			stack[sp] = Value{Type: ValBool, Bool: isValTruthy(l) && isValTruthy(r)}

		case OpOr:
			r := stack[sp]
			sp--
			l := stack[sp]
			stack[sp] = Value{Type: ValBool, Bool: isValTruthy(l) || isValTruthy(r)}

		case OpNot:
			l := stack[sp]
			stack[sp] = Value{Type: ValBool, Bool: !isValTruthy(l)}

		case OpJump:
			pc = int(inst.Arg)

		case OpJumpIfFalse:
			l := stack[sp]
			sp--
			if !isValTruthy(l) {
				pc = int(inst.Arg)
			}

		case OpJumpIfTrue:
			l := stack[sp]
			sp--
			if isValTruthy(l) {
				pc = int(inst.Arg)
			}

		case OpGetGlobal:
			name := consts[inst.Arg].String
			var val any
			if isMapCtx {
				val = mapCtx.vars[name]
			} else {
				val, _ = ctx.Get(name)
			}
			sp++
			stack[sp] = FromInterface(val)

		case OpSetGlobal:
			name := consts[inst.Arg].String
			val := stack[sp] // Keep on stack as result
			if isMapCtx {
				mapCtx.vars[name] = val.ToInterface()
			} else {
				ctx.Set(name, val.ToInterface())
			}

		case OpCall:
			nameIdx := inst.Arg & 0xFFFF
			numArgs := int(inst.Arg >> 16)
			name := consts[nameIdx].String

			args := make([]any, numArgs)
			for i := numArgs - 1; i >= 0; i-- {
				args[i] = stack[sp].ToInterface()
				sp--
			}

			if builtin, ok := builtins[name]; ok {
				res, err := builtin(args...)
				if err != nil { return nil, err }
				sp++
				stack[sp] = FromInterface(res)
			} else {
				return nil, fmt.Errorf("builtin function not found: %s", name)
			}
		}
	}

	if sp < 0 {
		return nil, nil
	}
	return stack[sp].ToInterface(), nil
}

func valToFloat64(v Value) (float64, bool) {
	switch v.Type {
	case ValFloat: return v.Float, true
	case ValInt: return float64(v.Int), true
	}
	return 0, false
}

func isValTruthy(v Value) bool {
	switch v.Type {
	case ValBool: return v.Bool
	case ValNil: return false
	default: return true
	}
}
