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
	if bc == nil || len(bc.Instructions) == 0 { return nil, nil }
	if mctx, ok := any(ctx).(*MapContext); ok { return runNeoVMMapped(bc, mctx) }
	return runNeoVMGeneral(bc, ctx)
}

func runNeoVMMapped(bc *NeoBytecode, ctx *MapContext) (any, error) {
	var stack [64]Value
	sp := -1; pc := 0; insts := bc.Instructions; consts := bc.Constants; nInsts := len(insts); vars := ctx.vars
	for pc < nInsts {
		inst := insts[pc]; pc++
		switch inst.Op {
		case NeoOpPush: sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = consts[inst.Arg]
		case NeoOpPop: sp--
		case NeoOpAdd:
			r := stack[sp]; sp--; l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt { stack[sp] = Value{Type: ValInt, Num: l.Num + r.Num} } else if l.Type == ValString && r.Type == ValString { stack[sp] = Value{Type: ValString, Str: l.Str + r.Str} } else { stack[sp] = l.Add(r) }
		case NeoOpSub:
			r := stack[sp]; sp--; l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt { stack[sp] = Value{Type: ValInt, Num: l.Num - r.Num} } else { stack[sp] = l.Sub(r) }
		case NeoOpMul:
			r := stack[sp]; sp--; l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt { stack[sp] = Value{Type: ValInt, Num: l.Num * r.Num} } else { stack[sp] = l.Mul(r) }
		case NeoOpDiv:
			r := stack[sp]; sp--; l := stack[sp]
			res, err := l.DivErr(r); if err != nil { return nil, err }; stack[sp] = res
		case NeoOpMod:
			r := stack[sp]; sp--; l := stack[sp]
			res, err := l.ModErr(r); if err != nil { return nil, err }; stack[sp] = res
		case NeoOpEqual: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(l.Equal(r))}
		case NeoOpGreater: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(l.Greater(r))}
		case NeoOpLess: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(r.Greater(l))}
		case NeoOpGreaterEqual: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(l.Greater(r) || l.Equal(r))}
		case NeoOpLessEqual: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(r.Greater(l) || l.Equal(r))}
		case NeoOpAnd: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(isValTruthy(l) && isValTruthy(r))}
		case NeoOpOr: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(isValTruthy(l) || isValTruthy(r))}
		case NeoOpNot: l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(!isValTruthy(l))}
		case NeoOpJump: pc = int(inst.Arg)
		case NeoOpJumpIfFalse: l := stack[sp]; sp--; if !isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpJumpIfTrue: l := stack[sp]; sp--; if isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpGetGlobal:
			name := consts[inst.Arg].Str; sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }
			stack[sp] = FromInterface(vars[name])
		case NeoOpSetGlobal: name := consts[inst.Arg].Str; vars[name] = stack[sp].ToInterface()
		case NeoOpCall:
			nameIdx := inst.Arg & 0xFFFF; numArgs := int(inst.Arg >> 16); name := consts[nameIdx].Str
			var argsBuf [8]any; var args []any
			if numArgs <= 8 { args = argsBuf[:numArgs] } else { args = make([]any, numArgs) }
			for i := numArgs - 1; i >= 0; i-- { args[i] = stack[sp].ToInterface(); sp-- }
			if builtin, ok := builtins[name]; ok {
				res, err := builtin(args...); if err != nil { return nil, err }
				sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = FromInterface(res)
			} else { return nil, fmt.Errorf("builtin function not found: %s", name) }
		case NeoOpEqualConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF
			lv := FromInterface(vars[consts[gIdx].Str]); r := consts[cIdx]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValBool, Num: boolToUint64(lv.Equal(r))}
		case NeoOpAddGlobal:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; rv := consts[cIdx]; name := consts[gIdx].Str
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = AddAny(vars[name], rv.ToInterface())
		case NeoOpAddConstGlobal:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; lv := consts[cIdx]; name := consts[gIdx].Str
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = AddAny(lv.ToInterface(), vars[name])
		case NeoOpAddGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; v1 := vars[consts[g1Idx].Str]; v2 := vars[consts[g2Idx].Str]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = AddAny(v1, v2)
		case NeoOpSubGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; v1 := vars[consts[g1Idx].Str]; v2 := vars[consts[g2Idx].Str]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = SubAny(v1, v2)
		case NeoOpMulGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; v1 := vars[consts[g1Idx].Str]; v2 := vars[consts[g2Idx].Str]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = MulAny(v1, v2)
		case NeoOpAddGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; rv := consts[cIdx]; name := consts[gIdx].Str
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = AddAny(vars[name], rv.ToInterface())
		case NeoOpSubGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; rv := consts[cIdx]; name := consts[gIdx].Str
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = SubAny(vars[name], rv.ToInterface())
		case NeoOpMulGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; rv := consts[cIdx]; name := consts[gIdx].Str
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = MulAny(vars[name], rv.ToInterface())
		case NeoOpDivGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; rv := consts[cIdx]; name := consts[gIdx].Str
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = DivAny(vars[name], rv.ToInterface())
		case NeoOpEqualGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; r := consts[cIdx]; name := consts[gIdx].Str
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValBool, Num: boolToUint64(EqualAny(vars[name], r.ToInterface()))}
		case NeoOpGreaterGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; r := consts[cIdx]; name := consts[gIdx].Str
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValBool, Num: boolToUint64(GreaterAny(vars[name], r.ToInterface()))}
		case NeoOpLessGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; r := consts[cIdx]; name := consts[gIdx].Str
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValBool, Num: boolToUint64(LessAny(vars[name], r.ToInterface()))}
		case NeoOpAddC: cv := consts[inst.Arg]; stack[sp] = stack[sp].Add(cv)
		case NeoOpSubC: cv := consts[inst.Arg]; stack[sp] = stack[sp].Sub(cv)
		case NeoOpMulC: cv := consts[inst.Arg]; stack[sp] = stack[sp].Mul(cv)
		case NeoOpDivC: cv := consts[inst.Arg]; stack[sp] = stack[sp].Div(cv)
		case NeoOpFusedCompareGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
			if !EqualAny(vars[consts[gIdx].Str], consts[cIdx].ToInterface()) { pc = jTarget }
		case NeoOpFusedGreaterGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
			if !GreaterAny(vars[consts[gIdx].Str], consts[cIdx].ToInterface()) { pc = jTarget }
		case NeoOpFusedLessGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
			if !LessAny(vars[consts[gIdx].Str], consts[cIdx].ToInterface()) { pc = jTarget }
		case NeoOpGetGlobalJumpIfFalse:
			gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
			if !isTruthy(vars[consts[gIdx].Str]) { pc = int(jTarget) }
		case NeoOpGetGlobalJumpIfTrue:
			gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF
			if isTruthy(vars[consts[gIdx].Str]) { pc = int(jTarget) }
		case NeoOpConcat:
			numArgs := int(inst.Arg); totalLen := 0; var argStringsBuf [8]string; var argStrings []string
			if numArgs <= 8 { argStrings = argStringsBuf[:numArgs] } else { argStrings = make([]string, numArgs) }
			for i := numArgs - 1; i >= 0; i-- {
				v := stack[sp]; sp--; var s string
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
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValString, Str: res}
		case NeoOpConcat2:
			r := stack[sp]; sp--; l := stack[sp]
			var s1, s2 string
			if l.Type == ValString { s1 = l.Str } else { s1 = fmt.Sprintf("%v", l.ToInterface()) }
			if r.Type == ValString { s2 = r.Str } else { s2 = fmt.Sprintf("%v", r.ToInterface()) }
			stack[sp] = Value{Type: ValString, Str: s1 + s2}
		case NeoOpConcatGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; lv := vars[consts[gIdx].Str]; rv := consts[cIdx]
			var s1, s2 string
			if s, ok := lv.(string); ok { s1 = s } else { s1 = fmt.Sprintf("%v", lv) }
			if rv.Type == ValString { s2 = rv.Str } else { s2 = fmt.Sprintf("%v", rv.ToInterface()) }
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValString, Str: s1 + s2}
		case NeoOpConcatCG:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; lv := consts[cIdx]; rv := vars[consts[gIdx].Str]
			var s1, s2 string
			if lv.Type == ValString { s1 = lv.Str } else { s1 = fmt.Sprintf("%v", lv.ToInterface()) }
			if s, ok := rv.(string); ok { s2 = s } else { s2 = fmt.Sprintf("%v", rv) }
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValString, Str: s1 + s2}
		case NeoOpReturn: if sp < 0 { return nil, nil }; return stack[sp].ToInterface(), nil
		}
	}
	if sp < 0 { return nil, nil }; return stack[sp].ToInterface(), nil
}

