// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package ast

func IsZero(node Node) bool {
	if n, ok := node.(*NumberLiteral); ok {
		return (n.IsInt && n.Int64Value == 0) || (!n.IsInt && n.Float64Value == 0)
	}
	return false
}

func IsOne(node Node) bool {
	if n, ok := node.(*NumberLiteral); ok {
		return (n.IsInt && n.Int64Value == 1) || (!n.IsInt && n.Float64Value == 1)
	}
	return false
}

func IsSameIdentifier(l, r Node) bool {
	il, okL := l.(*Identifier)
	ir, okR := r.(*Identifier)
	return okL && okR && il.Value == ir.Value
}

func HasSideEffects(node Node) bool {
	switch n := node.(type) {
	case *AssignExpression:
		return true
	case *CallExpression:
		return true // Assume functions have side effects
	case *InfixExpression:
		return HasSideEffects(n.Left) || HasSideEffects(n.Right)
	case *PrefixExpression:
		return HasSideEffects(n.Right)
	case *IfExpression:
		res := HasSideEffects(n.Condition)
		if n.Consequence != nil { res = res || HasSideEffects(n.Consequence) }
		if n.Alternative != nil { res = res || HasSideEffects(n.Alternative) }
		return res
	}
	return false
}
