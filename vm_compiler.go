// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"math"
)

type VMCompiler struct {
	instructions []vmInstruction
	constants    []Value
	constMap     map[any]int32
	errors       []string
}

func NewVMCompiler() *VMCompiler {
	return &VMCompiler{
		constMap: make(map[any]int32),
	}
}

func (c *VMCompiler) Compile(node Node) (*RenderedBytecode, error) {
	err := c.walk(node)
	if err != nil {
		return nil, err
	}
	c.peephole()
	return &RenderedBytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
	}, nil
}

func (c *VMCompiler) peephole() {
	if len(c.instructions) < 2 {
		return
	}

	newInsts := make([]vmInstruction, 0, len(c.instructions))
	oldToNew := make([]int, len(c.instructions)+1)

	for i := 0; i < len(c.instructions); i++ {
		oldToNew[i] = len(newInsts)
		inst := c.instructions[i]

		// 3-instruction fusion: GetGlobal + Push + Equal/Greater/Less + JumpIfFalse
		if i+3 < len(c.instructions) &&
			inst.Op == OpGetGlobal &&
			c.instructions[i+1].Op == OpPush &&
			(c.instructions[i+2].Op == OpEqual || c.instructions[i+2].Op == OpGreater || c.instructions[i+2].Op == OpLess) &&
			c.instructions[i+3].Op == OpJumpIfFalse {

			gIdx := inst.Arg
			cIdx := c.instructions[i+1].Arg
			jTarget := c.instructions[i+3].Arg

			if gIdx < 1024 && cIdx < 1024 && jTarget < 4096 && c.instructions[i+2].Op == OpEqual {
				fusedArg := (gIdx << 22) | (cIdx << 12) | jTarget
				newInsts = append(newInsts, vmInstruction{Op: OpFusedCompareGlobalConstJumpIfFalse, Arg: fusedArg})
				oldToNew[i+1] = len(newInsts) - 1
				oldToNew[i+2] = len(newInsts) - 1
				oldToNew[i+3] = len(newInsts) - 1
				i += 3
				continue
			}
		}

		// 2-instruction fusion
		if i+2 < len(c.instructions) &&
			inst.Op == OpGetGlobal &&
			c.instructions[i+1].Op == OpPush {

			gIdx := inst.Arg
			cIdx := c.instructions[i+1].Arg

			if gIdx < 65536 && cIdx < 65536 {
				// GetGlobal + Push + Add -> AddGlobal
				if c.instructions[i+2].Op == OpAdd {
					newInsts = append(newInsts, vmInstruction{Op: OpAddGlobal, Arg: (cIdx << 16) | gIdx})
					oldToNew[i+1] = len(newInsts) - 1
					oldToNew[i+2] = len(newInsts) - 1
					i += 2
					continue
				}

				// GetGlobal + Push + Compare -> CompareGlobalConst
				op := OpCode(0)
				switch c.instructions[i+2].Op {
				case OpEqual: op = OpEqualGlobalConst
				case OpGreater: op = OpGreaterGlobalConst
				case OpLess: op = OpLessGlobalConst
				}

				if op != 0 {
					newInsts = append(newInsts, vmInstruction{Op: op, Arg: (gIdx << 16) | cIdx})
					oldToNew[i+1] = len(newInsts) - 1
					oldToNew[i+2] = len(newInsts) - 1
					i += 2
					continue
				}
			}
		}

		// 3-instruction fusion: GetGlobal + GetGlobal + Add -> AddGlobalGlobal
		if i+2 < len(c.instructions) &&
			inst.Op == OpGetGlobal &&
			c.instructions[i+1].Op == OpGetGlobal &&
			c.instructions[i+2].Op == OpAdd {

			g1Idx := inst.Arg
			g2Idx := c.instructions[i+1].Arg
			if g1Idx < 65536 && g2Idx < 65536 {
				newInsts = append(newInsts, vmInstruction{Op: OpAddGlobalGlobal, Arg: (g1Idx << 16) | g2Idx})
				oldToNew[i+1] = len(newInsts) - 1
				oldToNew[i+2] = len(newInsts) - 1
				i += 2
				continue
			}
		}

		// 2-instruction fusion: GetGlobal + JumpIfFalse/True
		if i+1 < len(c.instructions) &&
			inst.Op == OpGetGlobal {

			gIdx := inst.Arg
			jTarget := c.instructions[i+1].Arg

			if gIdx < 65536 && jTarget < 65536 {
				op := OpCode(0)
				switch c.instructions[i+1].Op {
				case OpJumpIfFalse: op = OpGetGlobalJumpIfFalse
				case OpJumpIfTrue: op = OpGetGlobalJumpIfTrue
				}

				if op != 0 {
					newInsts = append(newInsts, vmInstruction{Op: op, Arg: (gIdx << 16) | jTarget})
					oldToNew[i+1] = len(newInsts) - 1
					i += 1
					continue
				}
			}
		}

		newInsts = append(newInsts, inst)
	}
	oldToNew[len(c.instructions)] = len(newInsts)

	// Fix jump targets
	for i := range newInsts {
		switch newInsts[i].Op {
		case OpJump, OpJumpIfFalse, OpJumpIfTrue:
			newInsts[i].Arg = int32(oldToNew[newInsts[i].Arg])
		case OpFusedCompareGlobalConstJumpIfFalse:
			gIdx := (newInsts[i].Arg >> 22) & 0x3FF
			cIdx := (newInsts[i].Arg >> 12) & 0x3FF
			jTarget := newInsts[i].Arg & 0xFFF
			newInsts[i].Arg = (gIdx << 22) | (cIdx << 12) | int32(oldToNew[jTarget])
		case OpGetGlobalJumpIfFalse, OpGetGlobalJumpIfTrue:
			gIdx := newInsts[i].Arg >> 16
			jTarget := newInsts[i].Arg & 0xFFFF
			newInsts[i].Arg = (gIdx << 16) | int32(oldToNew[jTarget])
		}
	}

	c.instructions = newInsts
}

