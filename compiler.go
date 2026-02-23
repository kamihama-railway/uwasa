// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import "fmt"

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
		if isSameIdentifier(n.Name, n.Value) {
			return n.Name
		}
		return n

	default:
		return n
	}
}

func (o *Recompiler) simplifyPrefix(pe *PrefixExpression) Node {
	// 静态类型检查
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

	// 代数简化
	switch ie.Operator {
	case "+":
		if isZero(left) {
			return right
		}
		if isZero(right) {
			return left
		}
	case "-":
		if isZero(right) {
			return left
		}
		if isSameIdentifier(left, right) {
			return &NumberLiteral{Int64Value: 0, IsInt: true}
		}
	case "=":
		if isSameIdentifier(left, right) {
			// a = a is redundant
			// We might want to keep it or flag it. Let's flag it.
			// However, in rules it might be used to ensure a variable exists.
			// For now, let's just simplify it to 'a'
			return left
		}
	case "*":
		if isZero(left) {
			return &NumberLiteral{Int64Value: 0, IsInt: true}
		}
		if isZero(right) {
			return &NumberLiteral{Int64Value: 0, IsInt: true}
		}
		if isOne(left) {
			return right
		}
		if isOne(right) {
			return left
		}
	case "/":
		if isZero(right) {
			o.errors = append(o.errors, "division by zero")
			return ie
		}
		if isOne(right) {
			return left
		}
		if isSameIdentifier(left, right) && !hasSideEffects(left) {
			return &NumberLiteral{Int64Value: 1, IsInt: true}
		}
	case "==":
		if isSameIdentifier(left, right) && !hasSideEffects(left) {
			return &BooleanLiteral{Value: true}
		}
		// Constant comparison
		if l, okL := left.(*NumberLiteral); okL {
			if r, okR := right.(*NumberLiteral); okR {
				lv, rv := getFloatValues(l, r)
				if lv != rv {
					// We can fold this, but Fold() already does it for literals.
					// This is more for cases where literals were produced by other simplifications.
				}
			}
		}
	case "!=": // 虽然还没定义这个 Token，但为了完整性考虑
		if isSameIdentifier(left, right) && !hasSideEffects(left) {
			return &BooleanLiteral{Value: false}
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
		// Flag obvious non-boolean types in logic
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
				// We don't necessarily error out, but we could warn.
				// For now, static analysis in this context usually means error/flag.
				// The user asked for "检查功能" (check functions).
			}
		} else {
			if ie.Alternative != nil {
				// Alternative is unreachable
			}
		}
	}
}

func isZero(n Node) bool {
	lit, ok := n.(*NumberLiteral)
	if !ok {
		return false
	}
	if lit.IsInt {
		return lit.Int64Value == 0
	}
	return lit.Float64Value == 0
}

func isOne(n Node) bool {
	lit, ok := n.(*NumberLiteral)
	if !ok {
		return false
	}
	if lit.IsInt {
		return lit.Int64Value == 1
	}
	return lit.Float64Value == 1
}

func isSameIdentifier(left, right Node) bool {
	l, okL := left.(*Identifier)
	r, okR := right.(*Identifier)
	if !okL || !okR {
		return false
	}
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
	if node == nil {
		return
	}
	fn(node)
	switch n := node.(type) {
	case *PrefixExpression:
		walk(n.Right, fn)
	case *InfixExpression:
		walk(n.Left, fn)
		walk(n.Right, fn)
	case *IfExpression:
		walk(n.Condition, fn)
		walk(n.Consequence, fn)
		walk(n.Alternative, fn)
	case *AssignExpression:
		walk(n.Value, fn)
	case *CallExpression:
		walk(n.Function, fn)
		for _, arg := range n.Arguments {
			walk(arg, fn)
		}
	}
}
