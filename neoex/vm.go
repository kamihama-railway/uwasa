package neoex

import (
	"bytes"
	"fmt"
	"math"
	"github.com/kamihama-railway/uwasa/types"
)

func Run[C types.Context](bc *Bytecode, ctx C) (any, error) {
	if bc == nil || len(bc.Instructions) == 0 {
		return nil, nil
	}

	if mapCtx, ok := any(ctx).(*types.MapContext); ok {
		return runMapped(bc, mapCtx)
	}
	return runGeneral(bc, ctx)
}

func runMapped(bc *Bytecode, ctx *types.MapContext) (any, error) {
	var regs [256]types.Value
	pc := 0
	insts := bc.Instructions
	consts := bc.Constants
	nInsts := len(insts)
	vars := ctx.Vars

	for pc < nInsts {
		inst := insts[pc]
		pc++

		switch inst.Op {
		case OpLoadConst:
			regs[inst.Dest] = consts[inst.Arg]
		case OpGetGlobal:
			val := vars[consts[inst.Arg].Str]
			switch v := val.(type) {
			case int64: regs[inst.Dest] = types.Value{Type: types.ValInt, Num: uint64(v)}
			case float64: regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(v)}
			case int: regs[inst.Dest] = types.Value{Type: types.ValInt, Num: uint64(v)}
			case bool:
				num := uint64(0)
				if v { num = 1 }
				regs[inst.Dest] = types.Value{Type: types.ValBool, Num: num}
			case string: regs[inst.Dest] = types.Value{Type: types.ValString, Str: v}
			case nil: regs[inst.Dest] = types.Value{Type: types.ValNil}
			default: regs[inst.Dest] = types.FromInterface(v)
			}
		case OpSetGlobal:
			vars[consts[inst.Arg].Str] = regs[inst.Src1].ToInterface()
		case OpAdd:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num + r.Num}
			} else if l.Type == types.ValString && r.Type == types.ValString {
				regs[inst.Dest] = types.Value{Type: types.ValString, Str: l.Str + r.Str}
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case OpSub:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num - r.Num}
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf - rf)}
			}
		case OpMul:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num * r.Num}
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf * rf)}
			}
		case OpDiv:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			if r.Num == 0 { return nil, fmt.Errorf("division by zero") }
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num / r.Num}
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf / rf)}
			}
		case OpMod:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			if r.Type != types.ValInt || r.Num == 0 { return nil, fmt.Errorf("invalid modulo") }
			regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num % r.Num}
		case OpEqual:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			res := false
			if l.Type == r.Type {
				if l.Type == types.ValString { res = l.Str == r.Str } else { res = l.Num == r.Num }
			} else {
				lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpGreater:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) > int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf > rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpLess:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) < int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf < rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpGreaterEqual:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) >= int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf >= rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpLessEqual:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) <= int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf <= rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpAnd:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(isValTruthy(l) && isValTruthy(r))}
		case OpOr:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(isValTruthy(l) || isValTruthy(r))}
		case OpNot:
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(!isValTruthy(regs[inst.Src1]))}
		case OpJump:
			pc = int(inst.Arg)
		case OpJumpIfFalse:
			if !isValTruthy(regs[inst.Src1]) { pc = int(inst.Arg) }
		case OpJumpIfTrue:
			if isValTruthy(regs[inst.Src1]) { pc = int(inst.Arg) }
		case OpCall:
			name := consts[inst.Arg].Str; numArgs := int(inst.Src2); argsStart := int(inst.Src1)
			args := make([]any, numArgs)
			for i := 0; i < numArgs; i++ { args[i] = regs[argsStart+i].ToInterface() }
			if builtin, ok := types.Builtins[name]; ok {
				res, err := builtin(args...)
				if err != nil { return nil, err }
				regs[inst.Dest] = fromInterface(res)
			} else {
				return nil, fmt.Errorf("builtin function not found: %s", name)
			}
		case OpConcat:
			numArgs := int(inst.Src2); argsStart := int(inst.Src1)
			totalLen := 0
			var argStringsBuf [8]string
			var argStrings []string
			if numArgs <= 8 { argStrings = argStringsBuf[:numArgs] } else { argStrings = make([]string, numArgs) }
			for i := 0; i < numArgs; i++ {
				v := regs[argsStart+i]; var s string
				switch v.Type {
				case types.ValString: s = v.Str
				case types.ValInt: s = fmt.Sprintf("%d", int64(v.Num))
				case types.ValFloat: s = fmt.Sprintf("%g", math.Float64frombits(v.Num))
				case types.ValBool:
					if v.Num != 0 { s = "true" } else { s = "false" }
				default: s = fmt.Sprintf("%v", v.ToInterface())
				}
				argStrings[i] = s; totalLen += len(s)
			}
			buf := types.BufferPool.Get().(*bytes.Buffer)
			buf.Reset(); buf.Grow(totalLen)
			for _, s := range argStrings { buf.WriteString(s) }
			res := buf.String(); types.BufferPool.Put(buf)
			regs[inst.Dest] = types.Value{Type: types.ValString, Str: res}
		case OpEqualGlobalConst:
			name := consts[inst.Arg>>16].Str
			lv := fromInterface(vars[name])
			r := consts[inst.Arg&0xFFFF]
			res := false
			if lv.Type == r.Type {
				if lv.Type == types.ValString { res = lv.Str == r.Str } else { res = lv.Num == r.Num }
			} else {
				lf, okL := valToFloat64(lv); rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpAddGlobalConst:
			name := consts[inst.Arg>>16].Str
			lv := fromInterface(vars[name])
			rv := consts[inst.Arg&0xFFFF]
			if lv.Type == types.ValInt && rv.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: lv.Num + rv.Num}
			} else {
				lf, _ := valToFloat64(lv); rf, _ := valToFloat64(rv)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case OpGetGlobalJumpIfFalse:
			val := vars[consts[inst.Arg>>16].Str]
			var truthy bool
			switch v := val.(type) {
			case bool: truthy = v
			case nil: truthy = false
			default: truthy = true
			}
			if !truthy { pc = int(inst.Arg & 0xFFFF) }
		case OpFusedCompareGlobalConstJumpIfFalse:
			gIdx := int(inst.Arg >> 22) & 0x3FF
			cIdx := int(inst.Arg >> 12) & 0x3FF
			lv := fromInterface(vars[consts[gIdx].Str])
			r := consts[cIdx]; res := false
			if lv.Type == r.Type {
				if lv.Type == types.ValString { res = lv.Str == r.Str } else { res = lv.Num == r.Num }
			} else {
				lf, okL := valToFloat64(lv); rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			if !res { pc = int(inst.Arg & 0xFFF) }
		case OpAddGlobalGlobal:
			v1 := fromInterface(vars[consts[inst.Arg>>16].Str])
			v2 := fromInterface(vars[consts[inst.Arg&0xFFFF].Str])
			if v1.Type == types.ValInt && v2.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: v1.Num + v2.Num}
			} else {
				lf, _ := valToFloat64(v1); rf, _ := valToFloat64(v2)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case OpSubGlobalGlobal:
			v1 := fromInterface(vars[consts[inst.Arg>>16].Str])
			v2 := fromInterface(vars[consts[inst.Arg&0xFFFF].Str])
			if v1.Type == types.ValInt && v2.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: v1.Num - v2.Num}
			} else {
				lf, _ := valToFloat64(v1); rf, _ := valToFloat64(v2)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf - rf)}
			}
		case OpReturn:
			return regs[inst.Src1].ToInterface(), nil
		}
	}
	return nil, nil
}