func (c *VMCompiler) CompileOptimized(node Node, opts EngineOptions) (*RenderedBytecode, error) {
	optimized := node
	if opts.OptimizationLevel >= OptBasic {
		optimized = Fold(optimized)
	}

	if opts.UseRecompiler {
		var err error
		optimized, err = c.optimize(optimized)
		if err != nil {
			return nil, err
		}
	}

	return c.Compile(optimized)
}

func (c *VMCompiler) optimize(node Node) (Node, error) {
	res := c.simplify(node)
	if len(c.errors) > 0 {
		return nil, fmt.Errorf("VM static analysis errors: %v", c.errors)
	}
	return res, nil
}

func (c *VMCompiler) simplify(node Node) Node {
	if node == nil { return nil }
	switch n := node.(type) {
	case *PrefixExpression:
		n.Right = c.simplify(n.Right).(Expression)
		if n.Operator == "-" {
			if _, ok := n.Right.(*StringLiteral); ok {
				c.errors = append(c.errors, "invalid operation: -string")
			}
		}
		return n
	case *InfixExpression:
		n.Left = c.simplify(n.Left).(Expression)
		n.Right = c.simplify(n.Right).(Expression)
		c.checkTypeMismatch(n)

		switch n.Operator {
		case "+":
			if isZero(n.Left) { return n.Right }
			if isZero(n.Right) { return n.Left }
		case "-":
			if isZero(n.Right) { return n.Left }
			if isSameIdentifier(n.Left, n.Right) {
				return &NumberLiteral{Int64Value: 0, IsInt: true}
			}
		case "*":
			if isZero(n.Left) || isZero(n.Right) { return &NumberLiteral{Int64Value: 0, IsInt: true} }
			if isOne(n.Left) { return n.Right }
			if isOne(n.Right) { return n.Left }
		case "/":
			if isZero(n.Right) {
				c.errors = append(c.errors, "division by zero")
				return n
			}
			if isOne(n.Right) { return n.Left }
			if isSameIdentifier(n.Left, n.Right) && !hasSideEffects(n.Left) {
				return &NumberLiteral{Int64Value: 1, IsInt: true}
			}
		case "==":
			if isSameIdentifier(n.Left, n.Right) && !hasSideEffects(n.Left) {
				return &BooleanLiteral{Value: true}
			}
		}
		return n
	case *IfExpression:
		n.Condition = c.simplify(n.Condition).(Expression)
		if n.Consequence != nil { n.Consequence = c.simplify(n.Consequence).(Expression) }
		if n.Alternative != nil { n.Alternative = c.simplify(n.Alternative).(Expression) }
		return n
	case *AssignExpression:
		n.Value = c.simplify(n.Value).(Expression)
		return n
	default:
		return n
	}
}

func (c *VMCompiler) checkTypeMismatch(ie *InfixExpression) {
	left := ie.Left
	right := ie.Right
	_, okLS := left.(*StringLiteral)
	_, okRS := right.(*StringLiteral)
	_, okLN := left.(*NumberLiteral)
	_, okRN := right.(*NumberLiteral)

	switch ie.Operator {
	case "-", "*", "/", "%", ">", "<", ">=", "<=":
		if okLS || okRS {
			c.errors = append(c.errors, fmt.Sprintf("invalid operation: string %s string/number", ie.Operator))
		}
	case "+":
		if (okLS && okRN) || (okLN && okRS) {
			c.errors = append(c.errors, "invalid operation: string + number mismatch")
		}
	}
}

