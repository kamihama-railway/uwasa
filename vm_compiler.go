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
	// Simple constant folding for name strings could be done here if needed,
	// but let's just append for now.
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
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
	var renderedIns []vmInstruction

	// First pass: identify instruction boundaries and map offsets to indices
	i := 0
	for i < len(b.Instructions) {
		offsetToIdx[i] = len(renderedIns)
		op := b.Instructions[i]
		def, _ := Lookup(op)
		renderedIns = append(renderedIns, vmInstruction{op: OpCode(op)})
		i += 1
		for _, w := range def.OperandWidths {
			i += w
		}
	}
	offsetToIdx[i] = len(renderedIns) // handle end of stream

	// Second pass: fill in operands and re-map jumps
	i = 0
	idx := 0
	for i < len(b.Instructions) {
		op := b.Instructions[i]
		def, _ := Lookup(op)
		operands, read := ReadOperands(def, b.Instructions[i+1:])

		arg := uint16(0)
		if len(operands) > 0 {
			if op == byte(OpJump) || op == byte(OpJumpIfFalse) || op == byte(OpJumpIfTrue) {
				arg = uint16(offsetToIdx[operands[0]])
			} else {
				arg = uint16(operands[0])
			}
		}
		renderedIns[idx].arg = arg

		i += 1 + read
		idx++
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
