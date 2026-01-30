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
	sp := -1; pc := 0; insts := bc.Instructions; consts := bc.Constants; names := bc.Names; nInsts := len(insts)
	if nInsts == 0 { return nil, nil }
	pInsts := unsafe.SliceData(insts)
	pConsts := unsafe.SliceData(consts)
	pNames := unsafe.SliceData(names)
	pStack := unsafe.SliceData(stack[:])
	for pc < nInsts {
		inst := (*neoInstruction)(unsafe.Add(unsafe.Pointer(pInsts), uintptr(pc)*unsafe.Sizeof(neoInstruction{})))
		pc++
		switch inst.Op {
		case NeoOpPush:
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))) = *(*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*unsafe.Sizeof(Value{})))
		case NeoOpPop: sp--
		case NeoOpAdd:
			r := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			if l.Type == ValInt && r.Type == ValInt { l.Num += r.Num } else if l.Type == ValString && r.Type == ValString { l.Str += r.Str } else { *l = l.Add(*r) }
		case NeoOpSub:
			r := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			if l.Type == ValInt && r.Type == ValInt { l.Num -= r.Num } else { *l = l.Sub(*r) }
		case NeoOpMul:
			r := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			if l.Type == ValInt && r.Type == ValInt { l.Num *= r.Num } else { *l = l.Mul(*r) }
		case NeoOpDiv:
			rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			res, err := l.DivErr(rv); if err != nil { return nil, err }; *l = res
		case NeoOpEqual:
			rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			*l = Value{Type: ValBool, Num: boolToUint64(l.Equal(rv))}
		case NeoOpGreater:
			rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			*l = Value{Type: ValBool, Num: boolToUint64(l.Greater(rv))}
		case NeoOpLess:
			rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			*l = Value{Type: ValBool, Num: boolToUint64(rv.Greater(*l))}
		case NeoOpJump: pc = int(inst.Arg)
		case NeoOpJumpIfFalse:
			l := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			if !isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpGetGlobal:
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(inst.Arg)*unsafe.Sizeof("")))
			val := vars[name]
			target := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			switch v := val.(type) {
			case int64: *target = Value{Type: ValInt, Num: uint64(v)}
			case float64: *target = Value{Type: ValFloat, Num: math.Float64bits(v)}
			case string: *target = Value{Type: ValString, Str: v}
			case bool: *target = Value{Type: ValBool, Num: boolToUint64(v)}
			case nil: *target = Value{Type: ValNil}
			default: *target = FromInterface(v)
			}
		case NeoOpSetGlobal:
			name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(inst.Arg)*unsafe.Sizeof("")))
			val := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			vars[name] = val.ToInterface()
		case NeoOpEqualConst, NeoOpEqualGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
			val := vars[name]
			res := false
			switch v := val.(type) {
			case int64: res = cv.Type == ValInt && v == int64(cv.Num)
			case float64: res = cv.Type == ValFloat && v == math.Float64frombits(cv.Num)
			case string: res = cv.Type == ValString && v == cv.Str
			default: res = EqualAny(val, cv.ToInterface())
			}
			*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))) = Value{Type: ValBool, Num: boolToUint64(res)}
		case NeoOpAddGlobal, NeoOpAddGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
			val := vars[name]
			target := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			switch v := val.(type) {
			case int64:
				if cv.Type == ValInt { *target = Value{Type: ValInt, Num: uint64(v + int64(cv.Num))} } else if cv.Type == ValFloat { *target = Value{Type: ValFloat, Num: math.Float64bits(float64(v) + math.Float64frombits(cv.Num))} } else { *target = AddAny(v, cv.ToInterface()) }
			case float64:
				if cv.Type == ValInt { *target = Value{Type: ValFloat, Num: math.Float64bits(v + float64(int64(cv.Num)))} } else if cv.Type == ValFloat { *target = Value{Type: ValFloat, Num: math.Float64bits(v + math.Float64frombits(cv.Num))} } else { *target = AddAny(v, cv.ToInterface()) }
			case string:
				if cv.Type == ValString { *target = Value{Type: ValString, Str: v + cv.Str} } else { *target = AddAny(v, cv.ToInterface()) }
			default: *target = AddAny(v, cv.ToInterface())
			}
		case NeoOpFusedCompareGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
			name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
			cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
			val := vars[name]
			res := false
			switch v := val.(type) {
			case int64: res = cv.Type == ValInt && v == int64(cv.Num)
			case float64: res = cv.Type == ValFloat && v == math.Float64frombits(cv.Num)
			case string: res = cv.Type == ValString && v == cv.Str
			default: res = EqualAny(val, cv.ToInterface())
			}
			if !res { pc = jTarget }
		case NeoOpReturn:
			if sp < 0 { return nil, nil }
			return (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))).ToInterface(), nil
		default:
			// Fallback to non-optimized path if needed, but we should have most ops here
			res, err := runNeoVMWithMapFallback(inst, pStack, &sp, &pc, pNames, pConsts, vars)
			if err != nil { return nil, err }
			if inst.Op == NeoOpReturn { return res, nil }
		}
	}
	if sp < 0 { return nil, nil }
	return (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))).ToInterface(), nil
}