func (c *VMCompiler) walk(node Node) error {
	switch n := node.(type) {
	case *Identifier:
		c.emit(OpGetGlobal, c.addConstant(Value{Type: ValString, Str: n.Value}))
	case *NumberLiteral:
		if n.IsInt {
			c.emit(OpPush, c.addConstant(Value{Type: ValInt, Num: uint64(n.Int64Value)}))
		} else {
			c.emit(OpPush, c.addConstant(Value{Type: ValFloat, Num: math.Float64bits(n.Float64Value)}))
		}
	case *StringLiteral:
		c.emit(OpPush, c.addConstant(Value{Type: ValString, Str: n.Value}))
	case *BooleanLiteral:
		val := uint64(0)
		if n.Value { val = 1 }
		c.emit(OpPush, c.addConstant(Value{Type: ValBool, Num: val}))
	case *PrefixExpression:
		if n.Operator == "-" {
			c.emit(OpPush, c.addConstant(Value{Type: ValInt, Num: 0}))
			err := c.walk(n.Right)
			if err != nil { return err }
			c.emit(OpSub, 0)
		} else if n.Operator == "!" {
			err := c.walk(n.Right)
			if err != nil { return err }
			c.emit(OpNot, 0)
		}
	case *InfixExpression:
		if n.Operator == "&&" {
			err := c.walk(n.Left)
			if err != nil { return err }
			jumpFalse := c.emit(OpJumpIfFalse, 0)
			err = c.walk(n.Right)
			if err != nil { return err }
			c.emit(OpNot, 0)
			c.emit(OpNot, 0)
			jumpEnd := c.emit(OpJump, 0)
			c.patch(jumpFalse, int32(len(c.instructions)))
			c.emit(OpPush, c.addConstant(Value{Type: ValBool, Num: 0}))
			c.patch(jumpEnd, int32(len(c.instructions)))
			return nil
		}
		if n.Operator == "||" {
			err := c.walk(n.Left)
			if err != nil { return err }
			jumpTrue := c.emit(OpJumpIfTrue, 0)
			err = c.walk(n.Right)
			if err != nil { return err }
			c.emit(OpNot, 0)
			c.emit(OpNot, 0)
			jumpEnd := c.emit(OpJump, 0)
			c.patch(jumpTrue, int32(len(c.instructions)))
			c.emit(OpPush, c.addConstant(Value{Type: ValBool, Num: 1}))
			c.patch(jumpEnd, int32(len(c.instructions)))
			return nil
		}

		err := c.walk(n.Left)
		if err != nil { return err }
		err = c.walk(n.Right)
		if err != nil { return err }

		switch n.Operator {
		case "+": c.emit(OpAdd, 0)
		case "-": c.emit(OpSub, 0)
		case "*": c.emit(OpMul, 0)
		case "/": c.emit(OpDiv, 0)
		case "%": c.emit(OpMod, 0)
		case "==": c.emit(OpEqual, 0)
		case ">": c.emit(OpGreater, 0)
		case "<": c.emit(OpLess, 0)
		case ">=": c.emit(OpGreaterEqual, 0)
		case "<=": c.emit(OpLessEqual, 0)
		default: return fmt.Errorf("unknown operator: %s", n.Operator)
		}
	case *IfExpression:
		err := c.walk(n.Condition)
		if err != nil { return err }

		if n.IsSimple {
			return nil
		}

		jumpFalse := c.emit(OpJumpIfFalse, 0)
		err = c.walk(n.Consequence)
		if err != nil { return err }

		jumpEnd := c.emit(OpJump, 0)
		c.patch(jumpFalse, int32(len(c.instructions)))

		if n.Alternative != nil {
			err = c.walk(n.Alternative)
			if err != nil { return err }
		} else {
			c.emit(OpPush, c.addConstant(Value{Type: ValNil}))
		}
		c.patch(jumpEnd, int32(len(c.instructions)))

	case *AssignExpression:
		err := c.walk(n.Value)
		if err != nil { return err }
		c.emit(OpSetGlobal, c.addConstant(Value{Type: ValString, Str: n.Name.Value}))

	case *CallExpression:
		if ident, ok := n.Function.(*Identifier); ok && ident.Value == "concat" {
			for _, arg := range n.Arguments {
				err := c.walk(arg)
				if err != nil { return err }
			}
			c.emit(OpConcat, int32(len(n.Arguments)))
			return nil
		}

		for _, arg := range n.Arguments {
			err := c.walk(arg)
			if err != nil { return err }
		}
		if ident, ok := n.Function.(*Identifier); ok {
			c.emit(OpCall, c.addConstant(Value{Type: ValString, Str: ident.Value}))
			c.instructions[len(c.instructions)-1].Arg |= int32(len(n.Arguments)) << 16
		} else {
			return fmt.Errorf("calling non-identifier functions not supported in VM yet")
		}
	}
	return nil
}

func (c *VMCompiler) addConstant(v Value) int32 {
	var key any
	switch v.Type {
	case ValInt: key = int64(v.Num)
	case ValFloat: key = math.Float64frombits(v.Num)
	case ValBool: key = v.Num != 0
	case ValString: key = v.Str
	case ValNil: key = nil
	}
	if idx, ok := c.constMap[key]; ok {
		return idx
	}
	idx := int32(len(c.constants))
	c.constants = append(c.constants, v)
	c.constMap[key] = idx
	return idx
}

func (c *VMCompiler) emit(op OpCode, arg int32) int {
	c.instructions = append(c.instructions, vmInstruction{Op: op, Arg: arg})
	return len(c.instructions) - 1
}

func (c *VMCompiler) patch(pos int, arg int32) {
	c.instructions[pos].Arg = arg
}
