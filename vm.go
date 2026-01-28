// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"math"
)

func RunVM(bc *RenderedBytecode, ctx Context) (any, error) {
	if bc == nil || len(bc.Instructions) == 0 {
		return nil, nil
	}

	if mapCtx, ok := ctx.(*MapContext); ok {
		return runVM(bc, ctx, mapCtx, true)
	}
	return runVM(bc, ctx, nil, false)
}

func runVM(bc *RenderedBytecode, ctx Context, mapCtx *MapContext, isMapCtx bool) (any, error) {
	var stack [64]Value
	sp := -1
	pc := 0
	insts := bc.Instructions
	consts := bc.Constants
	nInsts := len(insts)

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
				stack[sp] = Value{Type: ValInt, Num: uint64(int64(l.Num) + int64(r.Num))}
			} else if l.Type == ValString && r.Type == ValString {
				stack[sp] = Value{Type: ValString, Str: l.Str + r.Str}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}

		case OpSub:
			r := stack[sp]
			sp--
			l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: uint64(int64(l.Num) - int64(r.Num))}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf - rf)}
			}

		case OpMul:
			r := stack[sp]
			sp--
			l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: uint64(int64(l.Num) * int64(r.Num))}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf * rf)}
			}

		case OpDiv:
			r := stack[sp]
			sp--
			l := stack[sp]
			if r.Type == ValInt && r.Num == 0 { return nil, fmt.Errorf("division by zero") }
			if r.Type == ValFloat && math.Float64frombits(r.Num) == 0 { return nil, fmt.Errorf("division by zero") }
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: uint64(int64(l.Num) / int64(r.Num))}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf / rf)}
			}

		case OpMod:
			r := stack[sp]
			sp--
			l := stack[sp]
			if r.Type != ValInt || l.Type != ValInt {
				return nil, fmt.Errorf("modulo operator supports only integers")
			}
			if r.Num == 0 { return nil, fmt.Errorf("division by zero") }
			stack[sp] = Value{Type: ValInt, Num: uint64(int64(l.Num) % int64(r.Num))}

		case OpEqual:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == r.Type {
				switch l.Type {
				case ValInt, ValFloat, ValBool: res = l.Num == r.Num
				case ValString: res = l.Str == r.Str
				case ValNil: res = true
				}
			} else {
				lf, okL := valToFloat64(l)
				rf, okR := valToFloat64(r)
				if okL && okR {
					res = lf == rf
				}
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}

		case OpGreater:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) > int64(r.Num)
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf > rf
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}

		case OpLess:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) < int64(r.Num)
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf < rf
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}

		case OpGreaterEqual:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) >= int64(r.Num)
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf >= rf
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}

		case OpLessEqual:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) <= int64(r.Num)
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf <= rf
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}

		case OpAnd:
			r := stack[sp]
			sp--
			l := stack[sp]
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(isValTruthy(l) && isValTruthy(r))}

		case OpOr:
			r := stack[sp]
			sp--
			l := stack[sp]
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(isValTruthy(l) || isValTruthy(r))}

		case OpNot:
			l := stack[sp]
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(!isValTruthy(l))}

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
			name := consts[inst.Arg].Str
			var val any
			if isMapCtx {
				val = mapCtx.vars[name]
			} else {
				val, _ = ctx.Get(name)
			}
			sp++
			stack[sp] = FromInterface(val)

		case OpSetGlobal:
			name := consts[inst.Arg].Str
			val := stack[sp]
			if isMapCtx {
				mapCtx.vars[name] = val.ToInterface()
			} else {
				ctx.Set(name, val.ToInterface())
			}

		case OpCall:
			nameIdx := inst.Arg & 0xFFFF
			numArgs := int(inst.Arg >> 16)
			name := consts[nameIdx].Str

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

		case OpEqualConst:
			r := consts[inst.Arg]
			l := stack[sp]
			res := false
			if l.Type == r.Type {
				switch l.Type {
				case ValInt, ValFloat, ValBool: res = l.Num == r.Num
				case ValString: res = l.Str == r.Str
				case ValNil: res = true
				}
			} else {
				lf, okL := valToFloat64(l)
				rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}

		case OpAddGlobal:
			gIdx := inst.Arg & 0xFFFF
			cIdx := inst.Arg >> 16
			name := consts[gIdx].Str
			var l any
			if isMapCtx {
				l = mapCtx.vars[name]
			} else {
				l, _ = ctx.Get(name)
			}
			lv := FromInterface(l)
			rv := consts[cIdx]
			sp++
			if lv.Type == ValInt && rv.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: uint64(int64(lv.Num) + int64(rv.Num))}
			} else {
				lf, _ := valToFloat64(lv)
				rf, _ := valToFloat64(rv)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}

		case OpAddGlobalGlobal:
			g1Idx := inst.Arg >> 16
			g2Idx := inst.Arg & 0xFFFF
			name1 := consts[g1Idx].Str
			name2 := consts[g2Idx].Str
			var v1, v2 any
			if isMapCtx {
				v1 = mapCtx.vars[name1]
				v2 = mapCtx.vars[name2]
			} else {
				v1, _ = ctx.Get(name1)
				v2, _ = ctx.Get(name2)
			}
			lv := FromInterface(v1)
			rv := FromInterface(v2)
			sp++
			if lv.Type == ValInt && rv.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: uint64(int64(lv.Num) + int64(rv.Num))}
			} else {
				lf, _ := valToFloat64(lv)
				rf, _ := valToFloat64(rv)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}

		case OpEqualGlobalConst:
			gIdx := inst.Arg >> 16
			cIdx := inst.Arg & 0xFFFF
			name := consts[gIdx].Str
			var l any
			if isMapCtx {
				l = mapCtx.vars[name]
			} else {
				l, _ = ctx.Get(name)
			}
			lv := FromInterface(l)
			r := consts[cIdx]
			res := false
			if lv.Type == r.Type {
				switch lv.Type {
				case ValInt, ValFloat, ValBool: res = lv.Num == r.Num
				case ValString: res = lv.Str == r.Str
				case ValNil: res = true
				}
			} else {
				lf, okL := valToFloat64(lv)
				rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			sp++
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}

		case OpGreaterGlobalConst:
			gIdx := inst.Arg >> 16
			cIdx := inst.Arg & 0xFFFF
			name := consts[gIdx].Str
			var l any
			if isMapCtx {
				l = mapCtx.vars[name]
			} else {
				l, _ = ctx.Get(name)
			}
			lv := FromInterface(l)
			r := consts[cIdx]
			res := false
			if lv.Type == ValInt && r.Type == ValInt {
				res = int64(lv.Num) > int64(r.Num)
			} else {
				lf, _ := valToFloat64(lv)
				rf, _ := valToFloat64(r)
				res = lf > rf
			}
			sp++
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}

		case OpLessGlobalConst:
			gIdx := inst.Arg >> 16
			cIdx := inst.Arg & 0xFFFF
			name := consts[gIdx].Str
			var l any
			if isMapCtx {
				l = mapCtx.vars[name]
			} else {
				l, _ = ctx.Get(name)
			}
			lv := FromInterface(l)
			r := consts[cIdx]
			res := false
			if lv.Type == ValInt && r.Type == ValInt {
				res = int64(lv.Num) < int64(r.Num)
			} else {
				lf, _ := valToFloat64(lv)
				rf, _ := valToFloat64(r)
				res = lf < rf
			}
			sp++
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}

		case OpFusedCompareGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF
			cIdx := int(inst.Arg >> 12) & 0x3FF
			jTarget := int(inst.Arg) & 0xFFF
			name := consts[gIdx].Str
			var l any
			if isMapCtx {
				l = mapCtx.vars[name]
			} else {
				l, _ = ctx.Get(name)
			}
			lv := FromInterface(l)
			r := consts[cIdx]
			res := false
			if lv.Type == r.Type {
				switch lv.Type {
				case ValInt, ValFloat, ValBool: res = lv.Num == r.Num
				case ValString: res = lv.Str == r.Str
				case ValNil: res = true
				}
			} else {
				lf, okL := valToFloat64(lv)
				rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			if !res {
				pc = jTarget
			}

		case OpGetGlobalJumpIfFalse:
			gIdx := inst.Arg >> 16
			jTarget := inst.Arg & 0xFFFF
			name := consts[gIdx].Str
			var val any
			if isMapCtx {
				val = mapCtx.vars[name]
			} else {
				val, _ = ctx.Get(name)
			}
			if !isValTruthy(FromInterface(val)) {
				pc = int(jTarget)
			}

		case OpGetGlobalJumpIfTrue:
			gIdx := inst.Arg >> 16
			jTarget := inst.Arg & 0xFFFF
			name := consts[gIdx].Str
			var val any
			if isMapCtx {
				val = mapCtx.vars[name]
			} else {
				val, _ = ctx.Get(name)
			}
			if isValTruthy(FromInterface(val)) {
				pc = int(jTarget)
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
	case ValFloat: return math.Float64frombits(v.Num), true
	case ValInt: return float64(int64(v.Num)), true
	}
	return 0, false
}

func isValTruthy(v Value) bool {
	switch v.Type {
	case ValBool: return v.Num != 0
	case ValNil: return false
	default: return true
	}
}

func boolToUint64(b bool) uint64 {
	if b { return 1 }
	return 0
}
