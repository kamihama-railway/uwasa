// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package rvm

import (
	"fmt"
	"math"
	"github.com/kamihama-railway/uwasa/types"
	"github.com/kamihama-railway/uwasa/ast"
)

type RegisterCompiler struct {
	instructions []regInstruction
	constants    []types.Value
	constMap     map[any]int32
	maxReg       uint8
	errors       []string
}

func NewRegisterCompiler() *RegisterCompiler {
	return &RegisterCompiler{
		constMap: make(map[any]int32),
	}
}

func (c *RegisterCompiler) Compile(node ast.Node) (*RegisterBytecode, error) {
	finalReg, err := c.walk(node, 0)
	if err != nil {
		return nil, err
	}
	c.emit(ROpReturn, 0, uint8(finalReg), 0, 0)

	bc := &RegisterBytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
		MaxRegisters: c.maxReg + 1,
	}

	// Safety check: ensure all instructions are within register bounds
	for _, inst := range bc.Instructions {
		switch inst.Op {
		case ROpCall, ROpConcat:
			if int(inst.Src1)+int(inst.Src2) > int(bc.MaxRegisters) {
				return nil, fmt.Errorf("register range out of bounds")
			}
			if inst.Dest >= bc.MaxRegisters || inst.Src1 >= bc.MaxRegisters {
				return nil, fmt.Errorf("register index out of bounds")
			}
		case ROpReturn, ROpNot, ROpMove, ROpJumpIfFalse, ROpJumpIfTrue:
			if inst.Dest >= bc.MaxRegisters || inst.Src1 >= bc.MaxRegisters {
				return nil, fmt.Errorf("register index out of bounds")
			}
		case ROpLoadConst, ROpGetGlobal:
			if inst.Dest >= bc.MaxRegisters {
				return nil, fmt.Errorf("register index out of bounds")
			}
		case ROpSetGlobal:
			if inst.Src1 >= bc.MaxRegisters {
				return nil, fmt.Errorf("register index out of bounds")
			}
		case ROpJump:
			// No registers to check
		default:
			if inst.Dest >= bc.MaxRegisters || inst.Src1 >= bc.MaxRegisters || inst.Src2 >= bc.MaxRegisters {
				return nil, fmt.Errorf("register index out of bounds")
			}
		}
	}

	return bc, nil
}