func runNeoVMGeneral(bc *NeoBytecode, ctx Context) (any, error) {
	var stack [64]Value
	sp := -1; pc := 0; insts := bc.Instructions; consts := bc.Constants; nInsts := len(insts)
	for pc < nInsts {
		inst := insts[pc]; pc++
		switch inst.Op {
		case NeoOpPush: sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = consts[inst.Arg]
		case NeoOpPop: sp--
		case NeoOpAdd: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = l.Add(r)
		case NeoOpSub: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = l.Sub(r)
		case NeoOpMul: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = l.Mul(r)
		case NeoOpDiv: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = l.Div(r)
		case NeoOpMod: r := stack[sp]; sp--; l := stack[sp]; res, err := l.ModErr(r); if err != nil { return nil, err }; stack[sp] = res
		case NeoOpEqual: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(l.Equal(r))}
		case NeoOpGreater: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(l.Greater(r))}
		case NeoOpLess: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(r.Greater(l))}
		case NeoOpGreaterEqual: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(l.Greater(r) || l.Equal(r))}
		case NeoOpLessEqual: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(r.Greater(l) || l.Equal(r))}
		case NeoOpAnd: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(isValTruthy(l) && isValTruthy(r))}
		case NeoOpOr: r := stack[sp]; sp--; l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(isValTruthy(l) || isValTruthy(r))}
		case NeoOpNot: l := stack[sp]; stack[sp] = Value{Type: ValBool, Num: boolToUint64(!isValTruthy(l))}
		case NeoOpJump: pc = int(inst.Arg)
		case NeoOpJumpIfFalse: l := stack[sp]; sp--; if !isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpJumpIfTrue: l := stack[sp]; sp--; if isValTruthy(l) { pc = int(inst.Arg) }
		case NeoOpGetGlobal: name := consts[inst.Arg].Str; val, _ := ctx.Get(name); sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = FromInterface(val)
		case NeoOpSetGlobal: name := consts[inst.Arg].Str; ctx.Set(name, stack[sp].ToInterface())
		case NeoOpCall:
			nameIdx := inst.Arg & 0xFFFF; numArgs := int(inst.Arg >> 16); name := consts[nameIdx].Str
			args := make([]any, numArgs)
			for i := numArgs - 1; i >= 0; i-- { args[i] = stack[sp].ToInterface(); sp-- }
			if builtin, ok := builtins[name]; ok {
				res, err := builtin(args...); if err != nil { return nil, err }
				sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = FromInterface(res)
			} else { return nil, fmt.Errorf("builtin function not found: %s", name) }
		case NeoOpEqualConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str); r := consts[cIdx]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValBool, Num: boolToUint64(FromInterface(val).Equal(r))}
		case NeoOpAddGlobal:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str); rv := consts[cIdx]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = AddAny(val, rv.ToInterface())
		case NeoOpAddConstGlobal:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; lv := consts[cIdx]; val, _ := ctx.Get(consts[gIdx].Str)
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = AddAny(lv.ToInterface(), val)
		case NeoOpAddGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; v1, _ := ctx.Get(consts[g1Idx].Str); v2, _ := ctx.Get(consts[g2Idx].Str)
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = AddAny(v1, v2)
		case NeoOpSubGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; v1, _ := ctx.Get(consts[g1Idx].Str); v2, _ := ctx.Get(consts[g2Idx].Str)
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = SubAny(v1, v2)
		case NeoOpMulGlobalGlobal:
			g1Idx := inst.Arg >> 16; g2Idx := inst.Arg & 0xFFFF; v1, _ := ctx.Get(consts[g1Idx].Str); v2, _ := ctx.Get(consts[g2Idx].Str)
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = MulAny(v1, v2)
		case NeoOpAddGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str); rv := consts[cIdx]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = AddAny(val, rv.ToInterface())
		case NeoOpSubGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str); rv := consts[cIdx]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = SubAny(val, rv.ToInterface())
		case NeoOpMulGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str); rv := consts[cIdx]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = MulAny(val, rv.ToInterface())
		case NeoOpDivGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str); rv := consts[cIdx]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = DivAny(val, rv.ToInterface())
		case NeoOpEqualGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str); r := consts[cIdx]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValBool, Num: boolToUint64(EqualAny(val, r.ToInterface()))}
		case NeoOpGreaterGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str); r := consts[cIdx]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValBool, Num: boolToUint64(GreaterAny(val, r.ToInterface()))}
		case NeoOpLessGlobalConst:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str); r := consts[cIdx]
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValBool, Num: boolToUint64(LessAny(val, r.ToInterface()))}
		case NeoOpAddC: cv := consts[inst.Arg]; stack[sp] = stack[sp].Add(cv)
		case NeoOpSubC: cv := consts[inst.Arg]; stack[sp] = stack[sp].Sub(cv)
		case NeoOpMulC: cv := consts[inst.Arg]; stack[sp] = stack[sp].Mul(cv)
		case NeoOpDivC: cv := consts[inst.Arg]; stack[sp] = stack[sp].Div(cv)
		case NeoOpFusedCompareGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
			val, _ := ctx.Get(consts[gIdx].Str); r := consts[cIdx]
			if !EqualAny(val, r.ToInterface()) { pc = jTarget }
		case NeoOpFusedGreaterGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
			val, _ := ctx.Get(consts[gIdx].Str); r := consts[cIdx]
			if !GreaterAny(val, r.ToInterface()) { pc = jTarget }
		case NeoOpFusedLessGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF; cIdx := int(inst.Arg >> 12) & 0x3FF; jTarget := int(inst.Arg) & 0xFFF
			val, _ := ctx.Get(consts[gIdx].Str); r := consts[cIdx]
			if !LessAny(val, r.ToInterface()) { pc = jTarget }
		case NeoOpGetGlobalJumpIfFalse:
			gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str)
			if !isTruthy(val) { pc = int(jTarget) }
		case NeoOpGetGlobalJumpIfTrue:
			gIdx := inst.Arg >> 16; jTarget := inst.Arg & 0xFFFF; val, _ := ctx.Get(consts[gIdx].Str)
			if isTruthy(val) { pc = int(jTarget) }
		case NeoOpConcat:
			numArgs := int(inst.Arg); totalLen := 0; var argStringsBuf [8]string; var argStrings []string
			if numArgs <= 8 { argStrings = argStringsBuf[:numArgs] } else { argStrings = make([]string, numArgs) }
			for i := numArgs - 1; i >= 0; i-- {
				v := stack[sp]; sp--; var s string
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
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValString, Str: res}
		case NeoOpConcat2:
			r := stack[sp]; sp--; l := stack[sp]
			var s1, s2 string
			if l.Type == ValString { s1 = l.Str } else { s1 = fmt.Sprintf("%v", l.ToInterface()) }
			if r.Type == ValString { s2 = r.Str } else { s2 = fmt.Sprintf("%v", r.ToInterface()) }
			stack[sp] = Value{Type: ValString, Str: s1 + s2}
		case NeoOpConcatGC:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; lv, _ := ctx.Get(consts[gIdx].Str); rv := consts[cIdx]
			var s1, s2 string
			if s, ok := lv.(string); ok { s1 = s } else { s1 = fmt.Sprintf("%v", lv) }
			if rv.Type == ValString { s2 = rv.Str } else { s2 = fmt.Sprintf("%v", rv.ToInterface()) }
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValString, Str: s1 + s2}
		case NeoOpConcatCG:
			gIdx := inst.Arg >> 16; cIdx := inst.Arg & 0xFFFF; lv := consts[cIdx]; rv, _ := ctx.Get(consts[gIdx].Str)
			var s1, s2 string
			if lv.Type == ValString { s1 = lv.Str } else { s1 = fmt.Sprintf("%v", lv.ToInterface()) }
			if s, ok := rv.(string); ok { s2 = s } else { s2 = fmt.Sprintf("%v", rv) }
			sp++; if sp >= 64 { return nil, fmt.Errorf("NeoVM stack overflow") }; stack[sp] = Value{Type: ValString, Str: s1 + s2}
		case NeoOpReturn: if sp < 0 { return nil, nil }; return stack[sp].ToInterface(), nil
		}
	}
	if sp < 0 { return nil, nil }; return stack[sp].ToInterface(), nil
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
