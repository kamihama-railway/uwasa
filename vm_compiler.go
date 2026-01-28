// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
)

type Bytecode struct {
	Instructions Instructions
	Constants    []any
}

type Compiler struct {
	instructions Instructions
	constants    []any
}

func NewCompiler() *Compiler {
	return &Compiler{
		instructions: Instructions{},
		constants:    []any{},
	}
}

func (c *Compiler) Compile(node Node) error {
	switch n := node.(type) {
	case *Identifier:
		c.emit(OpGetGlobal, c.addConstant(n.Value))

	case *NumberLiteral:
		if n.IsInt {
			c.emit(OpConstant, c.addConstant(n.Int64Value))
		} else {
			c.emit(OpConstant, c.addConstant(n.Float64Value))
		}

	case *StringLiteral:
		c.emit(OpConstant, c.addConstant(n.Value))

	case *BooleanLiteral:
		c.emit(OpConstant, c.addConstant(n.Value))

	case *PrefixExpression:
		err := c.Compile(n.Right)
		if err != nil {
			return err
		}
		switch n.Operator {
		case "-":
			c.emit(OpMinus)
		default:
			return fmt.Errorf("unknown operator %s", n.Operator)
		}

	case *InfixExpression:
		if n.Operator == "&&" {
			err := c.Compile(n.Left)
			if err != nil {
				return err
			}
			jumpIfFalsePos := c.emit(OpJumpIfFalse, 9999)
			c.emit(OpPop)
			err = c.Compile(n.Right)
			if err != nil {
				return err
			}
			c.emit(OpToBool)
			jumpEndPos := c.emit(OpJump, 9999)

			c.changeOperand(jumpIfFalsePos, len(c.instructions))
			c.emit(OpPop)
			c.emit(OpConstant, c.addConstant(false))

			c.changeOperand(jumpEndPos, len(c.instructions))
		} else if n.Operator == "||" {
			err := c.Compile(n.Left)
			if err != nil {
				return err
			}
			jumpIfTruePos := c.emit(OpJumpIfTrue, 9999)
			c.emit(OpPop)
			err = c.Compile(n.Right)
			if err != nil {
				return err
			}
			c.emit(OpToBool)
			jumpEndPos := c.emit(OpJump, 9999)

			c.changeOperand(jumpIfTruePos, len(c.instructions))
			c.emit(OpPop)
			c.emit(OpConstant, c.addConstant(true))

			c.changeOperand(jumpEndPos, len(c.instructions))
		} else {
			err := c.Compile(n.Left)
			if err != nil {
				return err
			}

			// Optimization: fused compare with constant
			if lit, ok := isLiteral(n.Right); ok {
				constIdx := c.addConstant(lit)
				switch n.Operator {
				case "==": c.emit(OpEqualConst, constIdx)
				case ">":  c.emit(OpGreaterConst, constIdx)
				case "<":  c.emit(OpLessConst, constIdx)
				case ">=": c.emit(OpGreaterEqualConst, constIdx)
				case "<=": c.emit(OpLessEqualConst, constIdx)
				default:
					goto normalInfix
				}
				return nil
			}

		normalInfix:
			err = c.Compile(n.Right)
			if err != nil {
				return err
			}

			switch n.Operator {
			case "+": c.emit(OpAdd)
			case "-": c.emit(OpSub)
			case "*": c.emit(OpMul)
			case "/": c.emit(OpDiv)
			case "%": c.emit(OpMod)
			case "==": c.emit(OpEqual)
			case ">": c.emit(OpGreater)
			case "<": c.emit(OpLess)
			case ">=": c.emit(OpGreaterEqual)
			case "<=": c.emit(OpLessEqual)
			default:
				return fmt.Errorf("unknown operator %s", n.Operator)
			}
		}

	case *IfExpression:
		err := c.Compile(n.Condition)
		if err != nil {
			return err
		}

		if n.IsSimple {
			c.emit(OpToBool)
			return nil
		}

		jumpIfFalsePos := c.emit(OpJumpIfFalse, 9999)
		c.emit(OpPop) // pop condition (true)

		err = c.Compile(n.Consequence)
		if err != nil {
			return err
		}

		jumpEndPos := c.emit(OpJump, 9999)

		c.changeOperand(jumpIfFalsePos, len(c.instructions))
		c.emit(OpPop) // pop condition (false)

		if n.Alternative != nil {
			err = c.Compile(n.Alternative)
			if err != nil {
				return err
			}
		} else {
			c.emit(OpConstant, c.addConstant(nil))
		}

		c.changeOperand(jumpEndPos, len(c.instructions))

	case *AssignExpression:
		err := c.Compile(n.Value)
		if err != nil {
			return err
		}
		c.emit(OpSetGlobal, c.addConstant(n.Name.Value))

	case *CallExpression:
		for _, arg := range n.Arguments {
			err := c.Compile(arg)
			if err != nil {
				return err
			}
		}
		ident, ok := n.Function.(*Identifier)
		if !ok {
			return fmt.Errorf("only builtin function calls are supported")
		}
		c.emit(OpConstant, c.addConstant(ident.Value))
		c.emit(OpCall, len(n.Arguments))

	}
	return nil
}