func (c *RegisterCompiler) walk(node ast.Node, reg int) (int, error) {
	if reg > 250 {
		return 0, fmt.Errorf("register limit exceeded")
	}
	if uint8(reg) > c.maxReg {
		c.maxReg = uint8(reg)
	}

	uReg := uint8(reg)

	switch n := node.(type) {
	case *ast.Identifier:
		c.emit(ROpGetGlobal, uReg, 0, 0, c.addConstant(types.Value{Type: types.ValString, Str: n.Value}))
		return reg, nil

	case *ast.NumberLiteral:
		if n.IsInt {
			c.emit(ROpLoadConst, uReg, 0, 0, c.addConstant(types.Value{Type: types.ValInt, Num: uint64(n.Int64Value)}))
		} else {
			c.emit(ROpLoadConst, uReg, 0, 0, c.addConstant(types.Value{Type: types.ValFloat, Num: math.Float64bits(n.Float64Value)}))
		}
		return reg, nil

	case *ast.StringLiteral:
		c.emit(ROpLoadConst, uReg, 0, 0, c.addConstant(types.Value{Type: types.ValString, Str: n.Value}))
		return reg, nil

	case *ast.BooleanLiteral:
		val := uint64(0)
		if n.Value {
			val = 1
		}
		c.emit(ROpLoadConst, uReg, 0, 0, c.addConstant(types.Value{Type: types.ValBool, Num: val}))
		return reg, nil

	case *ast.PrefixExpression:
		if n.Operator == "-" {
			c.emit(ROpLoadConst, uReg, 0, 0, c.addConstant(types.Value{Type: types.ValInt, Num: 0}))
			_, err := c.walk(n.Right, reg+1)
			if err != nil {
				return 0, err
			}
			c.emit(ROpSub, uReg, uReg, uint8(reg+1), 0)
			return reg, nil
		} else if n.Operator == "!" {
			_, err := c.walk(n.Right, reg)
			if err != nil {
				return 0, err
			}
			c.emit(ROpNot, uReg, uReg, 0, 0)
			return reg, nil
		}

	case *ast.InfixExpression:
		if n.Operator == "&&" {
			_, err := c.walk(n.Left, reg)
			if err != nil {
				return 0, err
			}
			jumpFalse := c.emit(ROpJumpIfFalse, 0, uReg, 0, 0)
			_, err = c.walk(n.Right, reg)
			if err != nil {
				return 0, err
			}
			// ensure result is boolean 0/1
			c.emit(ROpNot, uReg, uReg, 0, 0)
			c.emit(ROpNot, uReg, uReg, 0, 0)
			jumpEnd := c.emit(ROpJump, 0, 0, 0, 0)
			c.patch(jumpFalse, int32(len(c.instructions)))
			c.emit(ROpLoadConst, uReg, 0, 0, c.addConstant(types.Value{Type: types.ValBool, Num: 0}))
			c.patch(jumpEnd, int32(len(c.instructions)))
			return reg, nil
		}
		if n.Operator == "||" {
			_, err := c.walk(n.Left, reg)
			if err != nil {
				return 0, err
			}
			jumpTrue := c.emit(ROpJumpIfTrue, 0, uReg, 0, 0)
			_, err = c.walk(n.Right, reg)
			if err != nil {
				return 0, err
			}
			// ensure result is boolean 0/1
			c.emit(ROpNot, uReg, uReg, 0, 0)
			c.emit(ROpNot, uReg, uReg, 0, 0)
			jumpEnd := c.emit(ROpJump, 0, 0, 0, 0)
			c.patch(jumpTrue, int32(len(c.instructions)))
			c.emit(ROpLoadConst, uReg, 0, 0, c.addConstant(types.Value{Type: types.ValBool, Num: 1}))
			c.patch(jumpEnd, int32(len(c.instructions)))
			return reg, nil
		}

		lReg, err := c.walk(n.Left, reg)
		if err != nil {
			return 0, err
		}
		rReg, err := c.walk(n.Right, reg+1)
		if err != nil {
			return 0, err
		}

		var op ROpCode
		switch n.Operator {
		case "+": op = ROpAdd
		case "-": op = ROpSub
		case "*": op = ROpMul
		case "/": op = ROpDiv
		case "%": op = ROpMod
		case "==": op = ROpEqual
		case ">": op = ROpGreater
		case "<": op = ROpLess
		case ">=": op = ROpGreaterEqual
		case "<=": op = ROpLessEqual
		default:
			return 0, fmt.Errorf("unknown operator: %s", n.Operator)
		}
		c.emit(op, uReg, uint8(lReg), uint8(rReg), 0)
		return reg, nil

	case *ast.IfExpression:
		cReg, err := c.walk(n.Condition, reg)
		if err != nil {
			return 0, err
		}

		if n.IsSimple {
			return cReg, nil
		}

		jumpFalse := c.emit(ROpJumpIfFalse, 0, uint8(cReg), 0, 0)
		_, err = c.walk(n.Consequence, reg)
		if err != nil {
			return 0, err
		}

		jumpEnd := c.emit(ROpJump, 0, 0, 0, 0)
		c.patch(jumpFalse, int32(len(c.instructions)))

		if n.Alternative != nil {
			_, err = c.walk(n.Alternative, reg)
			if err != nil {
				return 0, err
			}
		} else {
			c.emit(ROpLoadConst, uReg, 0, 0, c.addConstant(types.Value{Type: types.ValNil}))
		}
		c.patch(jumpEnd, int32(len(c.instructions)))
		return reg, nil

	case *ast.AssignExpression:
		vReg, err := c.walk(n.Value, reg)
		if err != nil {
			return 0, err
		}
		c.emit(ROpSetGlobal, 0, uint8(vReg), 0, c.addConstant(types.Value{Type: types.ValString, Str: n.Name.Value}))
		return vReg, nil

	case *ast.CallExpression:
		if ident, ok := n.Function.(*ast.Identifier); ok && ident.Value == "concat" {
			for i, arg := range n.Arguments {
				_, err := c.walk(arg, reg+i)
				if err != nil {
					return 0, err
				}
			}
			c.emit(ROpConcat, uReg, uReg, uint8(len(n.Arguments)), 0)
			return reg, nil
		}

		for i, arg := range n.Arguments {
			_, err := c.walk(arg, reg+i+1)
			if err != nil {
				return 0, err
			}
		}
		if ident, ok := n.Function.(*ast.Identifier); ok {
			c.emit(ROpCall, uReg, uint8(reg+1), uint8(len(n.Arguments)), c.addConstant(types.Value{Type: types.ValString, Str: ident.Value}))
		} else {
			return 0, fmt.Errorf("calling non-identifier functions not supported in Register VM yet")
		}
		return reg, nil
	}
	return reg, nil
}

func (c *RegisterCompiler) addConstant(v types.Value) int32 {
	var key any
	switch v.Type {
	case types.ValInt:
		key = int64(v.Num)
	case types.ValFloat:
		key = math.Float64frombits(v.Num)
	case types.ValBool:
		key = v.Num != 0
	case types.ValString:
		key = v.Str
	case types.ValNil:
		key = nil
	}
	if idx, ok := c.constMap[key]; ok {
		return idx
	}
	idx := int32(len(c.constants))
	c.constants = append(c.constants, v)
	c.constMap[key] = idx
	return idx
}

func (c *RegisterCompiler) emit(op ROpCode, dest, src1, src2 uint8, arg int32) int {
	c.instructions = append(c.instructions, regInstruction{Op: op, Dest: dest, Src1: src1, Src2: src2, Arg: arg})
	return len(c.instructions) - 1
}

func (c *RegisterCompiler) patch(pos int, arg int32) {
	c.instructions[pos].Arg = arg
}