func runNeoVMWithMapFallback(inst *neoInstruction, pStack *Value, sp *int, pc *int, pNames *string, pConsts *Value, vars map[string]any) (any, error) {
	switch inst.Op {
	case NeoOpGreaterEqual:
		rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		*l = Value{Type: ValBool, Num: boolToUint64(l.Greater(rv) || l.Equal(rv))}
	case NeoOpLessEqual:
		rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		*l = Value{Type: ValBool, Num: boolToUint64(rv.Greater(*l) || l.Equal(rv))}
	case NeoOpAnd:
		rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		*l = Value{Type: ValBool, Num: boolToUint64(isValTruthy(*l) && isValTruthy(rv))}
	case NeoOpOr:
		rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		*l = Value{Type: ValBool, Num: boolToUint64(isValTruthy(*l) || isValTruthy(rv))}
	case NeoOpNot:
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		*l = Value{Type: ValBool, Num: boolToUint64(!isValTruthy(*l))}
	case NeoOpAddConstGlobal:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = AddAny(cv.ToInterface(), vars[name])
	case NeoOpSubGC:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = SubAny(vars[name], cv.ToInterface())
	case NeoOpMulGC:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = MulAny(vars[name], cv.ToInterface())
	case NeoOpDivGC:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = DivAny(vars[name], cv.ToInterface())
	case NeoOpGreaterGlobalConst:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = Value{Type: ValBool, Num: boolToUint64(GreaterAny(vars[name], cv.ToInterface()))}
	case NeoOpLessGlobalConst:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = Value{Type: ValBool, Num: boolToUint64(LessAny(vars[name], cv.ToInterface()))}
	case NeoOpAddGlobalGlobal:
		g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		n1 := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(g1Idx)*unsafe.Sizeof("")))
		n2 := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(g2Idx)*unsafe.Sizeof("")))
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = AddAny(vars[n1], vars[n2])
	case NeoOpSubGlobalGlobal:
		g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		n1 := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(g1Idx)*unsafe.Sizeof("")))
		n2 := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(g2Idx)*unsafe.Sizeof("")))
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = SubAny(vars[n1], vars[n2])
	case NeoOpMulGlobalGlobal:
		g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		n1 := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(g1Idx)*unsafe.Sizeof("")))
		n2 := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(g2Idx)*unsafe.Sizeof("")))
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = MulAny(vars[n1], vars[n2])
	case NeoOpFusedGreaterGlobalConstJumpIfFalse:
		gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		if !GreaterAny(vars[name], cv.ToInterface()) { *pc = jTarget }
	case NeoOpFusedLessGlobalConstJumpIfFalse:
		gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		if !LessAny(vars[name], cv.ToInterface()) { *pc = jTarget }
	case NeoOpGetGlobalJumpIfFalse:
		gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		if !isTruthy(vars[name]) { *pc = int(jTarget) }
	case NeoOpGetGlobalJumpIfTrue:
		gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		if isTruthy(vars[name]) { *pc = int(jTarget) }
	case NeoOpAddC:
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*unsafe.Sizeof(Value{})))
		*l = l.Add(*cv)
	case NeoOpSubC:
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*unsafe.Sizeof(Value{})))
		*l = l.Sub(*cv)
	case NeoOpMulC:
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*unsafe.Sizeof(Value{})))
		*l = l.Mul(*cv)
	case NeoOpDivC:
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*unsafe.Sizeof(Value{})))
		*l = l.Div(*cv)
	case NeoOpConcat:
		numArgs := int(inst.Arg); totalLen := 0; var argStringsBuf [8]string; var argStrings []string
		if numArgs <= 8 { argStrings = argStringsBuf[:numArgs] } else { argStrings = make([]string, numArgs) }
		for i := numArgs - 1; i >= 0; i-- {
			v := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
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
		*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = Value{Type: ValString, Str: res}
	case NeoOpConcatGC:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		lv := vars[name]; var s1, s2 string
		if s, ok := lv.(string); ok { s1 = s } else { s1 = fmt.Sprintf("%v", lv) }
		if cv.Type == ValString { s2 = cv.Str } else { s2 = fmt.Sprintf("%v", cv.ToInterface()) }
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = Value{Type: ValString, Str: s1 + s2}
	case NeoOpConcatCG:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		rv := vars[name]; var s1, s2 string
		if cv.Type == ValString { s1 = cv.Str } else { s1 = fmt.Sprintf("%v", cv.ToInterface()) }
		if s, ok := rv.(string); ok { s2 = s } else { s2 = fmt.Sprintf("%v", rv) }
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = Value{Type: ValString, Str: s1 + s2}
	case NeoOpCall:
		nameIdx := inst.Arg & 0xFFFF; numArgs := int(inst.Arg >> 16)
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(nameIdx)*unsafe.Sizeof("")))
		var argsBuf [8]any; var args []any
		if numArgs <= 8 { args = argsBuf[:numArgs] } else { args = make([]any, numArgs) }
		for i := numArgs - 1; i >= 0; i-- {
			val := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
			args[i] = val.ToInterface(); *sp--
		}
		if builtin, ok := builtins[name]; ok {
			res, err := builtin(args...); if err != nil { return nil, err }
			*sp++; if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = FromInterface(res)
		} else { return nil, fmt.Errorf("builtin function not found: %s", name) }
	case NeoOpJumpIfTrue:
		l := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		if isValTruthy(l) { *pc = int(inst.Arg) }
	case NeoOpAddInt:
		r := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		l.Num += r.Num
	case NeoOpSubInt:
		r := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		l.Num -= r.Num
	case NeoOpMulInt:
		r := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		l.Num *= r.Num
	}
	return nil, nil
}

