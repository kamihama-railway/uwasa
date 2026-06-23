// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"bytes"
	"fmt"
	"math"
)

func RunVM(bc *RenderedBytecode, ctx Context) (any, error) {
	if bc == nil || len(bc.Instructions) == 0 {
		return nil, nil
	}

	mapCtx, isMapCtx := ctx.(*MapContext)
	if isMapCtx {
		return runVMMapped(bc, mapCtx)
	}
	return runVMGeneral(bc, ctx)
}

func runVMMapped(bc *RenderedBytecode, ctx *MapContext) (any, error) {
	var stack [64]Value
	sp := -1
	pc := 0
	insts := bc.Instructions
	consts := bc.Constants
	nInsts := len(insts)
	vars := ctx.vars

	for pc < nInsts {
		inst := insts[pc]
		pc++

		switch inst.Op {
		case OpPush:
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = consts[inst.Arg]
		case OpPop:
			sp--
		case OpAdd:
			r := stack[sp]
			sp--
			l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: l.Num + r.Num}
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
				stack[sp] = Value{Type: ValInt, Num: l.Num - r.Num}
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
				stack[sp] = Value{Type: ValInt, Num: l.Num * r.Num}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf * rf)}
			}
		case OpDiv:
			r := stack[sp]
			sp--
			l := stack[sp]
			if r.Type == ValInt && r.Num == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			if r.Type == ValFloat && math.Float64frombits(r.Num) == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: l.Num / r.Num}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf / rf)}
			}
		case OpMod:
			r := stack[sp]
			sp--
			l := stack[sp]
			if r.Type != ValInt {
				return nil, fmt.Errorf("modulo operator supports only integers")
			}
			if r.Num == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			stack[sp] = Value{Type: ValInt, Num: l.Num % r.Num}
		case OpEqual:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == r.Type {
				switch l.Type {
				case ValInt, ValFloat, ValBool:
					res = l.Num == r.Num
				case ValString:
					res = l.Str == r.Str
				case ValNil:
					res = true
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
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = FromInterface(vars[name])
		case OpSetGlobal:
			name := consts[inst.Arg].Str
			val := stack[sp]
			vars[name] = val.ToInterface()
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
				if err != nil {
					return nil, err
				}
				sp++
				if sp >= 64 {
					return nil, fmt.Errorf("VM stack overflow")
				}
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
				case ValInt, ValFloat, ValBool:
					res = l.Num == r.Num
				case ValString:
					res = l.Str == r.Str
				case ValNil:
					res = true
				}
			} else {
				lf, okL := valToFloat64(l)
				rf, okR := valToFloat64(r)
				if okL && okR {
					res = lf == rf
				}
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case OpAddGlobal:
			gIdx := inst.Arg & 0xFFFF
			cIdx := inst.Arg >> 16
			name := consts[gIdx].Str
			lv := FromInterface(vars[name])
			rv := consts[cIdx]
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			if lv.Type == ValInt && rv.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num}
			} else if lv.Type == ValString && rv.Type == ValString {
				stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str}
			} else {
				lf, _ := valToFloat64(lv)
				rf, _ := valToFloat64(rv)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case OpAddGlobalGlobal:
			g1Idx := inst.Arg >> 16
			g2Idx := inst.Arg & 0xFFFF
			lv := FromInterface(vars[consts[g1Idx].Str])
			rv := FromInterface(vars[consts[g2Idx].Str])
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			if lv.Type == ValInt && rv.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num}
			} else if lv.Type == ValString && rv.Type == ValString {
				stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str}
			} else {
				lf, _ := valToFloat64(lv)
				rf, _ := valToFloat64(rv)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case OpEqualGlobalConst:
			gIdx := inst.Arg >> 16
			cIdx := inst.Arg & 0xFFFF
			lv := FromInterface(vars[consts[gIdx].Str])
			r := consts[cIdx]
			res := false
			if lv.Type == r.Type {
				switch lv.Type {
				case ValInt, ValFloat, ValBool:
					res = lv.Num == r.Num
				case ValString:
					res = lv.Str == r.Str
				case ValNil:
					res = true
				}
			} else {
				lf, okL := valToFloat64(lv)
				rf, okR := valToFloat64(r)
				if okL && okR {
					res = lf == rf
				}
			}
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case OpGreaterGlobalConst:
			gIdx := inst.Arg >> 16
			cIdx := inst.Arg & 0xFFFF
			lv := FromInterface(vars[consts[gIdx].Str])
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
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case OpLessGlobalConst:
			gIdx := inst.Arg >> 16
			cIdx := inst.Arg & 0xFFFF
			lv := FromInterface(vars[consts[gIdx].Str])
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
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case OpFusedCompareGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg>>22) & 0x3FF
			cIdx := int(inst.Arg>>12) & 0x3FF
			jTarget := int(inst.Arg) & 0xFFF
			lv := FromInterface(vars[consts[gIdx].Str])
			r := consts[cIdx]
			res := false
			if lv.Type == r.Type {
				switch lv.Type {
				case ValInt, ValFloat, ValBool:
					res = lv.Num == r.Num
				case ValString:
					res = lv.Str == r.Str
				case ValNil:
					res = true
				}
			} else {
				lf, okL := valToFloat64(lv)
				rf, okR := valToFloat64(r)
				if okL && okR {
					res = lf == rf
				}
			}
			if !res {
				pc = jTarget
			}
		case OpGetGlobalJumpIfFalse:
			gIdx := inst.Arg >> 16
			jTarget := inst.Arg & 0xFFFF
			if !isValTruthyAny(vars[consts[gIdx].Str]) {
				pc = int(jTarget)
			}
		case OpGetGlobalJumpIfTrue:
			gIdx := inst.Arg >> 16
			jTarget := inst.Arg & 0xFFFF
			if isValTruthyAny(vars[consts[gIdx].Str]) {
				pc = int(jTarget)
			}
		case OpMapGet:
			key := stack[sp]
			sp--
			obj := stack[sp]
			if obj.Type != ValMap {
				return nil, fmt.Errorf("MGET: not a map")
			}
			if key.Type != ValString {
				return nil, fmt.Errorf("MGET: key must be string")
			}
			m := obj.Ptr.(map[string]any)
			stack[sp] = FromInterface(m[key.Str])
		case OpMapSet:
			val := stack[sp]
			sp--
			key := stack[sp]
			sp--
			obj := stack[sp]
			if obj.Type != ValMap {
				return nil, fmt.Errorf("MSET: not a map")
			}
			if key.Type != ValString {
				return nil, fmt.Errorf("MSET: key must be string")
			}
			m := obj.Ptr.(map[string]any)
			m[key.Str] = val.ToInterface()
			stack[sp] = Value{Type: ValNil}
		case OpMapHas:
			key := stack[sp]
			sp--
			obj := stack[sp]
			if obj.Type != ValMap {
				return nil, fmt.Errorf("MHAS: not a map")
			}
			if key.Type != ValString {
				return nil, fmt.Errorf("MHAS: key must be string")
			}
			m := obj.Ptr.(map[string]any)
			_, ok := m[key.Str]
			res := uint64(0)
			if ok {
				res = 1
			}
			stack[sp] = Value{Type: ValBool, Num: res}
		case OpMapGetConst:
			obj := &stack[sp]
			if obj.Type != ValMap { return nil, fmt.Errorf("MGETC: not a map") }
			m := obj.Ptr.(map[string]any)
			val := m[consts[inst.Arg].Str]
			switch v := val.(type) {
			case int64: *obj = Value{Type: ValInt, Num: uint64(v)}
			case int: *obj = Value{Type: ValInt, Num: uint64(int64(v))}
			case float64: *obj = Value{Type: ValFloat, Num: math.Float64bits(v)}
			case string: *obj = Value{Type: ValString, Str: v}
			case bool: *obj = Value{Type: ValBool, Num: boolToUint64(v)}
			case nil: *obj = Value{Type: ValNil}
			default: *obj = FromInterface(v)
			}
		case OpMapDel:
			key := stack[sp]
			sp--
			obj := stack[sp]
			if obj.Type != ValMap {
				return nil, fmt.Errorf("MDEL: not a map")
			}
			if key.Type != ValString {
				return nil, fmt.Errorf("MDEL: key must be string")
			}
			m := obj.Ptr.(map[string]any)
			delete(m, key.Str)
			stack[sp] = Value{Type: ValNil}
		case OpConcat:
			numArgs := int(inst.Arg)
			totalLen := 0
			var argStringsBuf [8]string
			var argStrings []string
			if numArgs <= 8 {
				argStrings = argStringsBuf[:numArgs]
			} else {
				argStrings = make([]string, numArgs)
			}
			for i := numArgs - 1; i >= 0; i-- {
				v := stack[sp]
				sp--
				var s string
				switch v.Type {
				case ValString:
					s = v.Str
				case ValInt:
					s = fmt.Sprintf("%d", int64(v.Num))
				case ValFloat:
					s = fmt.Sprintf("%g", math.Float64frombits(v.Num))
				case ValBool:
					if v.Num != 0 {
						s = "true"
					} else {
						s = "false"
					}
				default:
					s = fmt.Sprintf("%v", v.ToInterface())
				}
				argStrings[i] = s
				totalLen += len(s)
			}
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.Grow(totalLen)
			for _, s := range argStrings {
				buf.WriteString(s)
			}
			res := buf.String()
			bufferPool.Put(buf)
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = Value{Type: ValString, Str: res}
		}
	}
	if sp < 0 {
		return nil, nil
	}
	return stack[sp].ToInterface(), nil
}

