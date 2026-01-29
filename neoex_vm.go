// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"bytes"
	"fmt"
	"math"
)

func RunNeoVM[C Context](bc *NeoBytecode, ctx C) (any, error) {
	if bc == nil || len(bc.Instructions) == 0 {
		return nil, nil
	}

	var stack [64]Value
	sp := -1
	pc := 0
	insts := bc.Instructions
	consts := bc.Constants
	nInsts := len(insts)

	// Try to get MapContext for fast path
	mapCtx, isMapCtx := any(ctx).(*MapContext)
	var vars map[string]any
	if isMapCtx {
		vars = mapCtx.vars
	}

	for pc < nInsts {
		inst := insts[pc]
		pc++

		switch inst.Op {
		case NeoOpPush:
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = consts[inst.Arg]
		case NeoOpPop:
			sp--
		case NeoOpAdd:
			r := stack[sp]; sp--; l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: l.Num + r.Num}
			} else if l.Type == ValString && r.Type == ValString {
				stack[sp] = Value{Type: ValString, Str: l.Str + r.Str}
			} else if l.Type == ValFloat && r.Type == ValFloat {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(math.Float64frombits(l.Num) + math.Float64frombits(r.Num))}
			} else if l.Type == ValInt && r.Type == ValFloat {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(float64(int64(l.Num)) + math.Float64frombits(r.Num))}
			} else if l.Type == ValFloat && r.Type == ValInt {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(math.Float64frombits(l.Num) + float64(int64(r.Num)))}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case NeoOpSub:
			r := stack[sp]; sp--; l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: l.Num - r.Num}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf - rf)}
			}
		case NeoOpMul:
			r := stack[sp]; sp--; l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: l.Num * r.Num}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf * rf)}
			}
		case NeoOpDiv:
			r := stack[sp]; sp--; l := stack[sp]
			if r.Type == ValInt && r.Num == 0 { return nil, fmt.Errorf("division by zero") }
			if r.Type == ValFloat && math.Float64frombits(r.Num) == 0 { return nil, fmt.Errorf("division by zero") }
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: l.Num / r.Num}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf / rf)}
			}
		case NeoOpMod:
			r := stack[sp]; sp--; l := stack[sp]
			if r.Type != ValInt { return nil, fmt.Errorf("modulo operator supports only integers") }
			if r.Num == 0 { return nil, fmt.Errorf("division by zero") }
			stack[sp] = Value{Type: ValInt, Num: l.Num % r.Num}
		case NeoOpEqual:
			r := stack[sp]; sp--; l := stack[sp]
			res := false
			if l.Type == r.Type {
				switch l.Type {
				case ValInt, ValFloat, ValBool: res = l.Num == r.Num
				case ValString: res = l.Str == r.Str
				case ValNil: res = true
				}
			} else {
				lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpGreater:
			r := stack[sp]; sp--; l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) > int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf > rf
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpLess:
			r := stack[sp]; sp--; l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) < int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf < rf
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpGreaterEqual:
			r := stack[sp]; sp--; l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) >= int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf >= rf
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpLessEqual:
			r := stack[sp]; sp--; l := stack[sp]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) <= int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf <= rf
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpAnd:
			r := stack[sp]; sp--; l := stack[sp]
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(isValTruthy(l) && isValTruthy(r))}
		case NeoOpOr:
			r := stack[sp]; sp--; l := stack[sp]
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(isValTruthy(l) || isValTruthy(r))}
		case NeoOpNot:
			l := stack[sp]
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(!isValTruthy(l))}
		case NeoOpJump:
			pc = int(inst.Arg)
		case NeoOpJumpIfFalse:
			l := stack[sp]; sp--
			if !isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpJumpIfTrue:
			l := stack[sp]; sp--
			if isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpGetGlobal:
			name := consts[inst.Arg].Str
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			if isMapCtx {
				stack[sp] = FromInterface(vars[name])
			} else {
				val, _ := ctx.Get(name)
				stack[sp] = FromInterface(val)
			}
		case NeoOpSetGlobal:
			name := consts[inst.Arg].Str
			val := stack[sp]
			if isMapCtx {
				vars[name] = val.ToInterface()
			} else {
				ctx.Set(name, val.ToInterface())
			}
		case NeoOpCall:
			nameIdx := inst.Arg & 0xFFFF
			numArgs := int(inst.Arg >> 16)
			name := consts[nameIdx].Str

			var argsBuf [8]any
			var args []any
			if numArgs <= 8 {
				args = argsBuf[:numArgs]
			} else {
				args = make([]any, numArgs)
			}
			for i := numArgs - 1; i >= 0; i-- {
				args[i] = stack[sp].ToInterface(); sp--
			}
			if builtin, ok := builtins[name]; ok {
				res, err := builtin(args...)
				if err != nil { return nil, err }
				sp++
				if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
				stack[sp] = FromInterface(res)
			} else {
				return nil, fmt.Errorf("builtin function not found: %s", name)
			}
		case NeoOpEqualConst:
			r := consts[inst.Arg]; l := stack[sp]
			res := false
			if l.Type == r.Type {
				switch l.Type {
				case ValInt, ValFloat, ValBool: res = l.Num == r.Num
				case ValString: res = l.Str == r.Str
				case ValNil: res = true
				}
			} else {
				lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpAddGlobal:
			gIdx := inst.Arg & 0xFFFF; cIdx := inst.Arg >> 16
			name := consts[gIdx].Str
			var lv Value
			if isMapCtx {
				lv = FromInterface(vars[name])
			} else {
				val, _ := ctx.Get(name)
				lv = FromInterface(val)
			}
			rv := consts[cIdx]
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			if lv.Type == ValInt && rv.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num}
			} else if lv.Type == ValString && rv.Type == ValString {
				stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str}
			} else if lv.Type == ValFloat && rv.Type == ValFloat {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(math.Float64frombits(lv.Num) + math.Float64frombits(rv.Num))}
			} else if lv.Type == ValInt && rv.Type == ValFloat {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(float64(int64(lv.Num)) + math.Float64frombits(rv.Num))}
			} else if lv.Type == ValFloat && rv.Type == ValInt {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(math.Float64frombits(lv.Num) + float64(int64(rv.Num)))}
			} else {
				lf, _ := valToFloat64(lv); rf, _ := valToFloat64(rv)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case NeoOpAddConstGlobal:
			cIdx := inst.Arg >> 16; gIdx := inst.Arg & 0xFFFF
			name := consts[gIdx].Str
			var rv Value
			if isMapCtx {
				rv = FromInterface(vars[name])
			} else {
				val, _ := ctx.Get(name)
				rv = FromInterface(val)
			}
			lv := consts[cIdx]
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			if lv.Type == ValInt && rv.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num}
			} else if lv.Type == ValString && rv.Type == ValString {
				stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str}
			} else if lv.Type == ValFloat && rv.Type == ValFloat {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(math.Float64frombits(lv.Num) + math.Float64frombits(rv.Num))}
			} else if lv.Type == ValInt && rv.Type == ValFloat {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(float64(int64(lv.Num)) + math.Float64frombits(rv.Num))}
			} else if lv.Type == ValFloat && rv.Type == ValInt {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(math.Float64frombits(lv.Num) + float64(int64(rv.Num)))}
			} else {
				lf, _ := valToFloat64(lv); rf, _ := valToFloat64(rv)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case NeoOpAddGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF
			var lv, rv Value
			if isMapCtx {
				lv = FromInterface(vars[consts[g1Idx].Str])
				rv = FromInterface(vars[consts[g2Idx].Str])
			} else {
				v1, _ := ctx.Get(consts[g1Idx].Str)
				v2, _ := ctx.Get(consts[g2Idx].Str)
				lv = FromInterface(v1); rv = FromInterface(v2)
			}
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			if lv.Type == ValInt && rv.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num}
			} else if lv.Type == ValString && rv.Type == ValString {
				stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str}
			} else if lv.Type == ValFloat && rv.Type == ValFloat {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(math.Float64frombits(lv.Num) + math.Float64frombits(rv.Num))}
			} else if lv.Type == ValInt && rv.Type == ValFloat {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(float64(int64(lv.Num)) + math.Float64frombits(rv.Num))}
			} else if lv.Type == ValFloat && rv.Type == ValInt {
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(math.Float64frombits(lv.Num) + float64(int64(rv.Num)))}
			} else {
				lf, _ := valToFloat64(lv); rf, _ := valToFloat64(rv)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case NeoOpEqualGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
			var lv Value
			if isMapCtx {
				lv = FromInterface(vars[consts[gIdx].Str])
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				lv = FromInterface(val)
			}
			r := consts[cIdx]
			res := false
			if lv.Type == r.Type {
				switch lv.Type {
				case ValInt, ValFloat, ValBool: res = lv.Num == r.Num
				case ValString: res = lv.Str == r.Str
				case ValNil: res = true
				}
			} else {
				lf, okL := valToFloat64(lv); rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpGreaterGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
			var lv Value
			if isMapCtx {
				lv = FromInterface(vars[consts[gIdx].Str])
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				lv = FromInterface(val)
			}
			r := consts[cIdx]
			res := false
			if lv.Type == ValInt && r.Type == ValInt {
				res = int64(lv.Num) > int64(r.Num)
			} else {
				lf, _ := valToFloat64(lv); rf, _ := valToFloat64(r)
				res = lf > rf
			}
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpLessGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
			var lv Value
			if isMapCtx {
				lv = FromInterface(vars[consts[gIdx].Str])
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				lv = FromInterface(val)
			}
			r := consts[cIdx]
			res := false
			if lv.Type == ValInt && r.Type == ValInt {
				res = int64(lv.Num) < int64(r.Num)
			} else {
				lf, _ := valToFloat64(lv); rf, _ := valToFloat64(r)
				res = lf < rf
			}
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpFusedCompareGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF
			cIdx := int(inst.Arg >> 12) & 0x3FF
			jTarget := int(inst.Arg) & 0xFFF
			var lv Value
			if isMapCtx {
				lv = FromInterface(vars[consts[gIdx].Str])
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				lv = FromInterface(val)
			}
			r := consts[cIdx]
			res := false
			if lv.Type == r.Type {
				switch lv.Type {
				case ValInt, ValFloat, ValBool: res = lv.Num == r.Num
				case ValString: res = lv.Str == r.Str
				case ValNil: res = true
				}
			} else {
				lf, okL := valToFloat64(lv); rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			if !res { pc = jTarget }
		case NeoOpGetGlobalJumpIfFalse:
			gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
			var lv Value
			if isMapCtx {
				lv = FromInterface(vars[consts[gIdx].Str])
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				lv = FromInterface(val)
			}
			if !isValTruthy(lv) { pc = int(jTarget) }
		case NeoOpGetGlobalJumpIfTrue:
			gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
			var lv Value
			if isMapCtx {
				lv = FromInterface(vars[consts[gIdx].Str])
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				lv = FromInterface(val)
			}
			if isValTruthy(lv) { pc = int(jTarget) }
		case NeoOpConcat:
			numArgs := int(inst.Arg)
			totalLen := 0
			var argStringsBuf [8]string
			var argStrings []string
			if numArgs <= 8 { argStrings = argStringsBuf[:numArgs] } else { argStrings = make([]string, numArgs) }
			for i := numArgs - 1; i >= 0; i-- {
				v := stack[sp]; sp--; var s string
				switch v.Type {
				case ValString: s = v.Str
				case ValInt: s = fmt.Sprintf("%d", int64(v.Num))
				case ValFloat: s = fmt.Sprintf("%g", math.Float64frombits(v.Num))
				case ValBool:
					if v.Num != 0 { s = "true" } else { s = "false" }
				default: s = fmt.Sprintf("%v", v.ToInterface())
				}
				argStrings[i] = s; totalLen += len(s)
			}
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset(); buf.Grow(totalLen)
			for _, s := range argStrings { buf.WriteString(s) }
			res := buf.String(); bufferPool.Put(buf)
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = Value{Type: ValString, Str: res}
		case NeoOpReturn:
			if sp < 0 { return nil, nil }
			return stack[sp].ToInterface(), nil
		}
	}
	if sp < 0 { return nil, nil }
	return stack[sp].ToInterface(), nil
}
