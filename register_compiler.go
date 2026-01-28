// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"math"
)

type RegisterCompiler struct {
	instructions []regInstruction
	constants    []Value
	constMap     map[any]int32
	maxReg       uint8
	errors       []string
}

func NewRegisterCompiler() *RegisterCompiler {
	return &RegisterCompiler{
		constMap: make(map[any]int32),
	}
}

func (c *RegisterCompiler) Compile(node Node) (*RegisterBytecode, error) {
	finalReg, err := c.walk(node, 0)
	if err != nil {
		return nil, err
	}
	c.emit(ROpReturn, 0, finalReg, 0, 0)
	return &RegisterBytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
		MaxRegisters: c.maxReg + 1,
	}, nil
}

func (c *RegisterCompiler) walk(node Node, reg uint8) (uint8, error) {
	if reg > 250 {
		return 0, fmt.Errorf("register limit exceeded")
	}
	if reg > c.maxReg {
		c.maxReg = reg
	}

	switch n := node.(type) {
	case *Identifier:
		c.emit(ROpGetGlobal, reg, 0, 0, c.addConstant(Value{Type: ValString, Str: n.Value}))
		return reg, nil

	case *NumberLiteral:
		if n.IsInt {
			c.emit(ROpLoadConst, reg, 0, 0, c.addConstant(Value{Type: ValInt, Num: uint64(n.Int64Value)}))
		} else {
			c.emit(ROpLoadConst, reg, 0, 0, c.addConstant(Value{Type: ValFloat, Num: math.Float64bits(n.Float64Value)}))
		}
		return reg, nil

	case *StringLiteral:
		c.emit(ROpLoadConst, reg, 0, 0, c.addConstant(Value{Type: ValString, Str: n.Value}))
		return reg, nil

	case *BooleanLiteral:
		val := uint64(0)
		if n.Value {
			val = 1
		}
		c.emit(ROpLoadConst, reg, 0, 0, c.addConstant(Value{Type: ValBool, Num: val}))
		return reg, nil

	case *PrefixExpression:
		if n.Operator == "-" {
			c.emit(ROpLoadConst, reg, 0, 0, c.addConstant(Value{Type: ValInt, Num: 0}))
			_, err := c.walk(n.Right, reg+1)
			if err != nil {
				return 0, err
			}
			c.emit(ROpSub, reg, reg, reg+1, 0)
			return reg, nil
		} else if n.Operator == "!" {
			_, err := c.walk(n.Right, reg)
			if err != nil {
				return 0, err
			}
			c.emit(ROpNot, reg, reg, 0, 0)
			return reg, nil
		}

	case *InfixExpression:
		if n.Operator == "&&" {
			_, err := c.walk(n.Left, reg)
			if err != nil {
				return 0, err
			}
			jumpFalse := c.emit(ROpJumpIfFalse, 0, reg, 0, 0)
			_, err = c.walk(n.Right, reg)
			if err != nil {
				return 0, err
			}
			// ensure result is boolean 0/1
			c.emit(ROpNot, reg, reg, 0, 0)
			c.emit(ROpNot, reg, reg, 0, 0)
			jumpEnd := c.emit(ROpJump, 0, 0, 0, 0)
			c.patch(jumpFalse, int32(len(c.instructions)))
			c.emit(ROpLoadConst, reg, 0, 0, c.addConstant(Value{Type: ValBool, Num: 0}))
			c.patch(jumpEnd, int32(len(c.instructions)))
			return reg, nil
		}
		if n.Operator == "||" {
			_, err := c.walk(n.Left, reg)
			if err != nil {
				return 0, err
			}
			jumpTrue := c.emit(ROpJumpIfTrue, 0, reg, 0, 0)
			_, err = c.walk(n.Right, reg)
			if err != nil {
				return 0, err
			}
			// ensure result is boolean 0/1
			c.emit(ROpNot, reg, reg, 0, 0)
			c.emit(ROpNot, reg, reg, 0, 0)
			jumpEnd := c.emit(ROpJump, 0, 0, 0, 0)
			c.patch(jumpTrue, int32(len(c.instructions)))
			c.emit(ROpLoadConst, reg, 0, 0, c.addConstant(Value{Type: ValBool, Num: 1}))
			c.patch(jumpEnd, int32(len(c.instructions)))
			return reg, nil
		}

		leftReg, err := c.walk(n.Left, reg)
		if err != nil {
			return 0, err
		}
		rightReg, err := c.walk(n.Right, reg+1)
		if err != nil {
			return 0, err
		}

		var op ROpCode
		switch n.Operator {
		case "+":
			op = ROpAdd
		case "-":
			op = ROpSub
		case "*":
			op = ROpMul
		case "/":
			op = ROpDiv
		case "%":
			op = ROpMod
		case "==":
			op = ROpEqual
		case ">":
			op = ROpGreater
		case "<":
			op = ROpLess
		case ">=":
			op = ROpGreaterEqual
		case "<=":
			op = ROpLessEqual
		default:
			return 0, fmt.Errorf("unknown operator: %s", n.Operator)
		}
		c.emit(op, reg, leftReg, rightReg, 0)
		return reg, nil

	case *IfExpression:
		condReg, err := c.walk(n.Condition, reg)
		if err != nil {
			return 0, err
		}

		if n.IsSimple {
			return condReg, nil
		}

		jumpFalse := c.emit(ROpJumpIfFalse, 0, condReg, 0, 0)
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
			c.emit(ROpLoadConst, reg, 0, 0, c.addConstant(Value{Type: ValNil}))
		}
		c.patch(jumpEnd, int32(len(c.instructions)))
		return reg, nil

	case *AssignExpression:
		valReg, err := c.walk(n.Value, reg)
		if err != nil {
			return 0, err
		}
		c.emit(ROpSetGlobal, 0, valReg, 0, c.addConstant(Value{Type: ValString, Str: n.Name.Value}))
		return valReg, nil

	case *CallExpression:
		if ident, ok := n.Function.(*Identifier); ok && ident.Value == "concat" {
			for i, arg := range n.Arguments {
				_, err := c.walk(arg, reg+uint8(i))
				if err != nil {
					return 0, err
				}
			}
			c.emit(ROpConcat, reg, reg, uint8(len(n.Arguments)), 0)
			return reg, nil
		}

		for i, arg := range n.Arguments {
			_, err := c.walk(arg, reg+uint8(i+1))
			if err != nil {
				return 0, err
			}
		}
		if ident, ok := n.Function.(*Identifier); ok {
			c.emit(ROpCall, reg, reg+1, uint8(len(n.Arguments)), c.addConstant(Value{Type: ValString, Str: ident.Value}))
		} else {
			return 0, fmt.Errorf("calling non-identifier functions not supported in Register VM yet")
		}
		return reg, nil
	}
	return reg, nil
}

func (c *RegisterCompiler) addConstant(v Value) int32 {
	var key any
	switch v.Type {
	case ValInt:
		key = int64(v.Num)
	case ValFloat:
		key = math.Float64frombits(v.Num)
	case ValBool:
		key = v.Num != 0
	case ValString:
		key = v.Str
	case ValNil:
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