func runVMGeneral(bc *RenderedBytecode, ctx Context) (any, error) {
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
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = consts[inst.Arg]
		case OpPop:
			sp--
		case OpAdd:
			r := stack[sp]
			sp--
			l := stack[sp]
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: l.Num + r.Num}
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
				stack[sp] = Value{Type: ValInt, Num: l.Num - r.Num}
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
				stack[sp] = Value{Type: ValInt, Num: l.Num * r.Num}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf * rf)}
			}
		case OpDiv:
			r := stack[sp]
			sp--
			l := stack[sp]
			if r.Type == ValInt && r.Num == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			if r.Type == ValFloat && math.Float64frombits(r.Num) == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			if l.Type == ValInt && r.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: l.Num / r.Num}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf / rf)}
			}
		case OpMod:
			r := stack[sp]
			sp--
			l := stack[sp]
			if r.Type != ValInt {
				return nil, fmt.Errorf("modulo operator supports only integers")
			}
			if r.Num == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			stack[sp] = Value{Type: ValInt, Num: l.Num % r.Num}
		case OpEqual:
			r := stack[sp]
			sp--
			l := stack[sp]
			res := false
			if l.Type == r.Type {
				switch l.Type {
				case ValInt, ValFloat, ValBool:
					res = l.Num == r.Num
				case ValString:
					res = l.Str == r.Str
				case ValNil:
					res = true
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
			val, _ := ctx.Get(name)
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = FromInterface(val)
		case OpSetGlobal:
			name := consts[inst.Arg].Str
			val := stack[sp]
			ctx.Set(name, val.ToInterface())
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
				if err != nil {
					return nil, err
				}
				sp++
				if sp >= 64 {
					return nil, fmt.Errorf("VM stack overflow")
				}
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
				case ValInt, ValFloat, ValBool:
					res = l.Num == r.Num
				case ValString:
					res = l.Str == r.Str
				case ValNil:
					res = true
				}
			} else {
				lf, okL := valToFloat64(l)
				rf, okR := valToFloat64(r)
				if okL && okR {
					res = lf == rf
				}
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case OpAddGlobal:
			gIdx := inst.Arg & 0xFFFF
			cIdx := inst.Arg >> 16
			val, _ := ctx.Get(consts[gIdx].Str)
			lv := FromInterface(val)
			rv := consts[cIdx]
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			if lv.Type == ValInt && rv.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num}
			} else if lv.Type == ValString && rv.Type == ValString {
				stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str}
			} else {
				lf, _ := valToFloat64(lv)
				rf, _ := valToFloat64(rv)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case OpAddGlobalGlobal:
			g1Idx := inst.Arg >> 16
			g2Idx := inst.Arg & 0xFFFF
			v1, _ := ctx.Get(consts[g1Idx].Str)
			v2, _ := ctx.Get(consts[g2Idx].Str)
			lv := FromInterface(v1)
			rv := FromInterface(v2)
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			if lv.Type == ValInt && rv.Type == ValInt {
				stack[sp] = Value{Type: ValInt, Num: lv.Num + rv.Num}
			} else if lv.Type == ValString && rv.Type == ValString {
				stack[sp] = Value{Type: ValString, Str: lv.Str + rv.Str}
			} else {
				lf, _ := valToFloat64(lv)
				rf, _ := valToFloat64(rv)
				stack[sp] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case OpEqualGlobalConst:
			gIdx := inst.Arg >> 16
			cIdx := inst.Arg & 0xFFFF
			val, _ := ctx.Get(consts[gIdx].Str)
			lv := FromInterface(val)
			r := consts[cIdx]
			res := false
			if lv.Type == r.Type {
				switch lv.Type {
				case ValInt, ValFloat, ValBool:
					res = lv.Num == r.Num
				case ValString:
					res = lv.Str == r.Str
				case ValNil:
					res = true
				}
			} else {
				lf, okL := valToFloat64(lv)
				rf, okR := valToFloat64(r)
				if okL && okR {
					res = lf == rf
				}
			}
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case OpGreaterGlobalConst:
			gIdx := inst.Arg >> 16
			cIdx := inst.Arg & 0xFFFF
			val, _ := ctx.Get(consts[gIdx].Str)
			lv := FromInterface(val)
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
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case OpLessGlobalConst:
			gIdx := inst.Arg >> 16
			cIdx := inst.Arg & 0xFFFF
			val, _ := ctx.Get(consts[gIdx].Str)
			lv := FromInterface(val)
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
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = Value{Type: ValBool, Num: boolToUint64(res)}
		case OpFusedCompareGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg>>22) & 0x3FF
			cIdx := int(inst.Arg>>12) & 0x3FF
			jTarget := int(inst.Arg) & 0xFFF
			val, _ := ctx.Get(consts[gIdx].Str)
			lv := FromInterface(val)
			r := consts[cIdx]
			res := false
			if lv.Type == r.Type {
				switch lv.Type {
				case ValInt, ValFloat, ValBool:
					res = lv.Num == r.Num
				case ValString:
					res = lv.Str == r.Str
				case ValNil:
					res = true
				}
			} else {
				lf, okL := valToFloat64(lv)
				rf, okR := valToFloat64(r)
				if okL && okR {
					res = lf == rf
				}
			}
			if !res {
				pc = jTarget
			}
		case OpGetGlobalJumpIfFalse:
			gIdx := inst.Arg >> 16
			jTarget := inst.Arg & 0xFFFF
			val, _ := ctx.Get(consts[gIdx].Str)
			if !isValTruthyAny(val) {
				pc = int(jTarget)
			}
		case OpGetGlobalJumpIfTrue:
			gIdx := inst.Arg >> 16
			jTarget := inst.Arg & 0xFFFF
			val, _ := ctx.Get(consts[gIdx].Str)
			if isValTruthyAny(val) {
				pc = int(jTarget)
			}
		case OpMapGet:
			key := stack[sp]
			sp--
			obj := stack[sp]
			if obj.Type != ValMap {
				return nil, fmt.Errorf("MGET: not a map")
			}
			if key.Type != ValString {
				return nil, fmt.Errorf("MGET: key must be string")
			}
			m := obj.Ptr.(map[string]any)
			stack[sp] = FromInterface(m[key.Str])
		case OpMapSet:
			val := stack[sp]
			sp--
			key := stack[sp]
			sp--
			obj := stack[sp]
			if obj.Type != ValMap {
				return nil, fmt.Errorf("MSET: not a map")
			}
			if key.Type != ValString {
				return nil, fmt.Errorf("MSET: key must be string")
			}
			m := obj.Ptr.(map[string]any)
			m[key.Str] = val.ToInterface()
			stack[sp] = Value{Type: ValNil}
		case OpMapHas:
			key := stack[sp]
			sp--
			obj := stack[sp]
			if obj.Type != ValMap {
				return nil, fmt.Errorf("MHAS: not a map")
			}
			if key.Type != ValString {
				return nil, fmt.Errorf("MHAS: key must be string")
			}
			m := obj.Ptr.(map[string]any)
			_, ok := m[key.Str]
			res := uint64(0)
			if ok {
				res = 1
			}
			stack[sp] = Value{Type: ValBool, Num: res}
		case OpMapGetConst:
			obj := &stack[sp]
			if obj.Type != ValMap { return nil, fmt.Errorf("MGETC: not a map") }
			m := obj.Ptr.(map[string]any)
			val := m[consts[inst.Arg].Str]
			switch v := val.(type) {
			case int64: *obj = Value{Type: ValInt, Num: uint64(v)}
			case int: *obj = Value{Type: ValInt, Num: uint64(int64(v))}
			case float64: *obj = Value{Type: ValFloat, Num: math.Float64bits(v)}
			case string: *obj = Value{Type: ValString, Str: v}
			case bool: *obj = Value{Type: ValBool, Num: boolToUint64(v)}
			case nil: *obj = Value{Type: ValNil}
			default: *obj = FromInterface(v)
			}
		case OpMapDel:
			key := stack[sp]
			sp--
			obj := stack[sp]
			if obj.Type != ValMap {
				return nil, fmt.Errorf("MDEL: not a map")
			}
			if key.Type != ValString {
				return nil, fmt.Errorf("MDEL: key must be string")
			}
			m := obj.Ptr.(map[string]any)
			delete(m, key.Str)
			stack[sp] = Value{Type: ValNil}
		case OpConcat:
			numArgs := int(inst.Arg)
			totalLen := 0
			var argStringsBuf [8]string
			var argStrings []string
			if numArgs <= 8 {
				argStrings = argStringsBuf[:numArgs]
			} else {
				argStrings = make([]string, numArgs)
			}
			for i := numArgs - 1; i >= 0; i-- {
				v := stack[sp]
				sp--
				var s string
				switch v.Type {
				case ValString:
					s = v.Str
				case ValInt:
					s = fmt.Sprintf("%d", int64(v.Num))
				case ValFloat:
					s = fmt.Sprintf("%g", math.Float64frombits(v.Num))
				case ValBool:
					if v.Num != 0 {
						s = "true"
					} else {
						s = "false"
					}
				default:
					s = fmt.Sprintf("%v", v.ToInterface())
				}
				argStrings[i] = s
				totalLen += len(s)
			}
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.Grow(totalLen)
			for _, s := range argStrings {
				buf.WriteString(s)
			}
			res := buf.String()
			bufferPool.Put(buf)
			sp++
			if sp >= 64 {
				return nil, fmt.Errorf("VM stack overflow")
			}
			stack[sp] = Value{Type: ValString, Str: res}
		}
	}
	if sp < 0 {
		return nil, nil
	}
	return stack[sp].ToInterface(), nil
}
func isValTruthy(v Value) bool {
	switch v.Type {
	case ValBool:
		return v.Num != 0
	case ValNil:
		return false
	default:
		return true
	}
}
