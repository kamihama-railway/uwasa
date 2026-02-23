package uwasa

import (
	"testing"
)

func TestParser(t *testing.T) {
	inputs := []string{
		`if a == 0 && b >= 1`,
		`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`,
		`if a == 0 then b + 10`,
		`b = b + 10`,
	}

	for _, input := range inputs {
		l := NewLexer(input)
		p := NewParser(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Errorf("input %q has errors: %v", input, p.Errors())
			continue
		}
		if program == nil {
			t.Errorf("input %q returned nil program", input)
			continue
		}
		t.Logf("input: %q, output: %s", input, program.String())
	}
}

func TestParserBooleanLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		boolLit, ok := program.(*BooleanLiteral)
		if !ok {
			t.Fatalf("program is not *BooleanLiteral, got=%T", program)
		}
		if boolLit.Value != tt.expected {
			t.Errorf("bool value wrong. expected=%v, got=%v", tt.expected, boolLit.Value)
		}
	}
}

func TestParserStructure(t *testing.T) {
	input := `if a == 0 then b = 1`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	ifExp, ok := program.(*IfExpression)
	if !ok {
		t.Fatalf("program is not *IfExpression, got=%T", program)
	}

	// Check Condition
	cond, ok := ifExp.Condition.(*InfixExpression)
	if !ok {
		t.Fatalf("condition is not *InfixExpression, got=%T", ifExp.Condition)
	}
	if cond.Operator != "==" {
		t.Errorf("cond operator wrong. expected='==', got=%q", cond.Operator)
	}
	left, ok := cond.Left.(*Identifier)
	if !ok || left.Value != "a" {
		t.Errorf("cond left wrong. expected='a', got=%v", cond.Left)
	}

	// Check Consequence
	assign, ok := ifExp.Consequence.(*AssignExpression)
	if !ok {
		t.Fatalf("consequence is not *AssignExpression, got=%T", ifExp.Consequence)
	}
	if assign.Name.Value != "b" {
		t.Errorf("assign name wrong. expected='b', got=%q", assign.Name.Value)
	}
	val, ok := assign.Value.(*NumberLiteral)
	if !ok || (val.IsInt && val.Int64Value != 1) || (!val.IsInt && val.Float64Value != 1.0) {
		t.Errorf("assign value wrong. expected=1, got=%v", assign.Value)
	}

	if !ifExp.IsThen {
		t.Errorf("IsThen should be true")
	}
}

func TestParserPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-a + b", "((-a) + b)"},
		{"a + b + c", "((a + b) + c)"},
		{"a + b - c", "((a + b) - c)"},
		{"a + b * c", "(a + (b * c))"},
		{"a + b == c", "((a + b) == c)"},
		{"a == b && c == d", "((a == b) && (c == d))"},
		{"a == b || c == d", "((a == b) || (c == d))"},
		{"a == b && c == d || e == f", "(((a == b) && (c == d)) || (e == f))"},
		{"a == b || c == d && e == f", "((a == b) || ((c == d) && (e == f)))"},
		{"(a == b || c == d) && e == f", "(((a == b) || (c == d)) && (e == f))"},
		{"a = b = c", "(a = (b = c))"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Errorf("input %q has errors: %v", tt.input, p.Errors())
			continue
		}
		if program.String() != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, program.String())
		}
	}
}

func TestParserErrors(t *testing.T) {
	tests := []string{
		"if",
		"if a == 0", // this is valid in my implementation as IsSimple
		"if a == 0 is",
		"if a == 0 is 1 else",
		"a +",
		"= 1",
	}

	for _, input := range tests {
		l := NewLexer(input)
		p := NewParser(l)
		p.ParseProgram()
		if input == "if a == 0" {
			if len(p.Errors()) != 0 {
				t.Errorf("input %q should not have errors, got: %v", input, p.Errors())
			}
			continue
		}
		if len(p.Errors()) == 0 {
			t.Errorf("input %q should have errors", input)
		}
	}
}

func TestParserNewSyntax(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"m.get(\"k\")", "m.get(k)"},
		{"m.has(\"k\")", "m.has(k)"},
		{"m.set(\"k\", 1)", "m.set(k, 1)"},
		{"m.del(\"k\")", "m.del(k)"},
		{"a = 1 => b = 2", "((a = 1) => (b = 2))"},
		{"a = 1 => b = 2 => c = 3", "(((a = 1) => (b = 2)) => (c = 3))"},
		{"if a == 0 then a = 1 => b = 2", "if (a == 0) then ((a = 1) => (b = 2))"},
		{"m.get(\"k\").get(\"s\")", ""}, // Should fail (not identifier)
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()

		if tt.expected == "" {
			if len(p.Errors()) == 0 {
				t.Errorf("input %q should have errors", tt.input)
			}
			continue
		}

		if len(p.Errors()) != 0 {
			t.Errorf("input %q has errors: %v", tt.input, p.Errors())
			continue
		}
		if program.String() != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, program.String())
		}
	}
}
