// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
)

type Bytecode struct {
	Instructions  Instructions
	Constants     []any
	VariableNames []string
}

type Compiler struct {
	instructions  Instructions
	constants     []any
	variableNames []string
	constMap      map[any]int
}

func NewCompiler() *Compiler {
	return &Compiler{
		instructions: Instructions{},
		constants:    []any{},
		constMap:     make(map[any]int),
	}
}

func (c *Compiler) addVariable(name string) int {
	for i, v := range c.variableNames {
		if v == name {
			return i
		}
	}
	c.variableNames = append(c.variableNames, name)
	return len(c.variableNames) - 1
}

func (c *Compiler) Compile(node Node) error {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *Identifier:
		c.emit(OpGetGlobal, c.addVariable(n.Value))

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
		// Algebraic simplifications (inspired by Recompiler)
		switch n.Operator {
		case "+":
			if isZero(n.Left) { return c.Compile(n.Right) }
			if isZero(n.Right) { return c.Compile(n.Left) }
			// Optimization: Detect string concatenation chain
			if isStringChain(n) {
				args := collectConcatArgs(n)
				for _, arg := range args {
					if err := c.Compile(arg); err != nil { return err }
				}
				c.emit(OpConcat, len(args))
				return nil
			}
		case "-":
			if isZero(n.Right) { return c.Compile(n.Left) }
			if isSameIdentifier(n.Left, n.Right) {
				c.emit(OpConstant, c.addConstant(int64(0)))
				return nil
			}
		case "*":
			if isZero(n.Left) || isZero(n.Right) {
				c.emit(OpConstant, c.addConstant(int64(0)))
				return nil
			}
			if isOne(n.Left) { return c.Compile(n.Right) }
			if isOne(n.Right) { return c.Compile(n.Left) }
		case "==":
			if isSameIdentifier(n.Left, n.Right) {
				c.emit(OpConstant, c.addConstant(true))
				return nil
			}
		case "/":
			if isOne(n.Right) { return c.Compile(n.Left) }
			if isSameIdentifier(n.Left, n.Right) && !hasSideEffects(n.Left) {
				c.emit(OpConstant, c.addConstant(int64(1)))
				return nil
			}
		}

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
		// Static condition evaluation
		if lit, ok := n.Condition.(*BooleanLiteral); ok {
			if lit.Value {
				return c.Compile(n.Consequence)
			} else if n.Alternative != nil {
				return c.Compile(n.Alternative)
			} else {
				c.emit(OpConstant, c.addConstant(nil))
				return nil
			}
		}

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
		if isSameIdentifier(n.Name, n.Value) {
			return c.Compile(n.Name)
		}
		err := c.Compile(n.Value)
		if err != nil {
			return err
		}
		c.emit(OpSetGlobal, c.addVariable(n.Name.Value))

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

func isStringChain(ie *InfixExpression) bool {
	if ie.Operator != "+" { return false }
	// If any side is a string literal, we consider it a string chain.
	if _, ok := ie.Left.(*StringLiteral); ok { return true }
	if _, ok := ie.Right.(*StringLiteral); ok { return true }
	// If any side is another string chain, it's also a string chain.
	if leftIE, ok := ie.Left.(*InfixExpression); ok && isStringChain(leftIE) { return true }
	if rightIE, ok := ie.Right.(*InfixExpression); ok && isStringChain(rightIE) { return true }
	return false
}

func collectConcatArgs(node Node) []Node {
	ie, ok := node.(*InfixExpression)
	if !ok || ie.Operator != "+" {
		return []Node{node}
	}
	var args []Node
	args = append(args, collectConcatArgs(ie.Left)...)
	args = append(args, collectConcatArgs(ie.Right)...)
	return args
}

func (c *Compiler) addConstant(obj any) int {
	if idx, ok := c.constMap[obj]; ok {
		return idx
	}
	c.constants = append(c.constants, obj)
	idx := len(c.constants) - 1
	c.constMap[obj] = idx
	return idx
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

func (c *Bytecode) Render() *RenderedBytecode {
	offsetToIdx := make(map[int]int)
	type tempIns struct {
		op   OpCode
		args []int
	}
	var rawIns []tempIns

	i := 0
	for i < len(c.Instructions) {
		offsetToIdx[i] = len(rawIns)
		op := c.Instructions[i]
		def, _ := Lookup(op)
		operands, read := ReadOperands(def, c.Instructions[i+1:])
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
	var resolvedBuiltins []BuiltinFunc

	for idx := 0; idx < len(rawIns); idx++ {
		rawToRendered[idx] = len(renderedIns)
		inst := rawIns[idx]

		// Fusion: Constant(name) + OpCall -> OpCallResolved or OpConcat
		if inst.op == OpConstant && idx+1 < len(rawIns) && rawIns[idx+1].op == OpCall {
			constVal := c.Constants[inst.args[0]]
			if name, ok := constVal.(string); ok {
				numArgs := rawIns[idx+1].args[0]
				if name == "concat" {
					renderedIns = append(renderedIns, vmInstruction{
						op:   OpConcat,
						arg1: uint16(numArgs),
					})
					rawToRendered[idx+1] = len(renderedIns) - 1
					idx += 1
					continue
				}
				if builtin, ok := builtins[name]; ok {
					resolvedBuiltins = append(resolvedBuiltins, builtin)
					renderedIns = append(renderedIns, vmInstruction{
						op:   OpCallResolved,
						arg1: uint16(numArgs),
						arg2: uint16(len(resolvedBuiltins) - 1),
					})
					rawToRendered[idx+1] = len(renderedIns) - 1
					idx += 1
					continue
				}
			}
		}

		// Fusion: GetGlobal + EqualConst + JumpIfFalse + Pop
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

		// Fusion: GetGlobal + BinaryOp
		if inst.op == OpGetGlobal && idx+1 < len(rawIns) {
			next := rawIns[idx+1]
			var fused OpCode
			switch next.op {
			case OpAdd: fused = OpAddGlobal
			case OpSub: fused = OpSubGlobal
			case OpMul: fused = OpMulGlobal
			case OpDiv: fused = OpDivGlobal
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

		// Fusion: Constant + BinaryOp
		if inst.op == OpConstant && idx+1 < len(rawIns) {
			next := rawIns[idx+1]
			var fused OpCode
			switch next.op {
			case OpAdd: fused = OpAddConst
			case OpSub: fused = OpSubConst
			case OpMul: fused = OpMulConst
			case OpDiv: fused = OpDivConst
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

		// Fusion: JumpIfFalse + Pop -> JumpIfFalsePop
		if inst.op == OpJumpIfFalse && idx+1 < len(rawIns) && rawIns[idx+1].op == OpPop {
			renderedIns = append(renderedIns, vmInstruction{
				op:   OpJumpIfFalsePop,
				arg1: uint16(inst.args[0]),
			})
			rawToRendered[idx+1] = len(renderedIns) - 1
			idx += 1
			continue
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
		if inst.op == OpJump || inst.op == OpJumpIfFalse || inst.op == OpJumpIfTrue || inst.op == OpJumpIfFalsePop || inst.op == OpFusedCompareGlobalConstJumpIfFalse {
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

	renderedConstants := make([]Value, len(c.Constants))
	for i, constant := range c.Constants {
		renderedConstants[i] = FromAny(constant)
	}

	return &RenderedBytecode{
		Instructions:  renderedIns,
		Constants:     renderedConstants,
		VariableNames: c.VariableNames,
		Builtins:      resolvedBuiltins,
	}
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions:  c.instructions,
		Constants:     c.constants,
		VariableNames: c.variableNames,
	}
}
