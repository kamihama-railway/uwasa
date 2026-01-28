// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"math"
	"sync"
)

const StackSize = 2048

type valKind byte

const (
	valNil valKind = iota
	valInt
	valFloat
	valBool
	valString
	valAny
)

type Value struct {
	kind valKind
	num  uint64
	ptr  any
}

func (v Value) ToAny() any {
	switch v.kind {
	case valInt:
		return int64(v.num)
	case valFloat:
		return math.Float64frombits(v.num)
	case valBool:
		return v.num != 0
	case valString:
		return v.ptr.(string)
	case valAny:
		return v.ptr
	default:
		return nil
	}
}

func FromAny(a any) Value {
	switch v := a.(type) {
	case int64:
		return Value{kind: valInt, num: uint64(v)}
	case int:
		return Value{kind: valInt, num: uint64(int64(v))}
	case float64:
		return Value{kind: valFloat, num: math.Float64bits(v)}
	case bool:
		if v {
			return Value{kind: valBool, num: 1}
		}
		return Value{kind: valBool, num: 0}
	case string:
		return Value{kind: valString, ptr: v}
	case nil:
		return Value{kind: valNil}
	default:
		return Value{kind: valAny, ptr: v}
	}
}

func (v Value) toFloat() (float64, bool) {
	if v.kind == valFloat {
		return math.Float64frombits(v.num), true
	}
	if v.kind == valInt {
		return float64(int64(v.num)), true
	}
	return 0, false
}

type vmInstruction struct {
	op  OpCode
	arg uint16
}

type RenderedBytecode struct {
	Instructions []vmInstruction
	Constants    []Value
}

type VM struct {
	constants    []Value
	instructions []vmInstruction
	stack        []Value
	sp           int // stack pointer
}

var vmPool = sync.Pool{
	New: func() any {
		return &VM{
			stack: make([]Value, StackSize),
		}
	},
}

func NewVM(rendered *RenderedBytecode) *VM {
	vm := vmPool.Get().(*VM)
	vm.constants = rendered.Constants
	vm.instructions = rendered.Instructions
	vm.sp = 0
	return vm
}

func (vm *VM) Free() {
	// Clear stack pointers to help GC
	for i := 0; i < vm.sp; i++ {
		vm.stack[i].ptr = nil
	}
	vm.constants = nil
	vm.instructions = nil
	vmPool.Put(vm)
}

