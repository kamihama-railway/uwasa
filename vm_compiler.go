// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import "fmt"

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
	return &RenderedBytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
	}, nil
}

func (c *VMCompiler) CompileOptimized(node Node, opts EngineOptions) (*RenderedBytecode, error) {
	optimized := node
	if opts.OptimizationLevel >= OptBasic {
		optimized = Fold(optimized)
	}

	if opts.UseRecompiler {
		// Use a dedicated VM optimization pass that also does static analysis
		var err error
		optimized, err = c.optimize(optimized)
		if err != nil {
			return nil, err
		}
	}

	return c.Compile(optimized)
}

func (c *VMCompiler) optimize(node Node) (Node, error) {
	// Integrated VM optimizer (similar to Recompiler but specifically for VM path)
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

		// Algebraic simplification
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
		c.emit(OpGetGlobal, c.addConstant(Value{Type: ValString, String: n.Value}))
	case *NumberLiteral:
		if n.IsInt {
			c.emit(OpPush, c.addConstant(Value{Type: ValInt, Int: n.Int64Value}))
		} else {
			c.emit(OpPush, c.addConstant(Value{Type: ValFloat, Float: n.Float64Value}))
		}
	case *StringLiteral:
		c.emit(OpPush, c.addConstant(Value{Type: ValString, String: n.Value}))
	case *BooleanLiteral:
		c.emit(OpPush, c.addConstant(Value{Type: ValBool, Bool: n.Value}))
	case *PrefixExpression:
		if n.Operator == "-" {
			// -x is 0 - x or we can have a dedicated OpNeg. Let's use Push 0 and Sub for simplicity or OpPush -Val.
			// Actually let's just push 0 and sub if it's numeric literal, or better, implement OpSub.
			// For -Identifier, we need to push 0 then the identifier then sub.
			c.emit(OpPush, c.addConstant(Value{Type: ValInt, Int: 0}))
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
			c.emit(OpPush, c.addConstant(Value{Type: ValBool, Bool: false}))
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
			c.emit(OpPush, c.addConstant(Value{Type: ValBool, Bool: true}))
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
			return nil // Result is already on stack from condition
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
		c.emit(OpSetGlobal, c.addConstant(Value{Type: ValString, String: n.Name.Value}))

	case *CallExpression:
		for _, arg := range n.Arguments {
			err := c.walk(arg)
			if err != nil { return err }
		}
		if ident, ok := n.Function.(*Identifier); ok {
			c.emit(OpCall, c.addConstant(Value{Type: ValString, String: ident.Value}))
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
	case ValInt: key = v.Int
	case ValFloat: key = v.Float
	case ValBool: key = v.Bool
	case ValString: key = v.String
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
