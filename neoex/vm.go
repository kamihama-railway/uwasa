package neoex

import (
	"bytes"
	"fmt"
	"math"
	"github.com/kamihama-railway/uwasa/types"
)

// Run 采用泛型模板，可以针对特定的 Context 类型进行内联优化
func Run[C types.Context](bc *Bytecode, ctx C) (any, error) {
	if bc == nil || len(bc.Instructions) == 0 {
		return nil, nil
	}

	var registers [256]types.Value
	regs := registers[:]

	pc := 0
	insts := bc.Instructions
	consts := bc.Constants
	nInsts := len(insts)

	// 针对 MapContext 的快速路径优化
	mapCtx, isMapCtx := any(ctx).(*types.MapContext)

	for pc < nInsts {
		inst := insts[pc]
		pc++

		switch inst.Op {
		case OpLoadConst:
			regs[inst.Dest] = consts[inst.Arg]

		case OpGetGlobal:
			name := consts[inst.Arg].Str
			var val any
			if isMapCtx {
				val = mapCtx.Vars[name]
			} else {
				val, _ = ctx.Get(name)
			}
			regs[inst.Dest] = types.FromInterface(val)

		case OpSetGlobal:
			name := consts[inst.Arg].Str
			val := regs[inst.Src1]
			if isMapCtx {
				mapCtx.Vars[name] = val.ToInterface()
			} else {
				ctx.Set(name, val.ToInterface())
			}

		case OpAdd:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num + r.Num}
			} else if l.Type == types.ValString && r.Type == types.ValString {
				regs[inst.Dest] = types.Value{Type: types.ValString, Str: l.Str + r.Str}
			} else {
				lf, _ := types.ValToFloat64(l)
				rf, _ := types.ValToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf + rf)}
			}

		case OpSub:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num - r.Num}
			} else {
				lf, _ := types.ValToFloat64(l)
				rf, _ := types.ValToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf - rf)}
			}

		case OpMul:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num * r.Num}
			} else {
				lf, _ := types.ValToFloat64(l)
				rf, _ := types.ValToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf * rf)}
			}

		case OpDiv:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if r.Type == types.ValInt && r.Num == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			if r.Type == types.ValFloat && math.Float64frombits(r.Num) == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num / r.Num}
			} else {
				lf, _ := types.ValToFloat64(l)
				rf, _ := types.ValToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf / rf)}
			}

		case OpMod:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if r.Type != types.ValInt {
				return nil, fmt.Errorf("modulo operator supports only integers")
			}
			if r.Num == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num % r.Num}

		case OpEqual:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			res := false
			if l.Type == r.Type {
				switch l.Type {
				case types.ValInt, types.ValFloat, types.ValBool:
					res = l.Num == r.Num
				case types.ValString:
					res = l.Str == r.Str
				case types.ValNil:
					res = true
				}
			} else {
				lf, okL := types.ValToFloat64(l)
				rf, okR := types.ValToFloat64(r)
				if okL && okR {
					res = lf == rf
				}
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(res)}

		case OpGreater:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) > int64(r.Num)
			} else {
				lf, _ := types.ValToFloat64(l)
				rf, _ := types.ValToFloat64(r)
				res = lf > rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(res)}

		case OpLess:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) < int64(r.Num)
			} else {
				lf, _ := types.ValToFloat64(l)
				rf, _ := types.ValToFloat64(r)
				res = lf < rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(res)}

		case OpGreaterEqual:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) >= int64(r.Num)
			} else {
				lf, _ := types.ValToFloat64(l)
				rf, _ := types.ValToFloat64(r)
				res = lf >= rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(res)}

		case OpLessEqual:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			res := false
			if l.Type == types.ValInt && r.Type == types.ValInt {
				res = int64(l.Num) <= int64(r.Num)
			} else {
				lf, _ := types.ValToFloat64(l)
				rf, _ := types.ValToFloat64(r)
				res = lf <= rf
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(res)}

		case OpAnd:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(types.IsValTruthy(l) && types.IsValTruthy(r))}

		case OpOr:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(types.IsValTruthy(l) || types.IsValTruthy(r))}

		case OpNot:
			l := regs[inst.Src1]
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(!types.IsValTruthy(l))}

		case OpJump:
			pc = int(inst.Arg)

		case OpJumpIfFalse:
			l := regs[inst.Src1]
			if !types.IsValTruthy(l) {
				pc = int(inst.Arg)
			}

		case OpJumpIfTrue:
			l := regs[inst.Src1]
			if types.IsValTruthy(l) {
				pc = int(inst.Arg)
			}

		case OpCall:
			name := consts[inst.Arg].Str
			numArgs := int(inst.Src2)
			argsStart := int(inst.Src1)

			args := make([]any, numArgs)
			for i := 0; i < numArgs; i++ {
				args[i] = regs[argsStart+i].ToInterface()
			}

			if builtin, ok := types.Builtins[name]; ok {
				res, err := builtin(args...)
				if err != nil {
					return nil, err
				}
				regs[inst.Dest] = types.FromInterface(res)
			} else {
				return nil, fmt.Errorf("builtin function not found: %s", name)
			}

		case OpConcat:
			numArgs := int(inst.Src2)
			argsStart := int(inst.Src1)
			totalLen := 0
			var argStringsBuf [8]string
			var argStrings []string
			if numArgs <= 8 {
				argStrings = argStringsBuf[:numArgs]
			} else {
				argStrings = make([]string, numArgs)
			}
			for i := 0; i < numArgs; i++ {
				v := regs[argsStart+i]
				var s string
				switch v.Type {
				case types.ValString: s = v.Str
				case types.ValInt: s = fmt.Sprintf("%d", int64(v.Num))
				case types.ValFloat: s = fmt.Sprintf("%g", math.Float64frombits(v.Num))
				case types.ValBool:
					if v.Num != 0 { s = "true" } else { s = "false" }
				default: s = fmt.Sprintf("%v", v.ToInterface())
				}
				argStrings[i] = s
				totalLen += len(s)
			}
			buf := types.BufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.Grow(totalLen)
			for _, s := range argStrings {
				buf.WriteString(s)
			}
			res := buf.String()
			types.BufferPool.Put(buf)
			regs[inst.Dest] = types.Value{Type: types.ValString, Str: res}

		case OpEqualGlobalConst:
			name := consts[inst.Arg>>16].Str
			var lv types.Value
			if isMapCtx {
				lv = types.FromInterface(mapCtx.Vars[name])
			} else {
				v, _ := ctx.Get(name)
				lv = types.FromInterface(v)
			}
			r := consts[inst.Arg&0xFFFF]
			res := false
			if lv.Type == r.Type {
				switch lv.Type {
				case types.ValInt, types.ValFloat, types.ValBool:
					res = lv.Num == r.Num
				case types.ValString:
					res = lv.Str == r.Str
				case types.ValNil:
					res = true
				}
			} else {
				lf, okL := types.ValToFloat64(lv)
				rf, okR := types.ValToFloat64(r)
				if okL && okR {
					res = lf == rf
				}
			}
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(res)}

		case OpAddGlobalConst:
			name := consts[inst.Arg>>16].Str
			var lv types.Value
			if isMapCtx {
				lv = types.FromInterface(mapCtx.Vars[name])
			} else {
				v, _ := ctx.Get(name)
				lv = types.FromInterface(v)
			}
			rv := consts[inst.Arg&0xFFFF]
			if lv.Type == types.ValInt && rv.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: lv.Num + rv.Num}
			} else {
				lf, _ := types.ValToFloat64(lv)
				rf, _ := types.ValToFloat64(rv)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf + rf)}
			}

		case OpGetGlobalJumpIfFalse:
			name := consts[inst.Arg>>16].Str
			var lv types.Value
			if isMapCtx {
				lv = types.FromInterface(mapCtx.Vars[name])
			} else {
				v, _ := ctx.Get(name)
				lv = types.FromInterface(v)
			}
			if !types.IsValTruthy(lv) {
				pc = int(inst.Arg & 0xFFFF)
			}

		case OpReturn:
			return regs[inst.Src1].ToInterface(), nil
		}
	}

	return nil, nil
}
