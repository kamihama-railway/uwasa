package uwasa

import (
	"testing"
)

func TestEvaluator(t *testing.T) {
	tests := []struct {
		input    string
		vars     map[string]any
		expected any
	}{
		{`if a == 0 && b >= 1`, map[string]any{"a": 0, "b": 1}, true},
		{`if a == 0 && b >= 1`, map[string]any{"a": 0, "b": 0}, false},
		{`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`, map[string]any{"a": 0}, "yes"},
		{`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`, map[string]any{"a": 1}, "ok"},
		{`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`, map[string]any{"a": 2}, "bad"},
		{`if a == 0 then b + 10`, map[string]any{"a": 0, "b": 5}, 15.0},
		{`if a == 0 then b + 10`, map[string]any{"a": 1, "b": 5}, nil},
		{`b = b + 10`, map[string]any{"b": 5}, 15.0},
		{`if (1 == 0) && (b = 20) is "ok" else is "no"`, map[string]any{"b": 5}, "no"},
		{`if (1 == 1) || (b = 20) is "ok" else is "no"`, map[string]any{"b": 5}, "ok"},
		{`if (1 == 0) || (1 == 1) is "ok" else is "no"`, nil, "ok"},
		{`if (1 == 0) || (1 == 0) is "ok" else is "no"`, nil, "no"},
		{`is_active = true`, map[string]any{"is_active": false}, true},
		{`is_active = false`, map[string]any{"is_active": true}, false},
	}

	for i, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Errorf("test[%d] %q has errors: %v", i, tt.input, p.Errors())
			continue
		}
		ctx := NewMapContext(tt.vars)
		result, err := Eval(program, ctx)
		if err != nil {
			t.Errorf("test[%d] %q eval error: %v", i, tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("test[%d] %q expected=%v, got=%v", i, tt.input, tt.expected, result)
		}
		if tt.input == `b = b + 10` {
			if ctx.Get("b") != 15.0 {
				t.Errorf("test[%d] variable b not updated correctly, got %v", i, ctx.Get("b"))
			}
		}
		if tt.input == `if (1 == 0) && (b = 20) is "ok" else is "no"` {
			if ctx.Get("b") != 5 {
				t.Errorf("test[%d] variable b should not be updated due to short-circuit (&&), got %v", i, ctx.Get("b"))
			}
		}
		if tt.input == `if (1 == 1) || (b = 20) is "ok" else is "no"` {
			if ctx.Get("b") != 5 {
				t.Errorf("test[%d] variable b should not be updated due to short-circuit (||), got %v", i, ctx.Get("b"))
			}
		}
		if tt.input == `is_active = true` {
			if ctx.Get("is_active") != true {
				t.Errorf("test[%d] is_active should be true, got %v", i, ctx.Get("is_active"))
			}
		}
		if tt.input == `is_active = false` {
			if ctx.Get("is_active") != false {
				t.Errorf("test[%d] is_active should be false, got %v", i, ctx.Get("is_active"))
			}
		}
	}
}

func TestEvaluatorErrors(t *testing.T) {
	tests := []struct {
		input string
		vars  map[string]any
	}{
		{`a + "string"`, map[string]any{"a": 1}},
		{`- "string"`, nil},
		{`a > "string"`, map[string]any{"a": 1}},
	}

	for i, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			continue // Expected for some cases like 1 * 2
		}
		ctx := NewMapContext(tt.vars)
		_, err := Eval(program, ctx)
		if err == nil {
			t.Errorf("test[%d] %q should have error", i, tt.input)
		}
	}
}

func TestEvaluatorTypeConversion(t *testing.T) {
	tests := []struct {
		input    string
		vars     map[string]any
		expected any
	}{
		{`a + 1`, map[string]any{"a": int32(10)}, 11.0},
		{`a + 1`, map[string]any{"a": int64(10)}, 11.0},
		{`a + 1`, map[string]any{"a": float32(10.5)}, 11.5},
		{`a == 10`, map[string]any{"a": int(10)}, true},
		// Actually if I use supported ones
		{`((a + b) - (c - d)) - e`, map[string]any{"a": 50, "b": 60, "c": 10, "d": 5, "e": 5}, 100.0},
	}

	for i, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()
		ctx := NewMapContext(tt.vars)
		result, err := Eval(program, ctx)
		if err != nil {
			t.Errorf("test[%d] %q error: %v", i, tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("test[%d] %q expected=%v, got=%v", i, tt.input, tt.expected, result)
		}
	}
}
