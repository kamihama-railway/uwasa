// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"bytes"
	"fmt"
	"math"
)

func RunRegisterVM(bc *RegisterBytecode, ctx Context) (any, error) {
	if bc == nil || len(bc.Instructions) == 0 {
		return nil, nil
	}

	var registers [16]Value
	var regs []Value
	if bc.MaxRegisters <= 16 {
		regs = registers[:bc.MaxRegisters]
	} else {
		regs = make([]Value, bc.MaxRegisters)
	}

	pc := 0
	insts := bc.Instructions
	consts := bc.Constants
	nInsts := len(insts)

	mapCtx, isMapCtx := ctx.(*MapContext)

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
				val = mapCtx.vars[name]
			} else {
				val, _ = ctx.Get(name)
			}
			regs[inst.Dest] = FromInterface(val)

		case ROpSetGlobal:
			name := consts[inst.Arg].Str
			val := regs[inst.Src1]
			if isMapCtx {
				mapCtx.vars[name] = val.ToInterface()
			} else {
				ctx.Set(name, val.ToInterface())
			}

		case ROpMove:
			regs[inst.Dest] = regs[inst.Src1]

		case ROpAdd:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if l.Type == ValInt && r.Type == ValInt {
				regs[inst.Dest] = Value{Type: ValInt, Num: l.Num + r.Num}
			} else if l.Type == ValString && r.Type == ValString {
				regs[inst.Dest] = Value{Type: ValString, Str: l.Str + r.Str}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				regs[inst.Dest] = Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}
			}

		case ROpSub:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if l.Type == ValInt && r.Type == ValInt {
				regs[inst.Dest] = Value{Type: ValInt, Num: l.Num - r.Num}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				regs[inst.Dest] = Value{Type: ValFloat, Num: math.Float64bits(lf - rf)}
			}

		case ROpMul:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if l.Type == ValInt && r.Type == ValInt {
				regs[inst.Dest] = Value{Type: ValInt, Num: l.Num * r.Num}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				regs[inst.Dest] = Value{Type: ValFloat, Num: math.Float64bits(lf * rf)}
			}

		case ROpDiv:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if r.Type == ValInt && r.Num == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			if r.Type == ValFloat && math.Float64frombits(r.Num) == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			if l.Type == ValInt && r.Type == ValInt {
				regs[inst.Dest] = Value{Type: ValInt, Num: l.Num / r.Num}
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				regs[inst.Dest] = Value{Type: ValFloat, Num: math.Float64bits(lf / rf)}
			}

		case ROpMod:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			if r.Type != ValInt {
				return nil, fmt.Errorf("modulo operator supports only integers")
			}
			if r.Num == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			regs[inst.Dest] = Value{Type: ValInt, Num: l.Num % r.Num}

		case ROpEqual:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
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
			regs[inst.Dest] = Value{Type: ValBool, Num: boolToUint64(res)}

		case ROpGreater:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) > int64(r.Num)
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf > rf
			}
			regs[inst.Dest] = Value{Type: ValBool, Num: boolToUint64(res)}

		case ROpLess:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) < int64(r.Num)
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf < rf
			}
			regs[inst.Dest] = Value{Type: ValBool, Num: boolToUint64(res)}

		case ROpGreaterEqual:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) >= int64(r.Num)
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf >= rf
			}
			regs[inst.Dest] = Value{Type: ValBool, Num: boolToUint64(res)}

		case ROpLessEqual:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			res := false
			if l.Type == ValInt && r.Type == ValInt {
				res = int64(l.Num) <= int64(r.Num)
			} else {
				lf, _ := valToFloat64(l)
				rf, _ := valToFloat64(r)
				res = lf <= rf
			}
			regs[inst.Dest] = Value{Type: ValBool, Num: boolToUint64(res)}

		case ROpAnd:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			regs[inst.Dest] = Value{Type: ValBool, Num: boolToUint64(isValTruthy(l) && isValTruthy(r))}

		case ROpOr:
			l := regs[inst.Src1]
			r := regs[inst.Src2]
			regs[inst.Dest] = Value{Type: ValBool, Num: boolToUint64(isValTruthy(l) || isValTruthy(r))}

		case ROpNot:
			l := regs[inst.Src1]
			regs[inst.Dest] = Value{Type: ValBool, Num: boolToUint64(!isValTruthy(l))}

		case ROpJump:
			pc = int(inst.Arg)

		case ROpJumpIfFalse:
			l := regs[inst.Src1]
			if !isValTruthy(l) {
				pc = int(inst.Arg)
			}

		case ROpJumpIfTrue:
			l := regs[inst.Src1]
			if isValTruthy(l) {
				pc = int(inst.Arg)
			}

		case ROpCall:
			name := consts[inst.Arg].Str
			numArgs := int(inst.Src2)
			argsStart := int(inst.Src1)

			args := make([]any, numArgs)
			for i := 0; i < numArgs; i++ {
				args[i] = regs[argsStart+i].ToInterface()
			}

			if builtin, ok := builtins[name]; ok {
				res, err := builtin(args...)
				if err != nil {
					return nil, err
				}
				regs[inst.Dest] = FromInterface(res)
			} else {
				return nil, fmt.Errorf("builtin function not found: %s", name)
			}

		case ROpConcat:
			numArgs := int(inst.Src2)
			argsStart := int(inst.Src1)
			totalLen := 0
			argStrings := make([]string, numArgs)
			for i := 0; i < numArgs; i++ {
				v := regs[argsStart+i]
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
			regs[inst.Dest] = Value{Type: ValString, Str: res}

		case ROpReturn:
			return regs[inst.Src1].ToInterface(), nil
		}
	}

	return nil, nil
}
