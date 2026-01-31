// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"bytes"
	"fmt"
	"math"
	"sync"
	"unsafe"
)

var neoBufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func RunNeoVM[C Context](bc *NeoBytecode, ctx C) (any, error) {
	if bc == nil || len(bc.Instructions) == 0 { return nil, nil }
	if mctx, ok := any(ctx).(*MapContext); ok { return RunNeoVMWithMap(bc, mctx.vars) }
	return runNeoVMGeneral(bc, ctx)
}

func RunNeoVMWithMap(bc *NeoBytecode, vars map[string]any) (any, error) {
	if vars == nil { vars = make(map[string]any) }
	var stack [64]Value
	insts := bc.Instructions
	nInsts := len(insts)
	if nInsts == 0 { return nil, nil }

	pInsts := unsafe.SliceData(insts)
	pConsts := unsafe.SliceData(bc.Constants)

	sp := -1
	pc := 0

	const valSize = unsafe.Sizeof(Value{})
	const instSize = unsafe.Sizeof(neoInstruction{})

	for pc < nInsts {
		inst := (*neoInstruction)(unsafe.Add(unsafe.Pointer(pInsts), uintptr(pc)*instSize))
		pc++

		switch inst.Op {
		case NeoOpPush:
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = *(*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
		case NeoOpPop: sp--
		case NeoOpAdd:
			r := stack[sp]; sp--; l := &stack[sp]
			if l.Type == ValInt && r.Type == ValInt { l.Num += r.Num } else if l.Type == ValString && r.Type == ValString { l.Str += r.Str } else { *l = l.Add(r) }
		case NeoOpSub:
			r := stack[sp]; sp--; l := &stack[sp]
			if l.Type == ValInt && r.Type == ValInt { l.Num -= r.Num } else { *l = l.Sub(r) }
		case NeoOpMul:
			r := stack[sp]; sp--; l := &stack[sp]
			if l.Type == ValInt && r.Type == ValInt { l.Num *= r.Num } else { *l = l.Mul(r) }
		case NeoOpDiv:
			rv := stack[sp]; sp--; l := &stack[sp]
			res, err := l.DivErr(rv); if err != nil { return nil, err }; *l = res
		case NeoOpEqual:
			rv := stack[sp]; sp--; l := &stack[sp]
			*l = Value{Type: ValBool, Num: boolToUint64(l.Equal(rv))}
		case NeoOpGreater:
			rv := stack[sp]; sp--; l := &stack[sp]
			*l = Value{Type: ValBool, Num: boolToUint64(l.Greater(rv))}
		case NeoOpLess:
			rv := stack[sp]; sp--; l := &stack[sp]
			*l = Value{Type: ValBool, Num: boolToUint64(rv.Greater(*l))}
		case NeoOpJump: pc = int(inst.Arg)
		case NeoOpJumpIfFalse:
			l := stack[sp]; sp--
			if !isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpJumpIfTrue:
			l := stack[sp]; sp--
			if isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpGetGlobal:
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize)).Str
			val := vars[name]
			target := &stack[sp]
			switch v := val.(type) {
			case int64: *target = Value{Type: ValInt, Num: uint64(v)}
			case int: *target = Value{Type: ValInt, Num: uint64(int64(v))}
			case float64: *target = Value{Type: ValFloat, Num: math.Float64bits(v)}
			case string: *target = Value{Type: ValString, Str: v}
			case bool: *target = Value{Type: ValBool, Num: boolToUint64(v)}
			case nil: *target = Value{Type: ValNil}
			default: *target = FromInterface(v)
			}
		case NeoOpSetGlobal:
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize)).Str
			vars[name] = stack[sp].ToInterface()
		case NeoOpEqualConst, NeoOpEqualC:
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
			l := &stack[sp]
			*l = Value{Type: ValBool, Num: boolToUint64(l.Equal(*cv))}
		case NeoOpGreaterC:
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
			l := &stack[sp]
			*l = Value{Type: ValBool, Num: boolToUint64(l.Greater(*cv))}
		case NeoOpLessC:
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
			l := &stack[sp]
			*l = Value{Type: ValBool, Num: boolToUint64(cv.Greater(*l))}
		case NeoOpEqualGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			val := vars[name]
			res := false
			switch v := val.(type) {
			case int64: res = cv.Type == ValInt && v == int64(cv.Num)
			case float64: res = cv.Type == ValFloat && v == math.Float64frombits(cv.Num)
			case string: res = cv.Type == ValString && v == cv.Str
			default: res = EqualAny(val, cv.ToInterface())
			}
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpAddGlobal, NeoOpAddGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			val := vars[name]
			target := &stack[sp]
			switch v := val.(type) {
			case int64:
				if cv.Type == ValInt { *target = Value{Type: ValInt, Num: uint64(v + int64(cv.Num))} } else if cv.Type == ValFloat { *target = Value{Type: ValFloat, Num: math.Float64bits(float64(v) + math.Float64frombits(cv.Num))} } else { *target = AddAny(v, cv.ToInterface()) }
			case float64:
				if cv.Type == ValInt { *target = Value{Type: ValFloat, Num: math.Float64bits(v + float64(int64(cv.Num)))} } else if cv.Type == ValFloat { *target = Value{Type: ValFloat, Num: math.Float64bits(v + math.Float64frombits(cv.Num))} } else { *target = AddAny(v, cv.ToInterface()) }
			case string:
				if cv.Type == ValString { *target = Value{Type: ValString, Str: v + cv.Str} } else { *target = AddAny(v, cv.ToInterface()) }
			default: *target = AddAny(v, cv.ToInterface())
			}
		case NeoOpAddConstGlobal:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			stack[sp] = AddAny(cv.ToInterface(), vars[name])
		case NeoOpSubGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			stack[sp] = SubAny(vars[name], cv.ToInterface())
		case NeoOpMulGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			stack[sp] = MulAny(vars[name], cv.ToInterface())
		case NeoOpDivGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			stack[sp] = DivAny(vars[name], cv.ToInterface())
		case NeoOpSubCG:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			stack[sp] = SubAny(cv.ToInterface(), vars[name])
		case NeoOpMulCG:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			stack[sp] = MulAny(cv.ToInterface(), vars[name])
		case NeoOpDivCG:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			stack[sp] = DivAny(cv.ToInterface(), vars[name])
		case NeoOpGreaterGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			val := vars[name]
			res := false
			switch v := val.(type) {
			case int64:
				if cv.Type == ValInt { res = v > int64(cv.Num) } else if cv.Type == ValFloat { res = float64(v) > math.Float64frombits(cv.Num) } else { res = GreaterAny(v, cv.ToInterface()) }
			case float64:
				if cv.Type == ValInt { res = v > float64(int64(cv.Num)) } else if cv.Type == ValFloat { res = v > math.Float64frombits(cv.Num) } else { res = GreaterAny(v, cv.ToInterface()) }
			default: res = GreaterAny(val, cv.ToInterface())
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpLessGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			val := vars[name]
			res := false
			switch v := val.(type) {
			case int64:
				if cv.Type == ValInt { res = v < int64(cv.Num) } else if cv.Type == ValFloat { res = float64(v) < math.Float64frombits(cv.Num) } else { res = LessAny(v, cv.ToInterface()) }
			case float64:
				if cv.Type == ValInt { res = v < float64(int64(cv.Num)) } else if cv.Type == ValFloat { res = v < math.Float64frombits(cv.Num) } else { res = LessAny(v, cv.ToInterface()) }
			default: res = LessAny(val, cv.ToInterface())
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpAddGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			n1 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g1Idx)*valSize)).Str
			n2 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g2Idx)*valSize)).Str
			v1 := vars[n1]; v2 := vars[n2]
			if i1, ok1 := v1.(int64); ok1 {
				if i2, ok2 := v2.(int64); ok2 { stack[sp] = Value{Type: ValInt, Num: uint64(i1 + i2)}; continue }
			}
			stack[sp] = AddAny(v1, v2)
		case NeoOpSubGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			n1 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g1Idx)*valSize)).Str
			n2 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g2Idx)*valSize)).Str
			v1 := vars[n1]; v2 := vars[n2]
			if i1, ok1 := v1.(int64); ok1 {
				if i2, ok2 := v2.(int64); ok2 { stack[sp] = Value{Type: ValInt, Num: uint64(i1 - i2)}; continue }
			}
			stack[sp] = SubAny(v1, v2)
		case NeoOpMulGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			n1 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g1Idx)*valSize)).Str
			n2 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g2Idx)*valSize)).Str
			v1 := vars[n1]; v2 := vars[n2]
			if i1, ok1 := v1.(int64); ok1 {
				if i2, ok2 := v2.(int64); ok2 { stack[sp] = Value{Type: ValInt, Num: uint64(i1 * i2)}; continue }
			}
			stack[sp] = MulAny(v1, v2)
		case NeoOpFusedCompareGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			val := vars[name]; res := false
			switch v := val.(type) {
			case int64: res = cv.Type == ValInt && v == int64(cv.Num)
			case float64: res = cv.Type == ValFloat && v == math.Float64frombits(cv.Num)
			case string: res = cv.Type == ValString && v == cv.Str
			default: res = EqualAny(val, cv.ToInterface())
			}
			if !res { pc = jTarget }
		case NeoOpFusedGreaterGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			val := vars[name]; res := false
			switch v := val.(type) {
			case int64:
				if cv.Type == ValInt { res = v > int64(cv.Num) } else if cv.Type == ValFloat { res = float64(v) > math.Float64frombits(cv.Num) } else { res = GreaterAny(v, cv.ToInterface()) }
			case float64:
				if cv.Type == ValInt { res = v > float64(int64(cv.Num)) } else if cv.Type == ValFloat { res = v > math.Float64frombits(cv.Num) } else { res = GreaterAny(v, cv.ToInterface()) }
			default: res = GreaterAny(val, cv.ToInterface())
			}
			if !res { pc = jTarget }
		case NeoOpFusedLessGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			val := vars[name]; res := false
			switch v := val.(type) {
			case int64:
				if cv.Type == ValInt { res = v < int64(cv.Num) } else if cv.Type == ValFloat { res = float64(v) < math.Float64frombits(cv.Num) } else { res = LessAny(v, cv.ToInterface()) }
			case float64:
				if cv.Type == ValInt { res = v < float64(int64(cv.Num)) } else if cv.Type == ValFloat { res = v < math.Float64frombits(cv.Num) } else { res = LessAny(v, cv.ToInterface()) }
			default: res = LessAny(val, cv.ToInterface())
			}
			if !res { pc = jTarget }
		case NeoOpGetGlobalJumpIfFalse:
			gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			if !isTruthy(vars[name]) { pc = int(jTarget) }
		case NeoOpGetGlobalJumpIfTrue:
			gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			if isTruthy(vars[name]) { pc = int(jTarget) }
		case NeoOpAddC:
			l := &stack[sp]
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
			if l.Type == ValInt && cv.Type == ValInt { l.Num += cv.Num } else { *l = l.Add(*cv) }
		case NeoOpSubC:
			l := &stack[sp]
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
			if l.Type == ValInt && cv.Type == ValInt { l.Num -= cv.Num } else { *l = l.Sub(*cv) }
		case NeoOpMulC:
			l := &stack[sp]
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
			if l.Type == ValInt && cv.Type == ValInt { l.Num *= cv.Num } else { *l = l.Mul(*cv) }
		case NeoOpDivC:
			l := &stack[sp]
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
			*l = l.Div(*cv)
		case NeoOpAddInt:
			r := stack[sp]; sp--; l := &stack[sp]
			l.Num += r.Num
		case NeoOpSubInt:
			r := stack[sp]; sp--; l := &stack[sp]
			l.Num -= r.Num
		case NeoOpMulInt:
			r := stack[sp]; sp--; l := &stack[sp]
			l.Num *= r.Num
		case NeoOpConcat:
			numArgs := int(inst.Arg); totalLen := 0; var argStringsBuf [8]string; var argStrings []string
			if numArgs <= 8 { argStrings = argStringsBuf[:numArgs] } else { argStrings = make([]string, numArgs) }
			for i := numArgs - 1; i >= 0; i-- {
				v := stack[sp]; sp--
				var s string
				switch v.Type {
				case ValString: s = v.Str
				case ValInt: s = fmt.Sprintf("%d", int64(v.Num))
				case ValFloat: s = fmt.Sprintf("%g", math.Float64frombits(v.Num))
				case ValBool: if v.Num != 0 { s = "true" } else { s = "false" }
				default: s = fmt.Sprintf("%v", v.ToInterface())
				}
				argStrings[i] = s; totalLen += len(s)
			}
			buf := neoBufferPool.Get().(*bytes.Buffer); buf.Reset(); buf.Grow(totalLen)
			for _, s := range argStrings { buf.WriteString(s) }
			res := buf.String(); neoBufferPool.Put(buf)
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = Value{Type: ValString, Str: res}
		case NeoOpConcat2:
			r := stack[sp]; sp--; l := &stack[sp]
			var s1, s2 string
			if l.Type == ValString { s1 = l.Str } else { s1 = fmt.Sprintf("%v", l.ToInterface()) }
			if r.Type == ValString { s2 = r.Str } else { s2 = fmt.Sprintf("%v", r.ToInterface()) }
			*l = Value{Type: ValString, Str: s1 + s2}
		case NeoOpConcatGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			lv := vars[name]; var s1, s2 string
			if s, ok := lv.(string); ok { s1 = s } else { s1 = fmt.Sprintf("%v", lv) }
			if cv.Type == ValString { s2 = cv.Str } else { s2 = fmt.Sprintf("%v", cv.ToInterface()) }
			stack[sp] = Value{Type: ValString, Str: s1 + s2}
		case NeoOpConcatCG:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
			rv := vars[name]; var s1, s2 string
			if cv.Type == ValString { s1 = cv.Str } else { s1 = fmt.Sprintf("%v", cv.ToInterface()) }
			if s, ok := rv.(string); ok { s2 = s } else { s2 = fmt.Sprintf("%v", rv) }
			stack[sp] = Value{Type: ValString, Str: s1 + s2}
		case NeoOpCall:
			nameIdx := inst.Arg & 0xFFFF; numArgs := int(inst.Arg >> 16)
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(nameIdx)*valSize)).Str
			var argsBuf [8]any; var args []any
			if numArgs <= 8 { args = argsBuf[:numArgs] } else { args = make([]any, numArgs) }
			for i := numArgs - 1; i >= 0; i-- {
				args[i] = stack[sp].ToInterface(); sp--
			}
			if builtin, ok := builtins[name]; ok {
				res, err := builtin(args...); if err != nil { return nil, err }
				sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
				stack[sp] = FromInterface(res)
			} else { return nil, fmt.Errorf("builtin function not found: %s", name) }
		case NeoOpReturn:
			if sp < 0 { return nil, nil }
			return stack[sp].ToInterface(), nil
		default:
			res, err := runNeoVMWithMapFallback(inst, &stack, &sp, &pc, pConsts, vars)
			if err != nil { return nil, err }
			if inst.Op == NeoOpReturn { return res, nil }
		}
	}
	if sp < 0 { return nil, nil }
	return stack[sp].ToInterface(), nil
}

