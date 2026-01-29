package uwasa

import (
	"testing"
)

func TestParseNumberLiteral(t *testing.T) {
	input := "5"
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Errorf("parser errors: %v", p.Errors())
	}

	literal, ok := program.(*NumberLiteral)
	if !ok {
		t.Fatalf("program is not NumberLiteral, got=%T", program)
	}
	if literal.Int64Value != 5 {
		t.Errorf("literal.Int64Value not %d, got=%d", 5, literal.Int64Value)
	}
}

func TestParseInfixExpression(t *testing.T) {
	tests := []struct {
		input      string
		leftValue  int64
		operator   string
		rightValue int64
	}{
		{"5 + 5", 5, "+", 5},
		{"5 - 5", 5, "-", 5},
		{"5 * 5", 5, "*", 5},
		{"5 / 5", 5, "/", 5},
		{"5 > 5", 5, ">", 5},
		{"5 < 5", 5, "<", 5},
		{"5 == 5", 5, "==", 5},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Errorf("parser errors: %v", p.Errors())
		}

		exp, ok := program.(*InfixExpression)
		if !ok {
			t.Fatalf("program is not InfixExpression, got=%T", program)
		}

		if !testNumberLiteral(t, exp.Left, tt.leftValue) {
			return
		}

		if exp.Operator != tt.operator {
			t.Fatalf("exp.Operator is not '%s', got=%s", tt.operator, exp.Operator)
		}

		if !testNumberLiteral(t, exp.Right, tt.rightValue) {
			return
		}
	}
}

func testNumberLiteral(t *testing.T, il Expression, value int64) bool {
	ni, ok := il.(*NumberLiteral)
	if !ok {
		t.Errorf("il not *NumberLiteral, got=%T", il)
		return false
	}
	if ni.Int64Value != value {
		t.Errorf("ni.Int64Value not %d, got=%d", value, ni.Int64Value)
		return false
	}
	return true
}

func TestParseIfExpression(t *testing.T) {
	input := "if x < y then x else y"
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Errorf("parser errors: %v", p.Errors())
	}

	exp, ok := program.(*IfExpression)
	if !ok {
		t.Fatalf("program is not IfExpression, got=%T", program)
	}

	if exp.Condition.String() != "(x < y)" {
		t.Errorf("condition.String() not %s, got=%s", "(x < y)", exp.Condition.String())
	}

	if exp.Consequence.String() != "x" {
		t.Errorf("consequence.String() not %s, got=%s", "x", exp.Consequence.String())
	}

	if exp.Alternative.String() != "y" {
		t.Errorf("alternative.String() not %s, got=%s", "y", exp.Alternative.String())
	}
}
