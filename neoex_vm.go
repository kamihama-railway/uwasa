// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"bytes"
	"fmt"
	"math"
	"sync"
)

var neoBufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

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
				val := vars[name]
				switch v := val.(type) {
				case int64: stack[sp] = Value{Type: ValInt, Num: uint64(v)}
				case float64: stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(v)}
				case string: stack[sp] = Value{Type: ValString, Str: v}
				case bool:
					if v { stack[sp] = Value{Type: ValBool, Num: 1} } else { stack[sp] = Value{Type: ValBool, Num: 0} }
				case int: stack[sp] = Value{Type: ValInt, Num: uint64(v)}
				default: stack[sp] = FromInterface(v)
				}
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
			rv := consts[cIdx]
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			if isMapCtx {
				val := vars[name]
				switch lv := val.(type) {
				case int64:
					if rv.Type == ValInt { stack[sp] = Value{Type: ValInt, Num: uint64(lv + int64(rv.Num))} } else if rv.Type == ValFloat { stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(float64(lv) + math.Float64frombits(rv.Num))} } else { stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(float64(lv) + 0)} } // fallback
					continue
				case float64:
					if rv.Type == ValInt { stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lv + float64(int64(rv.Num)))} } else if rv.Type == ValFloat { stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lv + math.Float64frombits(rv.Num))} } else { stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lv + 0)} }
					continue
				case string:
					if rv.Type == ValString { stack[sp] = Value{Type: ValString, Str: lv + rv.Str} } else { stack[sp] = FromInterface(lv) } // problematic
					continue
				}
				lval := FromInterface(val)
				if lval.Type == ValInt && rv.Type == ValInt { stack[sp] = Value{Type: ValInt, Num: lval.Num + rv.Num} } else { lf, _ := valToFloat64(lval); rf, _ := valToFloat64(rv); stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)} }
			} else {
				val, _ := ctx.Get(name)
				lv := FromInterface(val)
				if lv.Type == ValInt && rv.Type == ValInt {
					stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num}
				} else if lv.Type == ValString && rv.Type == ValString {
					stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str}
				} else {
					lf, _ := valToFloat64(lv); rf, _ := valToFloat64(rv)
					stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
				}
			}
		case NeoOpAddConstGlobal:
			cIdx := inst.Arg >> 16; gIdx := inst.Arg & 0xFFFF
			name := consts[gIdx].Str
			lv := consts[cIdx]
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			if isMapCtx {
				val := vars[name]
				switch rv := val.(type) {
				case int64:
					if lv.Type == ValInt { stack[sp] = Value{Type: ValInt, Num: uint64(int64(lv.Num) + rv)} } else if lv.Type == ValFloat { stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(math.Float64frombits(lv.Num) + float64(rv))} } else { stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(0 + float64(rv))} }
					continue
				case float64:
					if lv.Type == ValInt { stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(float64(int64(lv.Num)) + rv)} } else if lv.Type == ValFloat { stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(math.Float64frombits(lv.Num) + rv)} } else { stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(0 + rv)} }
					continue
				case string:
					if lv.Type == ValString { stack[sp] = Value{Type: ValString, Str: lv.Str + rv} } else { stack[sp] = FromInterface(rv) }
					continue
				}
				rval := FromInterface(val)
				if lv.Type == ValInt && rval.Type == ValInt { stack[sp] = Value{Type: ValInt, Num: lv.Num + rval.Num} } else { lf, _ := valToFloat64(lv); rf, _ := valToFloat64(rval); stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)} }
			} else {
				val, _ := ctx.Get(name)
				rv := FromInterface(val)
				if lv.Type == ValInt && rv.Type == ValInt {
					stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num}
				} else if lv.Type == ValString && rv.Type == ValString {
					stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str}
				} else {
					lf, _ := valToFloat64(lv); rf, _ := valToFloat64(rv)
					stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
				}
			}
		case NeoOpAddGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			if isMapCtx {
				v1 := vars[consts[g1Idx].Str]
				v2 := vars[consts[g2Idx].Str]

				done := false
				switch lv := v1.(type) {
				case int64:
					switch rv := v2.(type) {
					case int64: stack[sp] = Value{Type: ValInt, Num: uint64(lv + rv)}; done = true
					case float64: stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(float64(lv) + rv)}; done = true
					}
				case float64:
					switch rv := v2.(type) {
					case int64: stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lv + float64(rv))}; done = true
					case float64: stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lv + rv)}; done = true
					}
				case string:
					if rv, ok := v2.(string); ok { stack[sp] = Value{Type: ValString, Str: lv + rv}; done = true }
				}
				if done { continue }

				lv := FromInterface(v1)
				rv := FromInterface(v2)
				if lv.Type == ValInt && rv.Type == ValInt { stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num} } else if lv.Type == ValString && rv.Type == ValString { stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str} } else { lf, _ := valToFloat64(lv); rf, _ := valToFloat64(rv); stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)} }
			} else {
				v1, _ := ctx.Get(consts[g1Idx].Str)
				v2, _ := ctx.Get(consts[g2Idx].Str)
				lv := FromInterface(v1); rv := FromInterface(v2)
				if lv.Type == ValInt && rv.Type == ValInt {
					stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num}
				} else if lv.Type == ValString && rv.Type == ValString {
					stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str}
				} else {
					lf, _ := valToFloat64(lv); rf, _ := valToFloat64(rv)
					stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
				}
			}
		case NeoOpEqualGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
			r := consts[cIdx]
			res := false
			if isMapCtx {
				val := vars[consts[gIdx].Str]
				switch lv := val.(type) {
				case int64:
					if r.Type == ValInt { res = lv == int64(r.Num) } else if r.Type == ValFloat { res = float64(lv) == math.Float64frombits(r.Num) }
				case float64:
					if r.Type == ValInt { res = lv == float64(int64(r.Num)) } else if r.Type == ValFloat { res = lv == math.Float64frombits(r.Num) }
				case string:
					if r.Type == ValString { res = lv == r.Str }
				case bool:
					if r.Type == ValBool { res = (lv && r.Num != 0) || (!lv && r.Num == 0) }
				default:
					lval := FromInterface(val)
					if lval.Type == r.Type {
						switch lval.Type {
						case ValInt, ValFloat, ValBool: res = lval.Num == r.Num
						case ValString: res = lval.Str == r.Str
						case ValNil: res = true
						}
					} else {
						lf, okL := valToFloat64(lval); rf, okR := valToFloat64(r)
						if okL && okR { res = lf == rf }
					}
				}
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				lv := FromInterface(val)
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
			}
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpGreaterGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
			r := consts[cIdx]
			res := false
			if isMapCtx {
				val := vars[consts[gIdx].Str]
				switch lv := val.(type) {
				case int64:
					if r.Type == ValInt { res = lv > int64(r.Num) } else if r.Type == ValFloat { res = float64(lv) > math.Float64frombits(r.Num) }
				case float64:
					if r.Type == ValInt { res = lv > float64(int64(r.Num)) } else if r.Type == ValFloat { res = lv > math.Float64frombits(r.Num) }
				default:
					lval := FromInterface(val)
					if lval.Type == ValInt && r.Type == ValInt { res = int64(lval.Num) > int64(r.Num) } else { lf, _ := valToFloat64(lval); rf, _ := valToFloat64(r); res = lf > rf }
				}
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				lv := FromInterface(val)
				if lv.Type == ValInt && r.Type == ValInt { res = int64(lv.Num) > int64(r.Num) } else { lf, _ := valToFloat64(lv); rf, _ := valToFloat64(r); res = lf > rf }
			}
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpLessGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
			r := consts[cIdx]
			res := false
			if isMapCtx {
				val := vars[consts[gIdx].Str]
				switch lv := val.(type) {
				case int64:
					if r.Type == ValInt { res = lv < int64(r.Num) } else if r.Type == ValFloat { res = float64(lv) < math.Float64frombits(r.Num) }
				case float64:
					if r.Type == ValInt { res = lv < float64(int64(r.Num)) } else if r.Type == ValFloat { res = lv < math.Float64frombits(r.Num) }
				default:
					lval := FromInterface(val)
					if lval.Type == ValInt && r.Type == ValInt { res = int64(lval.Num) < int64(r.Num) } else { lf, _ := valToFloat64(lval); rf, _ := valToFloat64(r); res = lf < rf }
				}
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				lv := FromInterface(val)
				if lv.Type == ValInt && r.Type == ValInt { res = int64(lv.Num) < int64(r.Num) } else { lf, _ := valToFloat64(lv); rf, _ := valToFloat64(r); res = lf < rf }
			}
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpFusedCompareGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF
			cIdx := int(inst.Arg >> 12) & 0x3FF
			jTarget := int(inst.Arg) & 0xFFF
			r := consts[cIdx]
			res := false
			if isMapCtx {
				val := vars[consts[gIdx].Str]
				switch lv := val.(type) {
				case int64:
					if r.Type == ValInt { res = lv == int64(r.Num) } else if r.Type == ValFloat { res = float64(lv) == math.Float64frombits(r.Num) }
				case float64:
					if r.Type == ValInt { res = lv == float64(int64(r.Num)) } else if r.Type == ValFloat { res = lv == math.Float64frombits(r.Num) }
				case string:
					if r.Type == ValString { res = lv == r.Str }
				case bool:
					if r.Type == ValBool { res = (lv && r.Num != 0) || (!lv && r.Num == 0) }
				default:
					lval := FromInterface(val)
					if lval.Type == r.Type {
						switch lval.Type {
						case ValInt, ValFloat, ValBool: res = lval.Num == r.Num
						case ValString: res = lval.Str == r.Str
						case ValNil: res = true
						}
					} else {
						lf, okL := valToFloat64(lval); rf, okR := valToFloat64(r)
						if okL && okR { res = lf == rf }
					}
				}
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				lv := FromInterface(val)
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
			}
			if !res { pc = jTarget }
		case NeoOpGetGlobalJumpIfFalse:
			gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
			var truth bool
			if isMapCtx {
				val := vars[consts[gIdx].Str]
				switch v := val.(type) {
				case bool: truth = v
				case nil: truth = false
				default: truth = true
				}
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				truth = isValTruthy(FromInterface(val))
			}
			if !truth { pc = int(jTarget) }
		case NeoOpGetGlobalJumpIfTrue:
			gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
			var truth bool
			if isMapCtx {
				val := vars[consts[gIdx].Str]
				switch v := val.(type) {
				case bool: truth = v
				case nil: truth = false
				default: truth = true
				}
			} else {
				val, _ := ctx.Get(consts[gIdx].Str)
				truth = isValTruthy(FromInterface(val))
			}
			if truth { pc = int(jTarget) }
		case NeoOpConcat:
			numArgs := int(inst.Arg)
			if numArgs == 2 {
				r := stack[sp]; sp--; l := stack[sp]
				var s1, s2 string
				if l.Type == ValString { s1 = l.Str } else { s1 = fmt.Sprintf("%v", l.ToInterface()) }
				if r.Type == ValString { s2 = r.Str } else { s2 = fmt.Sprintf("%v", r.ToInterface()) }
				stack[sp] = Value{Type: ValString, Str: s1 + s2}
				continue
			}
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
			buf := neoBufferPool.Get().(*bytes.Buffer)
			buf.Reset(); buf.Grow(totalLen)
			for _, s := range argStrings { buf.WriteString(s) }
			res := buf.String(); neoBufferPool.Put(buf)
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