func runNeoVMWithMapFallback(inst *neoInstruction, stack *[64]Value, sp *int, pc *int, pConsts *Value, vars map[string]any) (any, error) {
	const valSize = unsafe.Sizeof(Value{})
	switch inst.Op {
	case NeoOpGreaterEqual:
		rv := stack[*sp]; *sp--; l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(l.Greater(rv) || l.Equal(rv))}
	case NeoOpLessEqual:
		rv := stack[*sp]; *sp--; l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(rv.Greater(*l) || l.Equal(rv))}
	case NeoOpAnd:
		rv := stack[*sp]; *sp--; l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(isValTruthy(*l) && isValTruthy(rv))}
	case NeoOpOr:
		rv := stack[*sp]; *sp--; l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(isValTruthy(*l) || isValTruthy(rv))}
	case NeoOpNot:
		l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(!isValTruthy(*l))}
	case NeoOpMod:
		rv := stack[*sp]; *sp--; l := &stack[*sp]
		res, err := l.ModErr(rv); if err != nil { return nil, err }; *l = res
	}
	return nil, nil
}

func runNeoVMGeneral(bc *NeoBytecode, ctx Context) (any, error) {
	var stack [64]Value
	insts := bc.Instructions
	nInsts := len(insts)
	if nInsts == 0 { return nil, nil }

	pInsts := unsafe.SliceData(insts)
	pConsts := unsafe.SliceData(bc.Constants)

	sp := -1
	pc := 0

	const valSize = unsafe.Sizeof(Value{})
	const instSize = unsafe.Sizeof(neoInstruction{})

	for pc < nInsts {
		inst := (*neoInstruction)(unsafe.Add(unsafe.Pointer(pInsts), uintptr(pc)*instSize))
		pc++

		switch inst.Op {
		case NeoOpPush:
			sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = *(*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
		case NeoOpPop: sp--
		case NeoOpAdd:
			r := stack[sp]; sp--; l := &stack[sp]
			*l = l.Add(r)
		case NeoOpSub:
			r := stack[sp]; sp--; l := &stack[sp]
			*l = l.Sub(r)
		case NeoOpMul:
			r := stack[sp]; sp--; l := &stack[sp]
			*l = l.Mul(r)
		case NeoOpDiv:
			rv := stack[sp]; sp--; l := &stack[sp]
			*l = l.Div(rv)
		case NeoOpEqual:
			rv := stack[sp]; sp--; l := &stack[sp]
			*l = Value{Type: ValBool, Num: boolToUint64(l.Equal(rv))}
		case NeoOpGreater:
			rv := stack[sp]; sp--; l := &stack[sp]
			*l = Value{Type: ValBool, Num: boolToUint64(l.Greater(rv))}
		case NeoOpLess:
			rv := stack[sp]; sp--; l := &stack[sp]
			*l = Value{Type: ValBool, Num: boolToUint64(rv.Greater(*l))}
		case NeoOpJump: pc = int(inst.Arg)
		case NeoOpJumpIfFalse:
			l := stack[sp]; sp--
			if !isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpGetGlobal:
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize)).Str
			val, _ := ctx.Get(name); sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = FromInterface(val)
		case NeoOpSetGlobal:
			name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize)).Str
			ctx.Set(name, stack[sp].ToInterface())
		case NeoOpReturn:
			if sp < 0 { return nil, nil }
			return stack[sp].ToInterface(), nil
		default:
			res, err := runNeoVMGeneralFallback(inst, &stack, &sp, &pc, pConsts, ctx)
			if err != nil { return nil, err }
			if inst.Op == NeoOpReturn { return res, nil }
		}
	}
	if sp < 0 { return nil, nil }
	return stack[sp].ToInterface(), nil
}

