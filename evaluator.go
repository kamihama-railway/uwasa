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
		if n.IsInt {
			return n.Int64Value, nil
		}
		return n.Float64Value, nil
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
		switch r := right.(type) {
		case int64:
			return -r, nil
		case float64:
			return -r, nil
		case int:
			return -int64(r), nil
		}
		return nil, fmt.Errorf("unknown operator: -%T", right)
	default:
		return nil, fmt.Errorf("unknown operator: %s%T", operator, right)
	}
}

func evalInfixExpression(operator string, left, right any) (any, error) {
	switch operator {
	case "+", "-", "*", "/", "%":
		return evalArithmetic(operator, left, right)
	case "==", ">", "<", ">=", "<=":
		return evalComparison(operator, left, right)
	}
	return nil, fmt.Errorf("unknown operator: %T %s %T", left, operator, right)
}

func evalArithmetic(operator string, left, right any) (any, error) {
	// Fast path: both are int64
	il, okL := left.(int64)
	ir, okR := right.(int64)
	if okL && okR {
		switch operator {
		case "+": return il + ir, nil
		case "-": return il - ir, nil
		case "*": return il * ir, nil
		case "/":
			if ir == 0 { return nil, fmt.Errorf("division by zero") }
			return il / ir, nil
		case "%":
			if ir == 0 { return nil, fmt.Errorf("division by zero") }
			return il % ir, nil
		}
	}

	// String concatenation
	if operator == "+" {
		sl, okSL := left.(string)
		sr, okSR := right.(string)
		if okSL && okSR {
			return sl + sr, nil
		}
	}

	// Mixed or float
	fl, okFL := toFloat64(left)
	fr, okFR := toFloat64(right)
	if okFL && okFR {
		switch operator {
		case "+": return fl + fr, nil
		case "-": return fl - fr, nil
		case "*": return fl * fr, nil
		case "/":
			if fr == 0 { return nil, fmt.Errorf("division by zero") }
			return fl / fr, nil
		}
	}

	return nil, fmt.Errorf("invalid arithmetic: %T %s %T", left, operator, right)
}

func evalComparison(operator string, left, right any) (any, error) {
	// Fast path: both are int64
	il, okL := left.(int64)
	ir, okR := right.(int64)
	if okL && okR {
		switch operator {
		case "==": return boolToAny(il == ir), nil
		case ">":  return boolToAny(il > ir), nil
		case "<":  return boolToAny(il < ir), nil
		case ">=": return boolToAny(il >= ir), nil
		case "<=": return boolToAny(il <= ir), nil
		}
	}

	// Mixed or float
	fl, okFL := toFloat64(left)
	fr, okFR := toFloat64(right)
	if okFL && okFR {
		switch operator {
		case "==": return boolToAny(fl == fr), nil
		case ">":  return boolToAny(fl > fr), nil
		case "<":  return boolToAny(fl < fr), nil
		case ">=": return boolToAny(fl >= fr), nil
		case "<=": return boolToAny(fl <= fr), nil
		}
	}

	if operator == "==" {
		return boolToAny(left == right), nil
	}

	return nil, fmt.Errorf("invalid comparison: %T %s %T", left, operator, right)
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64: return val, true
	case int64:   return float64(val), true
	case int:     return float64(val), true
	case float32: return float64(val), true
	case int32:   return float64(val), true
	}
	return 0, false
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
