// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package rvm

import (
	"testing"
	"github.com/kamihama-railway/uwasa/lexer"
	"github.com/kamihama-railway/uwasa/parser"
)

func TestRegisterVM(t *testing.T) {
	tests := []struct {
		input    string
		vars     map[string]any
		expected any
	}{
		{"1 + 2 * 3", nil, int64(7)},
		{"(1 + 2) * 3", nil, int64(9)},
		{"a + b", map[string]any{"a": int64(10), "b": int64(20)}, int64(30)},
		{"a - b", map[string]any{"a": int64(20), "b": int64(5)}, int64(15)},
		{"a * b", map[string]any{"a": int64(4), "b": int64(5)}, int64(20)},
		{"a / b", map[string]any{"a": int64(20), "b": int64(4)}, int64(5)},
		{"a % b", map[string]any{"a": int64(7), "b": int64(3)}, int64(1)},
		{"a == 10", map[string]any{"a": int64(10)}, true},
		{"a > 10", map[string]any{"a": int64(15)}, true},
		{"a < 10", map[string]any{"a": int64(5)}, true},
		{"a >= 10", map[string]any{"a": int64(10)}, true},
		{"a <= 10", map[string]any{"a": int64(10)}, true},
		{"if a > 10 is \"big\" else is \"small\"", map[string]any{"a": int64(15)}, "big"},
		{"if a > 10 is \"big\" else is \"small\"", map[string]any{"a": int64(5)}, "small"},
		{"a = 10", map[string]any{"a": int64(0)}, int64(10)},
		{"concat(\"hello\", \" \", name)", map[string]any{"name": "world"}, "hello world"},
		{"a > 10 && b < 5", map[string]any{"a": int64(15), "b": int64(3)}, true},
		{"a > 10 && b < 5", map[string]any{"a": int64(5), "b": int64(3)}, false},
		{"a > 10 || b < 5", map[string]any{"a": int64(5), "b": int64(3)}, true},
		{"a > 10 || b < 5", map[string]any{"a": int64(5), "b": int64(10)}, false},
		{"!a", map[string]any{"a": true}, false},
		{"!a", map[string]any{"a": false}, true},
		{"-a", map[string]any{"a": int64(5)}, int64(-5)},
	}

	for _, tt := range tests {
		l := lexer.NewLexer(tt.input)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Errorf("input %s: parser error: %v", tt.input, p.Errors())
			continue
		}

		c := NewRegisterCompiler()
		bc, err := c.Compile(prog)
		if err != nil {
			t.Errorf("input %s: compile error: %v", tt.input, err)
			continue
		}

		// Simple MapContext implementation for testing
		ctx := &testMapContext{vars: tt.vars}
		if ctx.vars == nil {
			ctx.vars = make(map[string]any)
		}

		got, err := RunRegisterVM(bc, ctx)
		if err != nil {
			t.Errorf("input %s: Execute error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("%s: expected %v (%T), got %v (%T)", tt.input, tt.expected, tt.expected, got, got)
		}
	}
}

type testMapContext struct {
	vars map[string]any
}

func (m *testMapContext) Get(name string) (any, bool) {
	v, ok := m.vars[name]
	return v, ok
}

func (m *testMapContext) Set(name string, value any) error {
	m.vars[name] = value
	return nil
}

func TestRegisterVM_ShortCircuit(t *testing.T) {
	// Test if side effects are skipped
	input := "false && (a = 2)"
	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	prog := p.ParseProgram()
	c := NewRegisterCompiler()
	bc, _ := c.Compile(prog)

	vars := map[string]any{"a": int64(0)}
	ctx := &testMapContext{vars: vars}
	got, _ := RunRegisterVM(bc, ctx)
	if got != false {
		t.Errorf("Short-circuit && result failed: expected false, got %v", got)
	}
	if vars["a"] != int64(0) {
		t.Errorf("Short-circuit && side effect failed: expected 0, got %v", vars["a"])
	}

	input2 := "true || (a = 2)"
	l2 := lexer.NewLexer(input2)
	p2 := parser.NewParser(l2)
	prog2 := p2.ParseProgram()
	c2 := NewRegisterCompiler()
	bc2, _ := c2.Compile(prog2)
	vars2 := map[string]any{"a": int64(0)}
	ctx2 := &testMapContext{vars: vars2}
	got2, _ := RunRegisterVM(bc2, ctx2)
	if got2 != true {
		t.Errorf("Short-circuit || result failed: expected true, got %v", got2)
	}
	if vars2["a"] != int64(0) {
		t.Errorf("Short-circuit || side effect failed: expected 0, got %v", vars2["a"])
	}
}