func runNeoVMGeneralFallback(inst *neoInstruction, stack *[64]Value, sp *int, pc *int, pConsts *Value, ctx Context) (any, error) {
	const valSize = unsafe.Sizeof(Value{})
	switch inst.Op {
	case NeoOpGreaterEqual:
		rv := stack[*sp]; *sp--; l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(l.Greater(rv) || l.Equal(rv))}
	case NeoOpLessEqual:
		rv := stack[*sp]; *sp--; l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(rv.Greater(*l) || l.Equal(rv))}
	case NeoOpAnd:
		rv := stack[*sp]; *sp--; l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(isValTruthy(*l) && isValTruthy(rv))}
	case NeoOpOr:
		rv := stack[*sp]; *sp--; l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(isValTruthy(*l) || isValTruthy(rv))}
	case NeoOpNot:
		l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(!isValTruthy(*l))}
	case NeoOpMod:
		rv := stack[*sp]; *sp--; l := &stack[*sp]
		res, err := l.ModErr(rv); if err != nil { return nil, err }; *l = res
	case NeoOpJumpIfTrue:
		l := stack[*sp]; *sp--
		if isValTruthy(l) { *pc = int(inst.Arg) }
	case NeoOpEqualConst, NeoOpEqualC:
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
		l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(l.Equal(*cv))}
	case NeoOpGreaterC:
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
		l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(l.Greater(*cv))}
	case NeoOpLessC:
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize))
		l := &stack[*sp]
		*l = Value{Type: ValBool, Num: boolToUint64(cv.Greater(*l))}
	case NeoOpEqualGlobalConst:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = Value{Type: ValBool, Num: boolToUint64(EqualAny(val, cv.ToInterface()))}
	case NeoOpAddGlobal, NeoOpAddGC:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = AddAny(val, cv.ToInterface())
	case NeoOpAddConstGlobal:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = AddAny(cv.ToInterface(), val)
	case NeoOpSubGC:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = SubAny(val, cv.ToInterface())
	case NeoOpMulGC:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = MulAny(val, cv.ToInterface())
	case NeoOpDivGC:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = DivAny(val, cv.ToInterface())
	case NeoOpSubCG:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = SubAny(cv.ToInterface(), val)
	case NeoOpMulCG:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = MulAny(cv.ToInterface(), val)
	case NeoOpDivCG:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = DivAny(cv.ToInterface(), val)
	case NeoOpGreaterGlobalConst:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = Value{Type: ValBool, Num: boolToUint64(GreaterAny(val, cv.ToInterface()))}
	case NeoOpLessGlobalConst:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = Value{Type: ValBool, Num: boolToUint64(LessAny(val, cv.ToInterface()))}
	case NeoOpAddGlobalGlobal:
		g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF
		n1 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g1Idx)*valSize)).Str
		n2 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g2Idx)*valSize)).Str
		v1, _ := ctx.Get(n1); v2, _ := ctx.Get(n2)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = AddAny(v1, v2)
	case NeoOpSubGlobalGlobal:
		g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF
		n1 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g1Idx)*valSize)).Str
		n2 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g2Idx)*valSize)).Str
		v1, _ := ctx.Get(n1); v2, _ := ctx.Get(n2)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = SubAny(v1, v2)
	case NeoOpMulGlobalGlobal:
		g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF
		n1 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g1Idx)*valSize)).Str
		n2 := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(g2Idx)*valSize)).Str
		v1, _ := ctx.Get(n1); v2, _ := ctx.Get(n2)
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		stack[*sp] = MulAny(v1, v2)
	case NeoOpFusedCompareGlobalConstJumpIfFalse:
		gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		if !EqualAny(val, cv.ToInterface()) { *pc = jTarget }
	case NeoOpFusedGreaterGlobalConstJumpIfFalse:
		gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		if !GreaterAny(val, cv.ToInterface()) { *pc = jTarget }
	case NeoOpFusedLessGlobalConstJumpIfFalse:
		gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*valSize))
		val, _ := ctx.Get(name)
		if !LessAny(val, cv.ToInterface()) { *pc = jTarget }
	case NeoOpGetGlobalJumpIfFalse:
		gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		val, _ := ctx.Get(name)
		if !isTruthy(val) { *pc = int(jTarget) }
	case NeoOpGetGlobalJumpIfTrue:
		gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(gIdx)*valSize)).Str
		val, _ := ctx.Get(name)
		if isTruthy(val) { *pc = int(jTarget) }
	case NeoOpCall:
		nameIdx := inst.Arg & 0xFFFF; numArgs := int(inst.Arg >> 16)
		name := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(nameIdx)*valSize)).Str
		var argsBuf [8]any; var args []any
		if numArgs <= 8 { args = argsBuf[:numArgs] } else { args = make([]any, numArgs) }
		for i := numArgs - 1; i >= 0; i-- {
			args[i] = stack[*sp].ToInterface(); *sp--
		}
		if builtin, ok := builtins[name]; ok {
			res, err := builtin(args...); if err != nil { return nil, err }
			*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[*sp] = FromInterface(res)
		} else { return nil, fmt.Errorf("builtin function not found: %s", name) }
	}
	return nil, nil
}