func runNeoVMGeneral(bc *NeoBytecode, ctx Context) (any, error) {
	var stack [64]Value
	sp := -1; pc := 0; insts := bc.Instructions; consts := bc.Constants; names := bc.Names; nInsts := len(insts)
	pInsts := unsafe.SliceData(insts)
	pConsts := unsafe.SliceData(consts)
	pNames := unsafe.SliceData(names)
	pStack := unsafe.SliceData(stack[:])
	for pc < nInsts {
		inst := (*neoInstruction)(unsafe.Add(unsafe.Pointer(pInsts), uintptr(pc)*unsafe.Sizeof(neoInstruction{})))
		pc++
		switch inst.Op {
		case NeoOpPush:
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))) = *(*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*unsafe.Sizeof(Value{})))
		case NeoOpPop: sp--
		case NeoOpAdd:
			r := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			*l = l.Add(*r)
		case NeoOpSub:
			r := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			*l = l.Sub(*r)
		case NeoOpMul:
			r := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			*l = l.Mul(*r)
		case NeoOpDiv:
			rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			*l = l.Div(rv)
		case NeoOpMod:
			rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			res, err := l.ModErr(rv); if err != nil { return nil, err }; *l = res
		case NeoOpEqual:
			rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			*l = Value{Type: ValBool, Num: boolToUint64(l.Equal(rv))}
		case NeoOpGreater:
			rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			*l = Value{Type: ValBool, Num: boolToUint64(l.Greater(rv))}
		case NeoOpLess:
			rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			*l = Value{Type: ValBool, Num: boolToUint64(rv.Greater(*l))}
		case NeoOpJump: pc = int(inst.Arg)
		case NeoOpJumpIfFalse:
			l := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))); sp--
			if !isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpGetGlobal:
			name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(inst.Arg)*unsafe.Sizeof("")))
			val, _ := ctx.Get(name); sp++
			if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))) = FromInterface(val)
		case NeoOpSetGlobal:
			name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(inst.Arg)*unsafe.Sizeof("")))
			val := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{})))
			ctx.Set(name, val.ToInterface())
		case NeoOpReturn:
			if sp < 0 { return nil, nil }
			return (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))).ToInterface(), nil
		default:
			res, err := runNeoVMGeneralFallback(inst, pStack, &sp, &pc, pNames, pConsts, ctx)
			if err != nil { return nil, err }
			if inst.Op == NeoOpReturn { return res, nil }
		}
	}
	if sp < 0 { return nil, nil }
	return (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(sp)*unsafe.Sizeof(Value{}))).ToInterface(), nil
}

func runNeoVMGeneralFallback(inst *neoInstruction, pStack *Value, sp *int, pc *int, pNames *string, pConsts *Value, ctx Context) (any, error) {
	switch inst.Op {
	case NeoOpGreaterEqual:
		rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		*l = Value{Type: ValBool, Num: boolToUint64(l.Greater(rv) || l.Equal(rv))}
	case NeoOpLessEqual:
		rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		*l = Value{Type: ValBool, Num: boolToUint64(rv.Greater(*l) || l.Equal(rv))}
	case NeoOpAnd:
		rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		*l = Value{Type: ValBool, Num: boolToUint64(isValTruthy(*l) && isValTruthy(rv))}
	case NeoOpOr:
		rv := *(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))); *sp--
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		*l = Value{Type: ValBool, Num: boolToUint64(isValTruthy(*l) || isValTruthy(rv))}
	case NeoOpNot:
		l := (*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{})))
		*l = Value{Type: ValBool, Num: boolToUint64(!isValTruthy(*l))}
	case NeoOpAddConstGlobal:
		gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; *sp++
		if *sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
		name := *(*string)(unsafe.Add(unsafe.Pointer(pNames), uintptr(gIdx)*unsafe.Sizeof("")))
		cv := (*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(cIdx)*unsafe.Sizeof(Value{})))
		val, _ := ctx.Get(name)
		*(*Value)(unsafe.Add(unsafe.Pointer(pStack), uintptr(*sp)*unsafe.Sizeof(Value{}))) = AddAny(cv.ToInterface(), val)
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
