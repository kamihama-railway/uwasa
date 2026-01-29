// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"github.com/kamihama-railway/uwasa/ast"
)

// Recompiler 进行更激进的代数简化和静态检查
type Recompiler struct {
	errors []string
}

func NewRecompiler() *Recompiler {
	return &Recompiler{}
}

func (o *Recompiler) Optimize(node Node) (Node, error) {
	optimized := o.simplify(node)
	if len(o.errors) > 0 {
		return nil, fmt.Errorf("static analysis errors: %v", o.errors)
	}
	return optimized, nil
}

func (o *Recompiler) simplify(node Node) Node {
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
		o.checkTypeMismatch(n)
		return o.simplifyInfix(n)

	case *IfExpression:
		o.checkUnreachable(n)
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
		if ast.IsSameIdentifier(n.Name, n.Value) {
			return n.Name
		}
		return n

	default:
		return n
	}
}

func (o *Recompiler) simplifyPrefix(pe *PrefixExpression) Node {
	if pe.Operator == "-" {
		if _, ok := pe.Right.(*StringLiteral); ok {
			o.errors = append(o.errors, "invalid operation: -string")
		}
	}
	return pe
}

func (o *Recompiler) simplifyInfix(ie *InfixExpression) Node {
	left := ie.Left
	right := ie.Right

	switch ie.Operator {
	case "+":
		if ast.IsZero(left) { return right }
		if ast.IsZero(right) { return left }
	case "-":
		if ast.IsZero(right) { return left }
		if ast.IsSameIdentifier(left, right) {
			return &NumberLiteral{Int64Value: 0, IsInt: true}
		}
	case "=":
		if ast.IsSameIdentifier(left, right) {
			return left
		}
	case "*":
		if ast.IsZero(left) { return &NumberLiteral{Int64Value: 0, IsInt: true} }
		if ast.IsZero(right) { return &NumberLiteral{Int64Value: 0, IsInt: true} }
		if ast.IsOne(left) { return right }
		if ast.IsOne(right) { return left }
	case "/":
		if ast.IsZero(right) {
			o.errors = append(o.errors, "division by zero")
			return ie
		}
		if ast.IsOne(right) { return left }
		if ast.IsSameIdentifier(left, right) && !ast.HasSideEffects(left) {
			return &NumberLiteral{Int64Value: 1, IsInt: true}
		}
	case "==":
		if ast.IsSameIdentifier(left, right) && !ast.HasSideEffects(left) {
			return &BooleanLiteral{Value: true}
		}
	}

	return ie
}

func (o *Recompiler) checkTypeMismatch(ie *InfixExpression) {
	left := ie.Left
	right := ie.Right
	_, okLS := left.(*StringLiteral)
	_, okRS := right.(*StringLiteral)
	_, okLN := left.(*NumberLiteral)
	_, okRN := right.(*NumberLiteral)
	_, okLB := left.(*BooleanLiteral)
	_, okRB := right.(*BooleanLiteral)

	switch ie.Operator {
	case "-", "*", "/", "%", ">", "<", ">=", "<=":
		if okLS || okRS {
			o.errors = append(o.errors, fmt.Sprintf("invalid operation: string %s string/number", ie.Operator))
		}
		if okLB || okRB {
			o.errors = append(o.errors, fmt.Sprintf("invalid operation: boolean %s boolean/number", ie.Operator))
		}
	case "+":
		if (okLS && okRN) || (okLN && okRS) {
			o.errors = append(o.errors, "invalid operation: string + number mismatch")
		}
		if okLB || okRB {
			o.errors = append(o.errors, "invalid operation: boolean + any")
		}
	case "&&", "||":
		if okLN || okRN || okLS || okRS {
			o.errors = append(o.errors, fmt.Sprintf("invalid logic operation: %s used with non-boolean literal", ie.Operator))
		}
	}
}

func (o *Recompiler) checkUnreachable(ie *IfExpression) {
	if cond, ok := ie.Condition.(*BooleanLiteral); ok {
		if !cond.Value {
			if ie.Consequence != nil {
				// Consequence is unreachable
			}
		} else {
			if ie.Alternative != nil {
				// Alternative is unreachable
			}
		}
	}
}