func (l Value) Equal(r Value) bool {
	if l.Type == r.Type {
		switch l.Type {
		case ValInt, ValFloat, ValBool: return l.Num == r.Num
		case ValString: return l.Str == r.Str
		case ValNil: return true
		}
	}
	lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
	if okL && okR { return lf == rf }
	return false
}

func (l Value) Greater(r Value) bool {
	if l.Type == ValInt && r.Type == ValInt { return int64(l.Num) > int64(r.Num) }
	lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
	if okL && okR { return lf > rf }
	return false
}

func (l Value) Add(r Value) Value {
	if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num + r.Num} }
	if l.Type == ValString && r.Type == ValString { return Value{Type: ValString, Str: l.Str + r.Str} }
	lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
	return Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
}

func (l Value) Sub(r Value) Value {
	if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num - r.Num} }
	lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
	return Value{Type: ValFloat, Num: math.Float64bits(lf - rf)}
}

func (l Value) Mul(r Value) Value {
	if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num * r.Num} }
	lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
	return Value{Type: ValFloat, Num: math.Float64bits(lf * rf)}
}

func (l Value) Div(r Value) Value {
	if (r.Type == ValInt && r.Num == 0) || (r.Type == ValFloat && math.Float64frombits(r.Num) == 0) { return Value{Type: ValFloat, Num: math.Float64bits(math.Inf(1))} }
	if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num / r.Num} }
	lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
	return Value{Type: ValFloat, Num: math.Float64bits(lf / rf)}
}