func runGeneral(bc *Bytecode, ctx types.Context) (any, error) {
	var regs [256]types.Value
	pc := 0
	insts := bc.Instructions
	consts := bc.Constants
	nInsts := len(insts)

	for pc < nInsts {
		inst := insts[pc]
		pc++

		switch inst.Op {
		case OpLoadConst:
			regs[inst.Dest] = consts[inst.Arg]
		case OpGetGlobal:
			val, _ := ctx.Get(consts[inst.Arg].Str)
			regs[inst.Dest] = fromInterface(val)
		case OpSetGlobal:
			ctx.Set(consts[inst.Arg].Str, regs[inst.Src1].ToInterface())
		case OpAdd:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num + r.Num}
			} else if l.Type == types.ValString && r.Type == types.ValString {
				regs[inst.Dest] = types.Value{Type: types.ValString, Str: l.Str + r.Str}
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case OpSub:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num - r.Num}
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf - rf)}
			}
		case OpMul:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num * r.Num}
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf * rf)}
			}
		case OpDiv:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			if r.Num == 0 { return nil, fmt.Errorf("division by zero") }
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num / r.Num}
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf / rf)}
			}
		case OpMod:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			if r.Type != types.ValInt || r.Num == 0 { return nil, fmt.Errorf("invalid modulo") }
			regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num % r.Num}
		case OpEqual:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			res := false
			if l.Type == r.Type {
				if l.Type == types.ValString { res = l.Str == r.Str } else { res = l.Num == r.Num }
			} else {
				lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpGreater:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) > int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf > rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpLess:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) < int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf < rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpGreaterEqual:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) >= int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf >= rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpLessEqual:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) <= int64(r.Num)
			} else {
				lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
				res = lf <= rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpAnd:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(isValTruthy(l) && isValTruthy(r))}
		case OpOr:
			l := regs[inst.Src1]; r := regs[inst.Src2]
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(isValTruthy(l) || isValTruthy(r))}
		case OpNot:
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(!isValTruthy(regs[inst.Src1]))}
		case OpJump:
			pc = int(inst.Arg)
		case OpJumpIfFalse:
			if !isValTruthy(regs[inst.Src1]) { pc = int(inst.Arg) }
		case OpJumpIfTrue:
			if isValTruthy(regs[inst.Src1]) { pc = int(inst.Arg) }
		case OpCall:
			name := consts[inst.Arg].Str; numArgs := int(inst.Src2); argsStart := int(inst.Src1)
			args := make([]any, numArgs)
			for i := 0; i < numArgs; i++ { args[i] = regs[argsStart+i].ToInterface() }
			if builtin, ok := types.Builtins[name]; ok {
				res, err := builtin(args...)
				if err != nil { return nil, err }
				regs[inst.Dest] = fromInterface(res)
			} else {
				return nil, fmt.Errorf("builtin function not found: %s", name)
			}
		case OpConcat:
			numArgs := int(inst.Src2); argsStart := int(inst.Src1)
			totalLen := 0
			var argStringsBuf [8]string
			var argStrings []string
			if numArgs <= 8 { argStrings = argStringsBuf[:numArgs] } else { argStrings = make([]string, numArgs) }
			for i := 0; i < numArgs; i++ {
				v := regs[argsStart+i]; var s string
				switch v.Type {
				case types.ValString: s = v.Str
				case types.ValInt: s = fmt.Sprintf("%d", int64(v.Num))
				case types.ValFloat: s = fmt.Sprintf("%g", math.Float64frombits(v.Num))
				case types.ValBool:
					if v.Num != 0 { s = "true" } else { s = "false" }
				default: s = fmt.Sprintf("%v", v.ToInterface())
				}
				argStrings[i] = s; totalLen += len(s)
			}
			buf := types.BufferPool.Get().(*bytes.Buffer)
			buf.Reset(); buf.Grow(totalLen)
			for _, s := range argStrings { buf.WriteString(s) }
			res := buf.String(); types.BufferPool.Put(buf)
			regs[inst.Dest] = types.Value{Type: types.ValString, Str: res}
		case OpEqualGlobalConst:
			val, _ := ctx.Get(consts[inst.Arg>>16].Str); lv := fromInterface(val)
			r := consts[inst.Arg&0xFFFF]
			res := false
			if lv.Type == r.Type {
				if lv.Type == types.ValString { res = lv.Str == r.Str } else { res = lv.Num == r.Num }
			} else {
				lf, okL := valToFloat64(lv); rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: boolToUint64(res)}
		case OpAddGlobalConst:
			val, _ := ctx.Get(consts[inst.Arg>>16].Str); lv := fromInterface(val)
			rv := consts[inst.Arg&0xFFFF]
			if lv.Type == types.ValInt && rv.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: lv.Num + rv.Num}
			} else {
				lf, _ := valToFloat64(lv); rf, _ := valToFloat64(rv)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case OpGetGlobalJumpIfFalse:
			val, _ := ctx.Get(consts[inst.Arg>>16].Str)
			if !isValTruthy(fromInterface(val)) { pc = int(inst.Arg & 0xFFFF) }
		case OpFusedCompareGlobalConstJumpIfFalse:
			val, _ := ctx.Get(consts[int(inst.Arg>>22)&0x3FF].Str); lv := fromInterface(val)
			r := consts[int(inst.Arg>>12)&0x3FF]; res := false
			if lv.Type == r.Type {
				if lv.Type == types.ValString { res = lv.Str == r.Str } else { res = lv.Num == r.Num }
			} else {
				lf, okL := valToFloat64(lv); rf, okR := valToFloat64(r)
				if okL && okR { res = lf == rf }
			}
			if !res { pc = int(inst.Arg & 0xFFF) }
		case OpAddGlobalGlobal:
			v1Raw, _ := ctx.Get(consts[inst.Arg>>16].Str); v1 := fromInterface(v1Raw)
			v2Raw, _ := ctx.Get(consts[inst.Arg&0xFFFF].Str); v2 := fromInterface(v2Raw)
			if v1.Type == types.ValInt && v2.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: v1.Num + v2.Num}
			} else {
				lf, _ := valToFloat64(v1); rf, _ := valToFloat64(v2)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf + rf)}
			}
		case OpSubGlobalGlobal:
			v1Raw, _ := ctx.Get(consts[inst.Arg>>16].Str); v1 := fromInterface(v1Raw)
			v2Raw, _ := ctx.Get(consts[inst.Arg&0xFFFF].Str); v2 := fromInterface(v2Raw)
			if v1.Type == types.ValInt && v2.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: v1.Num - v2.Num}
			} else {
				lf, _ := valToFloat64(v1); rf, _ := valToFloat64(v2)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf - rf)}
			}
		case OpReturn:
			return regs[inst.Src1].ToInterface(), nil
		}
	}
	return nil, nil
}

func valToFloat64(v types.Value) (float64, bool) {
	switch v.Type {
	case types.ValFloat: return math.Float64frombits(v.Num), true
	case types.ValInt: return float64(int64(v.Num)), true
	}
	return 0, false
}

func isValTruthy(v types.Value) bool {
	switch v.Type {
	case types.ValBool: return v.Num != 0
	case types.ValNil: return false
	default: return true
	}
}

func boolToUint64(b bool) uint64 {
	if b { return 1 }
	return 0
}

func fromInterface(v any) types.Value {
	switch val := v.(type) {
	case int64:
		return types.Value{Type: types.ValInt, Num: uint64(val)}
	case int:
		return types.Value{Type: types.ValInt, Num: uint64(val)}
	case float64:
		return types.Value{Type: types.ValFloat, Num: math.Float64bits(val)}
	case bool:
		if val {
			return types.Value{Type: types.ValBool, Num: 1}
		}
		return types.Value{Type: types.ValBool, Num: 0}
	case string:
		return types.Value{Type: types.ValString, Str: val}
	default:
		return types.Value{Type: types.ValNil}
	}
}