func (vm *VM) Run(ctx Context) (any, error) {
	ins := vm.instructions
	consts := vm.constants
	stack := vm.stack
	sp := vm.sp
	ip := 0

	for ip < len(ins) {
		inst := ins[ip]
		op := inst.op

		switch op {
		case OpConstant:
			if sp >= StackSize { return nil, fmt.Errorf("stack overflow") }
			stack[sp] = consts[inst.arg]
			sp++

		case OpPop:
			if sp > 0 {
				sp--
			}

		case OpAdd:
			r := stack[sp-1]
			l := stack[sp-2]
			sp--
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) + int64(r.num))}
			} else if l.kind == valFloat && r.kind == valFloat {
				stack[sp-1] = Value{kind: valFloat, num: math.Float64bits(math.Float64frombits(l.num) + math.Float64frombits(r.num))}
			} else if l.kind == valString && r.kind == valString {
				stack[sp-1] = Value{kind: valString, ptr: l.ptr.(string) + r.ptr.(string)}
			} else {
				res, err := evalArithmetic("+", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpSub:
			r := stack[sp-1]
			l := stack[sp-2]
			sp--
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) - int64(r.num))}
			} else if l.kind == valFloat && r.kind == valFloat {
				stack[sp-1] = Value{kind: valFloat, num: math.Float64bits(math.Float64frombits(l.num) - math.Float64frombits(r.num))}
			} else {
				res, err := evalArithmetic("-", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpMul:
			r := stack[sp-1]
			l := stack[sp-2]
			sp--
			if l.kind == valInt && r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) * int64(r.num))}
			} else if l.kind == valFloat && r.kind == valFloat {
				stack[sp-1] = Value{kind: valFloat, num: math.Float64bits(math.Float64frombits(l.num) * math.Float64frombits(r.num))}
			} else {
				res, err := evalArithmetic("*", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpDiv:
			r := stack[sp-1]
			l := stack[sp-2]
			sp--
			if l.kind == valInt && r.kind == valInt {
				if r.num == 0 { return nil, fmt.Errorf("division by zero") }
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) / int64(r.num))}
			} else if l.kind == valFloat && r.kind == valFloat {
				if r.num == 0 { return nil, fmt.Errorf("division by zero") }
				stack[sp-1] = Value{kind: valFloat, num: math.Float64bits(math.Float64frombits(l.num) / math.Float64frombits(r.num))}
			} else {
				res, err := evalArithmetic("/", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpMod:
			r := stack[sp-1]
			l := stack[sp-2]
			sp--
			if l.kind == valInt && r.kind == valInt {
				if r.num == 0 { return nil, fmt.Errorf("division by zero") }
				stack[sp-1] = Value{kind: valInt, num: uint64(int64(l.num) % int64(r.num))}
			} else {
				res, err := evalArithmetic("%", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpEqual:
			r := stack[sp-1]
			l := stack[sp-2]
			sp--
			if l.kind == r.kind && l.num == r.num && l.ptr == r.ptr {
				stack[sp-1] = Value{kind: valBool, num: 1}
			} else if fl, okL := l.toFloat(); okL {
				if fr, okR := r.toFloat(); okR {
					if fl == fr {
						stack[sp-1] = Value{kind: valBool, num: 1}
					} else {
						stack[sp-1] = Value{kind: valBool, num: 0}
					}
				} else {
					res, err := evalComparison("==", l.ToAny(), r.ToAny())
					if err != nil { return nil, err }
					stack[sp-1] = FromAny(res)
				}
			} else {
				res, err := evalComparison("==", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpGreater:
			r := stack[sp-1]
			l := stack[sp-2]
			sp--
			if l.kind == valInt && r.kind == valInt {
				if int64(l.num) > int64(r.num) {
					stack[sp-1] = Value{kind: valBool, num: 1}
				} else {
					stack[sp-1] = Value{kind: valBool, num: 0}
				}
			} else if fl, okL := l.toFloat(); okL {
				if fr, okR := r.toFloat(); okR {
					if fl > fr {
						stack[sp-1] = Value{kind: valBool, num: 1}
					} else {
						stack[sp-1] = Value{kind: valBool, num: 0}
					}
				} else {
					res, err := evalComparison(">", l.ToAny(), r.ToAny())
					if err != nil { return nil, err }
					stack[sp-1] = FromAny(res)
				}
			} else {
				res, err := evalComparison(">", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpLess:
			r := stack[sp-1]
			l := stack[sp-2]
			sp--
			if l.kind == valInt && r.kind == valInt {
				if int64(l.num) < int64(r.num) {
					stack[sp-1] = Value{kind: valBool, num: 1}
				} else {
					stack[sp-1] = Value{kind: valBool, num: 0}
				}
			} else if fl, okL := l.toFloat(); okL {
				if fr, okR := r.toFloat(); okR {
					if fl < fr {
						stack[sp-1] = Value{kind: valBool, num: 1}
					} else {
						stack[sp-1] = Value{kind: valBool, num: 0}
					}
				} else {
					res, err := evalComparison("<", l.ToAny(), r.ToAny())
					if err != nil { return nil, err }
					stack[sp-1] = FromAny(res)
				}
			} else {
				res, err := evalComparison("<", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpGreaterEqual:
			r := stack[sp-1]
			l := stack[sp-2]
			sp--
			if l.kind == valInt && r.kind == valInt {
				if int64(l.num) >= int64(r.num) {
					stack[sp-1] = Value{kind: valBool, num: 1}
				} else {
					stack[sp-1] = Value{kind: valBool, num: 0}
				}
			} else if fl, okL := l.toFloat(); okL {
				if fr, okR := r.toFloat(); okR {
					if fl >= fr {
						stack[sp-1] = Value{kind: valBool, num: 1}
					} else {
						stack[sp-1] = Value{kind: valBool, num: 0}
					}
				} else {
					res, err := evalComparison(">=", l.ToAny(), r.ToAny())
					if err != nil { return nil, err }
					stack[sp-1] = FromAny(res)
				}
			} else {
				res, err := evalComparison(">=", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpLessEqual:
			r := stack[sp-1]
			l := stack[sp-2]
			sp--
			if l.kind == valInt && r.kind == valInt {
				if int64(l.num) <= int64(r.num) {
					stack[sp-1] = Value{kind: valBool, num: 1}
				} else {
					stack[sp-1] = Value{kind: valBool, num: 0}
				}
			} else if fl, okL := l.toFloat(); okL {
				if fr, okR := r.toFloat(); okR {
					if fl <= fr {
						stack[sp-1] = Value{kind: valBool, num: 1}
					} else {
						stack[sp-1] = Value{kind: valBool, num: 0}
					}
				} else {
					res, err := evalComparison("<=", l.ToAny(), r.ToAny())
					if err != nil { return nil, err }
					stack[sp-1] = FromAny(res)
				}
			} else {
				res, err := evalComparison("<=", l.ToAny(), r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpMinus:
			r := stack[sp-1]
			if r.kind == valInt {
				stack[sp-1] = Value{kind: valInt, num: uint64(-int64(r.num))}
			} else if r.kind == valFloat {
				stack[sp-1] = Value{kind: valFloat, num: math.Float64bits(-math.Float64frombits(r.num))}
			} else {
				res, err := evalPrefixExpression("-", r.ToAny())
				if err != nil { return nil, err }
				stack[sp-1] = FromAny(res)
			}

		case OpGetGlobal:
			if sp >= StackSize { return nil, fmt.Errorf("stack overflow") }
			name := consts[inst.arg].ptr.(string)
			val, _ := ctx.Get(name)
			stack[sp] = FromAny(val)
			sp++

		case OpSetGlobal:
			name := consts[inst.arg].ptr.(string)
			val := stack[sp-1]
			err := ctx.Set(name, val.ToAny())
			if err != nil { return nil, err }

		case OpJump:
			ip = int(inst.arg) - 1

		case OpJumpIfFalse:
			cond := stack[sp-1]
			if !isTruthyValue(cond) {
				ip = int(inst.arg) - 1
			}

		case OpJumpIfTrue:
			cond := stack[sp-1]
			if isTruthyValue(cond) {
				ip = int(inst.arg) - 1
			}

		case OpToBool:
			val := stack[sp-1]
			if isTruthyValue(val) {
				stack[sp-1] = Value{kind: valBool, num: 1}
			} else {
				stack[sp-1] = Value{kind: valBool, num: 0}
			}

		case OpCall:
			numArgs := int(inst.arg)
			funcName := stack[sp-1].ptr.(string)
			sp--
			args := make([]any, numArgs)
			for i := numArgs - 1; i >= 0; i-- {
				args[i] = stack[sp-1].ToAny()
				sp--
			}
			builtin, ok := builtins[funcName]
			if !ok {
				return nil, fmt.Errorf("builtin function not found: %s", funcName)
			}
			res, err := builtin(args...)
			if err != nil {
				return nil, err
			}
			if sp >= StackSize { return nil, fmt.Errorf("stack overflow") }
			stack[sp] = FromAny(res)
			sp++

		default:
			return nil, fmt.Errorf("unsupported opcode: %d", op)
		}

		ip++
	}

	if sp == 0 {
		return nil, nil
	}
	return stack[sp-1].ToAny(), nil
}

func isTruthyValue(v Value) bool {
	switch v.kind {
	case valBool:
		return v.num != 0
	case valNil:
		return false
	default:
		return true
	}
}