func (l Value) DivErr(r Value) (Value, error) {
	if (r.Type == ValInt && r.Num == 0) || (r.Type == ValFloat && math.Float64frombits(r.Num) == 0) { return Value{}, fmt.Errorf("division by zero") }
	if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num / r.Num}, nil }
	lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
	return Value{Type: ValFloat, Num: math.Float64bits(lf / rf)}, nil
}

func (l Value) ModErr(r Value) (Value, error) {
	if r.Type != ValInt { return Value{}, fmt.Errorf("modulo operator supports only integers") }
	if r.Num == 0 { return Value{}, fmt.Errorf("division by zero") }
	return Value{Type: ValInt, Num: l.Num % r.Num}, nil
}

func AddAny(v1, v2 any) Value {
	switch lv := v1.(type) {
	case int64:
		switch rv := v2.(type) {
		case int64: return Value{Type: ValInt, Num: uint64(lv + rv)}
		case float64: return Value{Type: ValFloat, Num: math.Float64bits(float64(lv) + rv)}
		}
	case float64:
		switch rv := v2.(type) {
		case int64: return Value{Type: ValFloat, Num: math.Float64bits(lv + float64(rv))}
		case float64: return Value{Type: ValFloat, Num: math.Float64bits(lv + rv)}
		}
	case string:
		if rv, ok := v2.(string); ok { return Value{Type: ValString, Str: lv + rv} }
	}
	return FromInterface(v1).Add(FromInterface(v2))
}

