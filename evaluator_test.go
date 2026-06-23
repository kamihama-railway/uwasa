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
		{`if a == 0 then b + 10`, map[string]any{"a": int64(0), "b": int64(5)}, int64(15)},
		{`if a == 0 then b + 10`, map[string]any{"a": int64(1), "b": int64(5)}, nil},
		{`b = b + 10`, map[string]any{"b": int64(5)}, int64(15)},
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

		// Additional checks for variable updates
		if tt.input == `b = b + 10` {
			val, _ := ctx.Get("b")
			if val != int64(15) {
				t.Errorf("test[%d] variable b not updated correctly, got %T %v", i, val, val)
			}
		}
		if tt.input == `if (1 == 0) && (b = 20) is "ok" else is "no"` {
			val, _ := ctx.Get("b")
			if val != 5 {
				t.Errorf("test[%d] variable b should not be updated due to short-circuit (&&), got %v", i, val)
			}
		}
		if tt.input == `if (1 == 1) || (b = 20) is "ok" else is "no"` {
			val, _ := ctx.Get("b")
			if val != 5 {
				t.Errorf("test[%d] variable b should not be updated due to short-circuit (||), got %v", i, val)
			}
		}
	}
}

func TestEvaluatorNewSyntax(t *testing.T) {
	tests := []struct {
		input    string
		vars     map[string]any
		expected any
	}{
		{`m.set("a", 1) => m.get("a")`, map[string]any{"m": make(map[string]any)}, int64(1)},
		{`m.set("a", 1) => m.has("a")`, map[string]any{"m": make(map[string]any)}, true},
		{`m.set("a", 1) => m.del("a") => m.has("a")`, map[string]any{"m": make(map[string]any)}, false},
		{`if m.has("a") then m.get("a") else is "none"`, map[string]any{"m": map[string]any{"a": "ok"}}, "ok"},
		{`if m.has("b") then m.get("b") else is "none"`, map[string]any{"m": map[string]any{"a": "ok"}}, "none"},
		{`a = 1 => a = a + 1 => a`, map[string]any{"a": 0}, int64(2)},
		{`m.set("x", 10) => m.set("y", 20) => m.get("x") + m.get("y")`, map[string]any{"m": make(map[string]any)}, int64(30)},
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
			t.Errorf("test[%d] %q expected=%v (%T), got=%v (%T)", i, tt.input, tt.expected, tt.expected, result, result)
		}
	}
}

func TestEvalNil(t *testing.T) {
	result, err := Eval(nil, nil)
	if err != nil {
		t.Errorf("Eval(nil, nil) error: %v", err)
	}
	if result != nil {
		t.Errorf("Eval(nil, nil) expected nil, got %v", result)
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
			continue
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
		{`a + 1`, map[string]any{"a": int64(10)}, int64(11)},
		{`a + 1`, map[string]any{"a": float32(10.5)}, 11.5},
		{`a == 10`, map[string]any{"a": int(10)}, true},
		{`((a + b) - (c - d)) - e`, map[string]any{
			"a": int64(50), "b": int64(60), "c": int64(10), "d": int64(5), "e": int64(5),
		}, int64(100)},
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