func (c *Compiler) addConstant(obj any) int {
	for i, constant := range c.constants {
		if constant == obj {
			return i
		}
	}
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func isLiteral(node Node) (any, bool) {
	switch n := node.(type) {
	case *NumberLiteral:
		if n.IsInt { return n.Int64Value, true }
		return n.Float64Value, true
	case *StringLiteral:
		return n.Value, true
	case *BooleanLiteral:
		return n.Value, true
	}
	return nil, false
}

func (c *Compiler) emit(op OpCode, operands ...int) int {
	ins := Make(op, operands...)
	pos := len(c.instructions)
	c.instructions = append(c.instructions, ins...)
	return pos
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := OpCode(c.instructions[opPos])
	newInstruction := Make(op, operand)

	for i := 0; i < len(newInstruction); i++ {
		c.instructions[opPos+i] = newInstruction[i]
	}
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
	}
}

func (b *Bytecode) Render() *RenderedBytecode {
	offsetToIdx := make(map[int]int)
	type tempIns struct {
		op   OpCode
		args []int
	}
	var rawIns []tempIns

	i := 0
	for i < len(b.Instructions) {
		offsetToIdx[i] = len(rawIns)
		op := b.Instructions[i]
		def, _ := Lookup(op)
		operands, read := ReadOperands(def, b.Instructions[i+1:])
		rawIns = append(rawIns, tempIns{op: OpCode(op), args: operands})
		i += 1 + read
	}
	offsetToIdx[i] = len(rawIns)

	for idx, inst := range rawIns {
		if inst.op == OpJump || inst.op == OpJumpIfFalse || inst.op == OpJumpIfTrue {
			rawIns[idx].args[0] = offsetToIdx[inst.args[0]]
		}
	}

	var renderedIns []vmInstruction
	rawToRendered := make(map[int]int)

	for idx := 0; idx < len(rawIns); idx++ {
		rawToRendered[idx] = len(renderedIns)
		inst := rawIns[idx]

		if inst.op == OpGetGlobal && idx+3 < len(rawIns) {
			next := rawIns[idx+1]
			nextNext := rawIns[idx+2]
			next3 := rawIns[idx+3]
			if next.op == OpEqualConst && nextNext.op == OpJumpIfFalse && next3.op == OpPop {
				renderedIns = append(renderedIns, vmInstruction{
					op:   OpFusedCompareGlobalConstJumpIfFalse,
					arg1: uint16(inst.args[0]),
					arg2: uint16(next.args[0]),
					arg3: uint16(nextNext.args[0]),
				})
				rawToRendered[idx+1] = len(renderedIns) - 1
				rawToRendered[idx+2] = len(renderedIns) - 1
				rawToRendered[idx+3] = len(renderedIns) - 1
				idx += 3
				continue
			}
		}

		if inst.op == OpGetGlobal && idx+1 < len(rawIns) {
			next := rawIns[idx+1]
			var fused OpCode
			switch next.op {
			case OpAdd: fused = OpAddGlobal
			}
			if fused != 0 {
				renderedIns = append(renderedIns, vmInstruction{
					op:   fused,
					arg1: uint16(inst.args[0]),
				})
				rawToRendered[idx+1] = len(renderedIns) - 1
				idx += 1
				continue
			}
		}

		renderedIns = append(renderedIns, vmInstruction{
			op: inst.op,
		})
		if len(inst.args) > 0 {
			renderedIns[len(renderedIns)-1].arg1 = uint16(inst.args[0])
		}
		if len(inst.args) > 1 {
			renderedIns[len(renderedIns)-1].arg2 = uint16(inst.args[1])
		}
		if len(inst.args) > 2 {
			renderedIns[len(renderedIns)-1].arg3 = uint16(inst.args[2])
		}
	}
	rawToRendered[len(rawIns)] = len(renderedIns)

	for idx := range renderedIns {
		inst := renderedIns[idx]
		if inst.op == OpJump || inst.op == OpJumpIfFalse || inst.op == OpJumpIfTrue || inst.op == OpFusedCompareGlobalConstJumpIfFalse {
			target := 0
			if inst.op == OpFusedCompareGlobalConstJumpIfFalse {
				target = int(inst.arg3)
			} else {
				target = int(inst.arg1)
			}
			newTarget := rawToRendered[target]
			if inst.op == OpFusedCompareGlobalConstJumpIfFalse {
				renderedIns[idx].arg3 = uint16(newTarget)
			} else {
				renderedIns[idx].arg1 = uint16(newTarget)
			}
		}
	}

	renderedConstants := make([]Value, len(b.Constants))
	for i, c := range b.Constants {
		renderedConstants[i] = FromAny(c)
	}

	return &RenderedBytecode{
		Instructions: renderedIns,
		Constants:    renderedConstants,
	}
}