func SubAny(v1, v2 any) Value {
	switch lv := v1.(type) {
	case int64:
		switch rv := v2.(type) {
		case int64: return Value{Type: ValInt, Num: uint64(lv - rv)}
		case float64: return Value{Type: ValFloat, Num: math.Float64bits(float64(lv) - rv)}
		}
	case float64:
		switch rv := v2.(type) {
		case int64: return Value{Type: ValFloat, Num: math.Float64bits(lv - float64(rv))}
		case float64: return Value{Type: ValFloat, Num: math.Float64bits(lv - rv)}
		}
	}
	return FromInterface(v1).Sub(FromInterface(v2))
}

func MulAny(v1, v2 any) Value {
	switch lv := v1.(type) {
	case int64:
		switch rv := v2.(type) {
		case int64: return Value{Type: ValInt, Num: uint64(lv * rv)}
		case float64: return Value{Type: ValFloat, Num: math.Float64bits(float64(lv) * rv)}
		}
	case float64:
		switch rv := v2.(type) {
		case int64: return Value{Type: ValFloat, Num: math.Float64bits(lv * float64(rv))}
		case float64: return Value{Type: ValFloat, Num: math.Float64bits(lv * rv)}
		}
	}
	return FromInterface(v1).Mul(FromInterface(v2))
}

