package uwasa

import (
	"testing"
)

func TestEval(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"5", int64(5)},
		{"10", int64(10)},
		{"true", true},
		{"false", false},
		{"-5", int64(-5)},
		{"-10", int64(-10)},
		{"5 + 5 + 5 + 5 - 10", int64(10)},
		{"2 * 2 * 2 * 2 * 2", int64(32)},
		{"-50 + 100 + -50", int64(0)},
		{"5 * 2 + 10", int64(20)},
		{"5 + 2 * 10", int64(25)},
		{"20 + 2 * -10", int64(0)},
		{"50 / 2 * 2 + 10", int64(60)},
		{"2 * (5 + 10)", int64(30)},
		{"3 * 3 * 3 + 10", int64(37)},
		{"3 * (3 * 3) + 10", int64(37)},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", int64(50)},
		{"true == true", true},
		{"false == false", true},
		{"true == false", false},
		{"true != false", true},
		{"false != true", true},
		{"(1 < 2) == true", true},
		{"(1 < 2) == false", false},
		{"(1 > 2) == true", false},
		{"(1 > 2) == false", true},
		{"if true is 10", int64(10)},
		{"if false is 10", nil},
		{"if true is 10 else 20", int64(10)},
		{"if false is 10 else 20", int64(20)},
		{"if 1 < 2 then 10", int64(10)},
		{"if 1 > 2 then 10 else 20", int64(20)},
		{"if 1 < 2 is 10 else if 1 > 2 is 20 else 30", int64(10)},
		{"!true", false},
		{"!false", true},
		{"!5", false},
		{"!!true", true},
		{"!!false", false},
		{"!!5", true},
		{"10 % 3", int64(1)},
		{"10 % 2", int64(0)},
		{"\"hello\" + \" \" + \"world\"", "hello world"},
		{"true && true", true},
		{"true && false", false},
		{"false && true", false},
		{"false && false", false},
		{"true || true", true},
		{"true || false", true},
		{"false || true", true},
		{"false || false", false},
		{"x = 10", int64(10)},
		{"concat(\"a\", \"b\", \"c\")", "abc"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()
		ctx := NewMapContext(make(map[string]any))

		result, err := Eval(program, ctx)
		if err != nil {
			t.Errorf("Eval(%q) error: %v", tt.input, err)
			continue
		}

		if result != tt.expected {
			t.Errorf("Eval(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}
