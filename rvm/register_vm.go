// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package rvm

import (
	"bytes"
	"fmt"
	"math"
	"github.com/kamihama-railway/uwasa/types"
)

func RunRegisterVM(bc *RegisterBytecode, ctx types.Context) (any, error) {
	if bc == nil || len(bc.Instructions) == 0 {
		return nil, nil
	}

	// Use a fixed size buffer that covers all possible uint8 register indices.
	// This ensures that single-register instructions (using uint8 indices)
	// can never trigger a Go panic for out-of-bounds access,
	// providing memory safety without per-instruction checks in the hot loop.
	var registers [256]types.Value
	regs := registers[:]

	pc := 0
	insts := bc.Instructions
	consts := bc.Constants
	nInsts := len(insts)

	mapCtx, isMapCtx := ctx.(*types.MapContext)

	for pc < nInsts {
		inst := insts[pc]
		pc++

		switch inst.Op {
		case ROpLoadConst:
			regs[inst.Dest] = consts[inst.Arg]

		case ROpGetGlobal:
			name := consts[inst.Arg].Str
			var val any
			if isMapCtx {
				val = mapCtx.Vars[name]
			} else {
				val, _ = ctx.Get(name)
			}
			regs[inst.Dest] = types.FromInterface(val)

		case ROpSetGlobal:
			name := consts[inst.Arg].Str
			val := regs[inst.Src1]
			if isMapCtx {
				mapCtx.Vars[name] = val.ToInterface()
			} else {
				ctx.Set(name, val.ToInterface())
			}

		case ROpMove:
			regs[inst.Dest] = regs[inst.Src1]

		case ROpAdd:
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

		case ROpSub:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num - r.Num}
			} else {
				lf, _ := types.ValToFloat64(l)
				rf, _ := types.ValToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf - rf)}
			}

		case ROpMul:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if l.Type == types.ValInt && r.Type == types.ValInt {
				regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num * r.Num}
			} else {
				lf, _ := types.ValToFloat64(l)
				rf, _ := types.ValToFloat64(r)
				regs[inst.Dest] = types.Value{Type: types.ValFloat, Num: math.Float64bits(lf * rf)}
			}

		case ROpDiv:
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

		case ROpMod:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if r.Type != types.ValInt {
				return nil, fmt.Errorf("modulo operator supports only integers")
			}
			if r.Num == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			regs[inst.Dest] = types.Value{Type: types.ValInt, Num: l.Num % r.Num}

		case ROpEqual:
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

		case ROpGreater:
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

		case ROpLess:
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

		case ROpGreaterEqual:
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

		case ROpLessEqual:
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

		case ROpAnd:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(types.IsValTruthy(l) && types.IsValTruthy(r))}

		case ROpOr:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(types.IsValTruthy(l) || types.IsValTruthy(r))}

		case ROpNot:
			l := regs[inst.Src1]
			regs[inst.Dest] = types.Value{Type: types.ValBool, Num: types.BoolToUint64(!types.IsValTruthy(l))}

		case ROpJump:
			pc = int(inst.Arg)

		case ROpJumpIfFalse:
			l := regs[inst.Src1]
			if !types.IsValTruthy(l) {
				pc = int(inst.Arg)
			}

		case ROpJumpIfTrue:
			l := regs[inst.Src1]
			if types.IsValTruthy(l) {
				pc = int(inst.Arg)
			}

		case ROpCall:
			name := consts[inst.Arg].Str
			numArgs := int(inst.Src2)
			argsStart := int(inst.Src1)

			if argsStart+numArgs > len(regs) {
				return nil, fmt.Errorf("register index out of bounds in CALL")
			}

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

		case ROpConcat:
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
			if argsStart+numArgs > len(regs) {
				return nil, fmt.Errorf("register index out of bounds in CONCAT")
			}
			for i := 0; i < numArgs; i++ {
				v := regs[argsStart+i]
				var s string
				switch v.Type {
				case types.ValString:
					s = v.Str
				case types.ValInt:
					s = fmt.Sprintf("%d", int64(v.Num))
				case types.ValFloat:
					s = fmt.Sprintf("%g", math.Float64frombits(v.Num))
				case types.ValBool:
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
			buf := types.BufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.Grow(totalLen)
			for _, s := range argStrings {
				buf.WriteString(s)
			}
			res := buf.String()
			types.BufferPool.Put(buf)
			regs[inst.Dest] = types.Value{Type: types.ValString, Str: res}

		case ROpReturn:
			return regs[inst.Src1].ToInterface(), nil
		}
	}

	return nil, nil
}
