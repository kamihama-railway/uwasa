package uwasa

import (
	"fmt"
	"testing"
)

func TestMapAndSequenceDetailed(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     func() map[string]any
		expected any
		wantErr  bool
	}{
		{
			"Get nonexistent key",
			`m.get("none")`,
			func() map[string]any { return map[string]any{"m": make(map[string]any)} },
			nil,
			false,
		},
		{
			"Has nonexistent key",
			`m.has("none")`,
			func() map[string]any { return map[string]any{"m": make(map[string]any)} },
			false,
			false,
		},
		{
			"Del nonexistent key (no error)",
			`m.del("none") => m.has("none")`,
			func() map[string]any { return map[string]any{"m": make(map[string]any)} },
			false,
			false,
		},
		{
			"Sequential updates",
			`m.set("a", 1) => m.set("a", 2) => m.get("a")`,
			func() map[string]any { return map[string]any{"m": make(map[string]any)} },
			int64(2),
			false,
		},
		{
			"Mixed operations and variables",
			`count = 10 => m.set("val", count * 2) => m.get("val") + count`,
			func() map[string]any { return map[string]any{"m": make(map[string]any), "count": int64(0)} },
			int64(30),
			false,
		},
		{
			"Complex Sequence in If-Then",
			`if true then m.set("status", "ok") => m.set("code", 200) => m.get("code")`,
			func() map[string]any { return map[string]any{"m": make(map[string]any)} },
			int64(200),
			false,
		},
		{
			"Type Error: Method on Integer",
			`a.get("key")`,
			func() map[string]any { return map[string]any{"a": 123} },
			nil,
			true,
		},
		{
			"Type Error: Method on String",
			`"hello".get("key")`,
			nil,
			nil,
			true,
		},
		{
			"Empty string key",
			`m.set("", "empty") => m.get("")`,
			func() map[string]any { return map[string]any{"m": make(map[string]any)} },
			"empty",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getVars := func() map[string]any {
				if tt.vars == nil {
					return make(map[string]any)
				}
				return tt.vars()
			}

			// Test Evaluator
			varsEval := getVars()
			resEval, errEval := runEvaluator(tt.input, varsEval)
			checkResult(t, "Evaluator", resEval, errEval, tt.expected, tt.wantErr)

			// Test VM
			varsVM := getVars()
			resVM, errVM := runVM(tt.input, varsVM)
			checkResult(t, "VM", resVM, errVM, tt.expected, tt.wantErr)

			// Test NeoEx
			varsNeo := getVars()
			resNeo, errNeo := runNeoEx(tt.input, varsNeo)
			checkResult(t, "NeoEx", resNeo, errNeo, tt.expected, tt.wantErr)
		})
	}
}

func runEvaluator(input string, vars map[string]any) (any, error) {
	l := NewLexer(input)
	p := NewParser(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parser errors: %v", p.Errors())
	}
	ctx := NewMapContext(vars)
	return Eval(prog, ctx)
}

func runVM(input string, vars map[string]any) (any, error) {
	engine, err := NewEngineVM(input)
	if err != nil {
		return nil, err
	}
	return engine.Execute(vars)
}

func runNeoEx(input string, vars map[string]any) (any, error) {
	engine, err := NewEngineVMNeo(input)
	if err != nil {
		return nil, err
	}
	return engine.Execute(vars)
}

func checkResult(t *testing.T, mode string, got any, err error, expected any, wantErr bool) {
	if (err != nil) != wantErr {
		t.Errorf("[%s] error = %v, wantErr %v", mode, err, wantErr)
		return
	}
	if wantErr {
		return
	}
	if got != expected {
		t.Errorf("[%s] got %v (%T), want %v (%T)", mode, got, got, expected, expected)
	}
}
