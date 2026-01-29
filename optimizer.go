// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import "fmt"

func Fold(node Node) Node {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *PrefixExpression:
		foldedRight := Fold(n.Right)
		if foldedRight != nil {
			n.Right = foldedRight.(Expression)
		}
		if right, ok := n.Right.(*NumberLiteral); ok {
			if n.Operator == "-" {
				if right.IsInt {
					return &NumberLiteral{Int64Value: -right.Int64Value, IsInt: true}
				}
				return &NumberLiteral{Float64Value: -right.Float64Value, IsInt: false}
			}
		}
	case *InfixExpression:
		foldedLeft := Fold(n.Left)
		if foldedLeft != nil {
			n.Left = foldedLeft.(Expression)
		}
		foldedRight := Fold(n.Right)
		if foldedRight != nil {
			n.Right = foldedRight.(Expression)
		}

		left, okL := n.Left.(*NumberLiteral)
		right, okR := n.Right.(*NumberLiteral)

		// Compile-time string concatenation
		if n.Operator == "+" {
			leftS, okLS := n.Left.(*StringLiteral)
			rightS, okRS := n.Right.(*StringLiteral)
			if okLS && okRS {
				return &StringLiteral{Value: leftS.Value + rightS.Value}
			}
		}

		if okL && okR {
			switch n.Operator {
			case "+":
				if left.IsInt && right.IsInt {
					return &NumberLiteral{Int64Value: left.Int64Value + right.Int64Value, IsInt: true}
				}
				lv, rv := getFloatValues(left, right)
				return &NumberLiteral{Float64Value: lv + rv, IsInt: false}
			case "-":
				if left.IsInt && right.IsInt {
					return &NumberLiteral{Int64Value: left.Int64Value - right.Int64Value, IsInt: true}
				}
				lv, rv := getFloatValues(left, right)
				return &NumberLiteral{Float64Value: lv - rv, IsInt: false}
			case "*":
				if left.IsInt && right.IsInt {
					return &NumberLiteral{Int64Value: left.Int64Value * right.Int64Value, IsInt: true}
				}
				lv, rv := getFloatValues(left, right)
				return &NumberLiteral{Float64Value: lv * rv, IsInt: false}
			case "/":
				if left.IsInt && right.IsInt && right.Int64Value != 0 {
					return &NumberLiteral{Int64Value: left.Int64Value / right.Int64Value, IsInt: true}
				}
				lv, rv := getFloatValues(left, right)
				if rv != 0 {
					return &NumberLiteral{Float64Value: lv / rv, IsInt: false}
				}
			case "%":
				if left.IsInt && right.IsInt && right.Int64Value != 0 {
					return &NumberLiteral{Int64Value: left.Int64Value % right.Int64Value, IsInt: true}
				}
			case "==":
				if left.IsInt && right.IsInt {
					return &BooleanLiteral{Value: left.Int64Value == right.Int64Value}
				}
				lv, rv := getFloatValues(left, right)
				return &BooleanLiteral{Value: lv == rv}
			case ">":
				if left.IsInt && right.IsInt {
					return &BooleanLiteral{Value: left.Int64Value > right.Int64Value}
				}
				lv, rv := getFloatValues(left, right)
				return &BooleanLiteral{Value: lv > rv}
			case "<":
				if left.IsInt && right.IsInt {
					return &BooleanLiteral{Value: left.Int64Value < right.Int64Value}
				}
				lv, rv := getFloatValues(left, right)
				return &BooleanLiteral{Value: lv < rv}
			case ">=":
				if left.IsInt && right.IsInt {
					return &BooleanLiteral{Value: left.Int64Value >= right.Int64Value}
				}
				lv, rv := getFloatValues(left, right)
				return &BooleanLiteral{Value: lv >= rv}
			case "<=":
				if left.IsInt && right.IsInt {
					return &BooleanLiteral{Value: left.Int64Value <= right.Int64Value}
				}
				lv, rv := getFloatValues(left, right)
				return &BooleanLiteral{Value: lv <= rv}
			}
		}

		// Handle Boolean logic folding
		leftB, okLB := n.Left.(*BooleanLiteral)
		rightB, okRB := n.Right.(*BooleanLiteral)

		if n.Operator == "&&" {
			if okLB {
				if !leftB.Value {
					return &BooleanLiteral{Value: false}
				}
				return n.Right
			}
			if okRB && rightB.Value {
				return n.Left
			}
		}
		if n.Operator == "||" {
			if okLB {
				if leftB.Value {
					return &BooleanLiteral{Value: true}
				}
				return n.Right
			}
			if okRB && !rightB.Value {
				return n.Left
			}
		}

		if okLB && okRB && n.Operator == "==" {
			return &BooleanLiteral{Value: leftB.Value == rightB.Value}
		}

	case *IfExpression:
		foldedCond := Fold(n.Condition)
		if foldedCond != nil {
			n.Condition = foldedCond.(Expression)
		}
		if n.Consequence != nil {
			foldedCons := Fold(n.Consequence)
			if foldedCons != nil {
				n.Consequence = foldedCons.(Expression)
			}
		}
		if n.Alternative != nil {
			foldedAlt := Fold(n.Alternative)
			if foldedAlt != nil {
				n.Alternative = foldedAlt.(Expression)
			}
		}

		if cond, ok := n.Condition.(*BooleanLiteral); ok {
			if n.IsSimple {
				return cond
			}
			if cond.Value {
				return n.Consequence
			} else if n.Alternative != nil {
				return n.Alternative
			} else {
				// if false then ... (nothing)
				return nil
			}
		}
	case *CallExpression:
		allConst := true
		for i, arg := range n.Arguments {
			folded := Fold(arg)
			if folded != nil {
				n.Arguments[i] = folded.(Expression)
			}
			// We consider StringLiteral, NumberLiteral, BooleanLiteral as constants
			switch n.Arguments[i].(type) {
			case *StringLiteral, *NumberLiteral, *BooleanLiteral:
			default:
				allConst = false
			}
		}
		// If it's a call to "concat" with all constant arguments, we can fold it
		if ident, ok := n.Function.(*Identifier); ok && ident.Value == "concat" && allConst {
			res := ""
			for _, arg := range n.Arguments {
				switch a := arg.(type) {
				case *StringLiteral:
					res += a.Value
				case *NumberLiteral:
					if a.IsInt {
						res += fmt.Sprintf("%d", a.Int64Value)
					} else {
						res += fmt.Sprintf("%g", a.Float64Value)
					}
				case *BooleanLiteral:
					res += fmt.Sprintf("%v", a.Value)
				}
			}
			return &StringLiteral{Value: res}
		}

	case *AssignExpression:
		foldedVal := Fold(n.Value)
		if foldedVal != nil {
			n.Value = foldedVal.(Expression)
		}
	}
	return node
}

func getFloatValues(l, r *NumberLiteral) (float64, float64) {
	var lv, rv float64
	if l.IsInt {
		lv = float64(l.Int64Value)
	} else {
		lv = l.Float64Value
	}
	if r.IsInt {
		rv = float64(r.Int64Value)
	} else {
		rv = r.Float64Value
	}
	return lv, rv
}
