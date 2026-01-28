// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
)

var (
	trueVal  any = true
	falseVal any = false
)

func boolToAny(b bool) any {
	if b {
		return trueVal
	}
	return falseVal
}

func Eval(node Node, ctx Context) (any, error) {
	switch n := node.(type) {
	case *Identifier:
		val, _ := ctx.Get(n.Value)
		return val, nil
	case *NumberLiteral:
		return n.Value, nil
	case *StringLiteral:
		return n.Value, nil
	case *BooleanLiteral:
		return boolToAny(n.Value), nil
	case *PrefixExpression:
		right, err := Eval(n.Right, ctx)
		if err != nil {
			return nil, err
		}
		return evalPrefixExpression(n.Operator, right)
	case *InfixExpression:
		if n.Operator == "&&" {
			left, err := Eval(n.Left, ctx)
			if err != nil {
				return nil, err
			}
			if !isTruthy(left) {
				return falseVal, nil
			}
			right, err := Eval(n.Right, ctx)
			if err != nil {
				return nil, err
			}
			return boolToAny(isTruthy(right)), nil
		}
		if n.Operator == "||" {
			left, err := Eval(n.Left, ctx)
			if err != nil {
				return nil, err
			}
			if isTruthy(left) {
				return trueVal, nil
			}
			right, err := Eval(n.Right, ctx)
			if err != nil {
				return nil, err
			}
			return boolToAny(isTruthy(right)), nil
		}
		left, err := Eval(n.Left, ctx)
		if err != nil {
			return nil, err
		}
		right, err := Eval(n.Right, ctx)
		if err != nil {
			return nil, err
		}
		return evalInfixExpression(n.Operator, left, right)
	case *IfExpression:
		return evalIfExpression(n, ctx)
	case *AssignExpression:
		val, err := Eval(n.Value, ctx)
		if err != nil {
			return nil, err
		}
		err = ctx.Set(n.Name.Value, val)
		return val, err
	}
	return nil, nil
}

func evalPrefixExpression(operator string, right any) (any, error) {
	switch operator {
	case "-":
		if r, ok := right.(float64); ok {
			return -r, nil
		}
		return nil, fmt.Errorf("unknown operator: -%T", right)
	default:
		return nil, fmt.Errorf("unknown operator: %s%T", operator, right)
	}
}

func evalInfixExpression(operator string, left, right any) (any, error) {
	switch {
	case operator == "==" || operator == ">" || operator == "<" || operator == ">=" || operator == "<=":
		fl, okL := toFloat64(left)
		fr, okR := toFloat64(right)
		if okL && okR {
			return evalNumberComparison(operator, fl, fr)
		}
		if operator == "==" {
			return boolToAny(left == right), nil
		}
	case operator == "+" || operator == "-":
		fl, okL := toFloat64(left)
		fr, okR := toFloat64(right)
		if okL && okR {
			return evalNumberArithmetic(operator, fl, fr)
		}
		if operator == "+" {
			sl, okSL := left.(string)
			sr, okSR := right.(string)
			if okSL && okSR {
				return sl + sr, nil
			}
		}
	}

	return nil, fmt.Errorf("unknown operator: %T %s %T", left, operator, right)
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	}
	return 0, false
}

func evalNumberComparison(operator string, left, right float64) (any, error) {
	switch operator {
	case "==":
		return boolToAny(left == right), nil
	case ">":
		return boolToAny(left > right), nil
	case "<":
		return boolToAny(left < right), nil
	case ">=":
		return boolToAny(left >= right), nil
	case "<=":
		return boolToAny(left <= right), nil
	}
	return nil, fmt.Errorf("unknown comparison operator: %s", operator)
}

func evalNumberArithmetic(operator string, left, right float64) (any, error) {
	switch operator {
	case "+":
		return left + right, nil
	case "-":
		return left - right, nil
	}
	return nil, fmt.Errorf("unknown arithmetic operator: %s", operator)
}


func evalIfExpression(ie *IfExpression, ctx Context) (any, error) {
	cond, err := Eval(ie.Condition, ctx)
	if err != nil {
		return nil, err
	}

	if ie.IsSimple {
		return boolToAny(isTruthy(cond)), nil
	}

	if isTruthy(cond) {
		return Eval(ie.Consequence, ctx)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, ctx)
	}

	return nil, nil
}

func isTruthy(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case nil:
		return false
	default:
		return true
	}
}
