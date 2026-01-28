// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"encoding/binary"
	"fmt"
	"sync"
)

const StackSize = 2048

type VM struct {
	constants    []any
	instructions Instructions
	stack        []any
	sp           int // stack pointer
}

var vmPool = sync.Pool{
	New: func() any {
		return &VM{
			stack: make([]any, StackSize),
		}
	},
}

func NewVM(bytecode *Bytecode) *VM {
	vm := vmPool.Get().(*VM)
	vm.constants = bytecode.Constants
	vm.instructions = bytecode.Instructions
	vm.sp = 0
	return vm
}

func (vm *VM) Free() {
	// Clear stack to help GC
	for i := 0; i < vm.sp; i++ {
		vm.stack[i] = nil
	}
	vm.constants = nil
	vm.instructions = nil
	vmPool.Put(vm)
}

func (vm *VM) Run(ctx Context) (any, error) {
	ip := 0
	for ip < len(vm.instructions) {
		op := OpCode(vm.instructions[ip])

		switch op {
		case OpConstant:
			constIndex := binary.BigEndian.Uint16(vm.instructions[ip+1:])
			ip += 2
			vm.push(vm.constants[constIndex])

		case OpPop:
			vm.pop()

		case OpAdd, OpSub, OpMul, OpDiv, OpMod:
			right := vm.pop()
			left := vm.pop()
			res, err := evalArithmetic(opToOperator(op), left, right)
			if err != nil {
				return nil, err
			}
			vm.push(res)

		case OpEqual, OpGreater, OpLess, OpGreaterEqual, OpLessEqual:
			right := vm.pop()
			left := vm.pop()
			res, err := evalComparison(opToOperator(op), left, right)
			if err != nil {
				return nil, err
			}
			vm.push(res)

		case OpMinus:
			right := vm.pop()
			res, err := evalPrefixExpression("-", right)
			if err != nil {
				return nil, err
			}
			vm.push(res)

		case OpGetGlobal:
			nameIndex := binary.BigEndian.Uint16(vm.instructions[ip+1:])
			ip += 2
			name := vm.constants[nameIndex].(string)
			val, _ := ctx.Get(name)
			vm.push(val)

		case OpSetGlobal:
			nameIndex := binary.BigEndian.Uint16(vm.instructions[ip+1:])
			ip += 2
			name := vm.constants[nameIndex].(string)
			val := vm.peek()
			err := ctx.Set(name, val)
			if err != nil {
				return nil, err
			}

		case OpJump:
			pos := int(binary.BigEndian.Uint16(vm.instructions[ip+1:]))
			ip = pos - 1

		case OpJumpIfFalse:
			pos := int(binary.BigEndian.Uint16(vm.instructions[ip+1:]))
			condition := vm.peek()
			if !isTruthy(condition) {
				ip = pos - 1
			} else {
				ip += 2
			}

		case OpJumpIfTrue:
			pos := int(binary.BigEndian.Uint16(vm.instructions[ip+1:]))
			condition := vm.peek()
			if isTruthy(condition) {
				ip = pos - 1
			} else {
				ip += 2
			}

		case OpToBool:
			val := vm.pop()
			vm.push(boolToAny(isTruthy(val)))

		case OpCall:
			numArgs := int(vm.instructions[ip+1])
			ip += 1
			funcName := vm.pop().(string)
			args := make([]any, numArgs)
			for i := numArgs - 1; i >= 0; i-- {
				args[i] = vm.pop()
			}
			builtin, ok := builtins[funcName]
			if !ok {
				return nil, fmt.Errorf("builtin function not found: %s", funcName)
			}
			res, err := builtin(args...)
			if err != nil {
				return nil, err
			}
			vm.push(res)

		default:
			return nil, fmt.Errorf("unsupported opcode: %d", op)
		}

		ip++
	}

	if vm.sp == 0 {
		return nil, nil
	}
	return vm.stack[vm.sp-1], nil
}

func (vm *VM) push(obj any) {
	if vm.sp >= StackSize {
		panic("stack overflow")
	}
	vm.stack[vm.sp] = obj
	vm.sp++
}

func (vm *VM) pop() any {
	if vm.sp == 0 {
		panic("stack underflow")
	}
	vm.sp--
	return vm.stack[vm.sp]
}

func (vm *VM) peek() any {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

func opToOperator(op OpCode) string {
	switch op {
	case OpAdd: return "+"
	case OpSub: return "-"
	case OpMul: return "*"
	case OpDiv: return "/"
	case OpMod: return "%"
	case OpEqual: return "=="
	case OpGreater: return ">"
	case OpLess: return "<"
	case OpGreaterEqual: return ">="
	case OpLessEqual: return "<="
	}
	return ""
}