func DivAny(v1, v2 any) Value {
	switch lv := v1.(type) {
	case int64:
		switch rv := v2.(type) {
		case int64: 
			if rv == 0 { return Value{Type: ValFloat, Num: math.Float64bits(math.Inf(1))} }
			return Value{Type: ValInt, Num: uint64(lv / rv)}
		case float64:
			if rv == 0 { return Value{Type: ValFloat, Num: math.Float64bits(math.Inf(1))} }
			return Value{Type: ValFloat, Num: math.Float64bits(float64(lv) / rv)}
		}
	case float64:
		switch rv := v2.(type) {
		case int64:
			if rv == 0 { return Value{Type: ValFloat, Num: math.Float64bits(math.Inf(1))} }
			return Value{Type: ValFloat, Num: math.Float64bits(lv / float64(rv))}
		case float64:
			if rv == 0 { return Value{Type: ValFloat, Num: math.Float64bits(math.Inf(1))} }
			return Value{Type: ValFloat, Num: math.Float64bits(lv / rv)}
		}
	}
	return FromInterface(v1).Div(FromInterface(v2))
}

func EqualAny(v1, v2 any) bool {
	switch lv := v1.(type) {
	case int64:
		switch rv := v2.(type) {
		case int64: return lv == rv
		case float64: return float64(lv) == rv
		}
	case float64:
		switch rv := v2.(type) {
		case int64: return lv == float64(rv)
		case float64: return lv == rv
		}
	case string:
		if rv, ok := v2.(string); ok { return lv == rv }
	case bool:
		if rv, ok := v2.(bool); ok { return lv == rv }
	case nil: return v2 == nil
	}
	return FromInterface(v1).Equal(FromInterface(v2))
}

func GreaterAny(v1, v2 any) bool {
	switch lv := v1.(type) {
	case int64:
		switch rv := v2.(type) {
		case int64: return lv > rv
		case float64: return float64(lv) > rv
		}
	case float64:
		switch rv := v2.(type) {
		case int64: return lv > float64(rv)
		case float64: return lv > rv
		}
	}
	return FromInterface(v1).Greater(FromInterface(v2))
}

func LessAny(v1, v2 any) bool {
	switch lv := v1.(type) {
	case int64:
		switch rv := v2.(type) {
		case int64: return lv < rv
		case float64: return float64(lv) < rv
		}
	case float64:
		switch rv := v2.(type) {
		case int64: return lv < float64(rv)
		case float64: return lv < rv
		}
	}
	return FromInterface(v2).Greater(FromInterface(v1))
}
