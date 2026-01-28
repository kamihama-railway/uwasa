// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import "fmt"

// AggressiveOptimizer 进行更激进的代数简化和静态检查
type AggressiveOptimizer struct {
	errors []string
}

func NewAggressiveOptimizer() *AggressiveOptimizer {
	return &AggressiveOptimizer{}
}

func (o *AggressiveOptimizer) Optimize(node Node) (Node, error) {
	optimized := o.simplify(node)
	if len(o.errors) > 0 {
		return nil, fmt.Errorf("static analysis errors: %v", o.errors)
	}
	return optimized, nil
}

func (o *AggressiveOptimizer) simplify(node Node) Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *PrefixExpression:
		n.Right = o.simplify(n.Right).(Expression)
		return o.simplifyPrefix(n)

	case *InfixExpression:
		n.Left = o.simplify(n.Left).(Expression)
		n.Right = o.simplify(n.Right).(Expression)
		return o.simplifyInfix(n)

	case *IfExpression:
		n.Condition = o.simplify(n.Condition).(Expression)
		if n.Consequence != nil {
			n.Consequence = o.simplify(n.Consequence).(Expression)
		}
		if n.Alternative != nil {
			n.Alternative = o.simplify(n.Alternative).(Expression)
		}
		return n

	case *AssignExpression:
		n.Value = o.simplify(n.Value).(Expression)
		return n

	default:
		return n
	}
}

func (o *AggressiveOptimizer) simplifyPrefix(pe *PrefixExpression) Node {
	// 静态类型检查
	if pe.Operator == "-" {
		if _, ok := pe.Right.(*StringLiteral); ok {
			o.errors = append(o.errors, "invalid operation: -string")
		}
	}
	return pe
}

func (o *AggressiveOptimizer) simplifyInfix(ie *InfixExpression) Node {
	left := ie.Left
	right := ie.Right

	// 1. 静态类型检查 (字面量级别)
	o.checkTypeMismatch(ie)

	// 2. 代数简化
	switch ie.Operator {
	case "+":
		if isZero(left) { return right }
		if isZero(right) { return left }
	case "-":
		if isZero(right) { return left }
		if isSameIdentifier(left, right) {
			return &NumberLiteral{Int64Value: 0, IsInt: true}
		}
	case "*":
		if isZero(left) { return &NumberLiteral{Int64Value: 0, IsInt: true} }
		if isZero(right) { return &NumberLiteral{Int64Value: 0, IsInt: true} }
		if isOne(left) { return right }
		if isOne(right) { return left }
	case "/":
		if isZero(right) {
			o.errors = append(o.errors, "division by zero")
			return ie
		}
		if isOne(right) { return left }
		if isSameIdentifier(left, right) && !hasSideEffects(left) {
			return &NumberLiteral{Int64Value: 1, IsInt: true}
		}
	case "==":
		if isSameIdentifier(left, right) && !hasSideEffects(left) {
			return &BooleanLiteral{Value: true}
		}
	case "!=": // 虽然还没定义这个 Token，但为了完整性考虑
		if isSameIdentifier(left, right) && !hasSideEffects(left) {
			return &BooleanLiteral{Value: false}
		}
	}

	return ie
}

func (o *AggressiveOptimizer) checkTypeMismatch(ie *InfixExpression) {
	_, okLS := ie.Left.(*StringLiteral)
	_, okRS := ie.Right.(*StringLiteral)
	_, okLN := ie.Left.(*NumberLiteral)
	_, okRN := ie.Right.(*NumberLiteral)

	switch ie.Operator {
	case "-", "*", "/", "%", ">", "<", ">=", "<=":
		if okLS || okRS {
			o.errors = append(o.errors, fmt.Sprintf("invalid operation: string %s string/number", ie.Operator))
		}
	case "+":
		if (okLS && okRN) || (okLN && okRS) {
			o.errors = append(o.errors, "invalid operation: string + number mismatch")
		}
	}
}

func isZero(n Node) bool {
	lit, ok := n.(*NumberLiteral)
	if !ok { return false }
	if lit.IsInt { return lit.Int64Value == 0 }
	return lit.Float64Value == 0
}

func isOne(n Node) bool {
	lit, ok := n.(*NumberLiteral)
	if !ok { return false }
	if lit.IsInt { return lit.Int64Value == 1 }
	return lit.Float64Value == 1
}

func isSameIdentifier(left, right Node) bool {
	l, okL := left.(*Identifier)
	r, okR := right.(*Identifier)
	if !okL || !okR { return false }
	return l.Value == r.Value
}

func hasSideEffects(n Node) bool {
	// 目前只有 AssignExpression 有副作用
	// 递归检查
	var found bool
	walk(n, func(node Node) {
		if _, ok := node.(*AssignExpression); ok {
			found = true
		}
	})
	return found
}

func walk(node Node, fn func(Node)) {
	if node == nil { return }
	fn(node)
	switch n := node.(type) {
	case *PrefixExpression: walk(n.Right, fn)
	case *InfixExpression:
		walk(n.Left, fn)
		walk(n.Right, fn)
	case *IfExpression:
		walk(n.Condition, fn)
		walk(n.Consequence, fn)
		walk(n.Alternative, fn)
	case *AssignExpression:
		walk(n.Value, fn)
	}
}
