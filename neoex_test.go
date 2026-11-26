// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"testing"
)

func TestNeoExVM(t *testing.T) {
	tests := []struct {
		input    string
		vars     map[string]any
		expected any
	}{
		{`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`, map[string]any{"a": int64(1)}, "ok"},
		{"1 + 2 * 3", nil, int64(7)},
		{"(1 + 2) * 3", nil, int64(9)},
		{"a + b", map[string]any{"a": int64(10), "b": int64(20)}, int64(30)},
		{"if a > 10 is \"high\" else is \"low\"", map[string]any{"a": int64(15)}, "high"},
		{"if a > 10 is \"high\" else is \"low\"", map[string]any{"a": int64(5)}, "low"},
		{"if a > 10 then b = 100", map[string]any{"a": int64(15), "b": int64(0)}, int64(100)},
		{"concat(\"a\", \"b\", c)", map[string]any{"c": "d"}, "abd"},
		{"a == 10 && b == 20", map[string]any{"a": int64(10), "b": int64(20)}, true},
		{"a == 10 || b == 20", map[string]any{"a": int64(10), "b": int64(0)}, true},
		{"!true", nil, false},
		{"-10 + 20", nil, int64(10)},
		{"10 - a", map[string]any{"a": int64(3)}, int64(7)},
		{"10 / a", map[string]any{"a": int64(2)}, int64(5)},
		{"\"a\" + b", map[string]any{"b": "b"}, "ab"},
		{`a != 0`, map[string]any{"a": int64(1)}, true},
		{`a != 0`, map[string]any{"a": int64(0)}, false},
		{`"a" != "b"`, nil, true},
		{`1 != 1.0`, nil, false},
		{`if a != 0 is "yes" else is "no"`, map[string]any{"a": int64(1)}, "yes"},
	}

	for _, tt := range tests {
		engine, err := NewEngineVMNeo(tt.input)
		if err != nil {
			t.Errorf("NewEngineVMNeoEx(%q) error: %v", tt.input, err)
			continue
		}
		got, err := engine.Execute(tt.vars)
		if err != nil {
			t.Errorf("Execute(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("Execute(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestNeoExVM_ShortCircuit(t *testing.T) {
	input := "false && (a = 2)"
	engine, err := NewEngineVMNeo(input)
	if err != nil {
		t.Fatalf("NewEngineVMNeo failed: %v", err)
	}
	vars := map[string]any{"a": int64(0)}
	got, _ := engine.Execute(vars)
	if got != false {
		t.Errorf("Short-circuit && result failed: expected false, got %v", got)
	}
	if vars["a"] != int64(0) {
		t.Errorf("Short-circuit && side effect failed: expected 0, got %v", vars["a"])
	}

	input2 := "true || (a = 2)"
	engine2, _ := NewEngineVMNeo(input2)
	vars2 := map[string]any{"a": int64(0)}
	got2, _ := engine2.Execute(vars2)
	if got2 != true {
		t.Errorf("Short-circuit || result failed: expected true, got %v", got2)
	}
	if vars2["a"] != int64(0) {
		t.Errorf("Short-circuit || side effect failed: expected 0, got %v", vars2["a"])
	}
}

func TestNeoExVMStackOverflow(t *testing.T) {
	depth := 70
	expr := "a"
	for range depth {
		expr = "a + (" + expr + ")"
	}
	engine, err := NewEngineVMNeo(expr)
	if err != nil {
		t.Fatalf("NewEngineVMNeo failed: %v", err)
	}
	_, err = engine.Execute(map[string]any{"a": 1})
	if err == nil || err.Error() != "NeoVM stack overflow" {
		t.Errorf("Expected stack overflow error, got: %v", err)
	}
}

func TestNeoExVM_StringFusion(t *testing.T) {
	input := `"a" + "b" + c + d + "e"`
	c := NewNeoCompiler(input)
	bc, err := c.Compile()
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// Should be: Push("ab"), GetG(c), GetG(d), Push("e"), Concat(4)
	// "a"+"b" folded to "ab"
	// Then "ab" + c -> Concat("ab", c)
	// Then Concat("ab", c) + d -> Concat("ab", c, d)
	// Then Concat("ab", c, d) + "e" -> Concat("ab", c, d, "e")

	foundConcat := false
	var nArgs int32
	for _, inst := range bc.Instructions {
		if inst.Op == NeoOpConcat {
			foundConcat = true
			nArgs = inst.Arg
		}
	}
	if !foundConcat {
		t.Errorf("String fusion failed: NeoOpConcat not found")
	}
	if nArgs != 4 {
		t.Errorf("String fusion failed: expected Concat with 4 args, got %d", nArgs)
	}
}

func TestNeoExVM_ConstantFold(t *testing.T) {
	input := `"foo" + "bar" + "baz"`
	c := NewNeoCompiler(input)
	bc, err := c.Compile()
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	if len(bc.Instructions) != 2 { // Push, Return
		t.Errorf("Expected 2 instructions, got %d", len(bc.Instructions))
	}
	res := bc.Constants[bc.Instructions[0].Arg].Str
	if res != "foobarbaz" {
		t.Errorf("Expected foobarbaz, got %s", res)
	}
}

type neoTestContext struct {
	vars map[string]any
}

func (c *neoTestContext) Get(name string) (any, bool) {
	v, ok := c.vars[name]
	return v, ok
}

func (c *neoTestContext) Set(name string, value any) error {
	c.vars[name] = value
	return nil
}

// The general (custom Context) VM path must implement every opcode the
// compiler can emit, including the fused concat variants implemented only in
// RunNeoVMWithMap before this regression test was added.
func TestNeoExVM_ConcatFusionWithContext(t *testing.T) {
	tests := []struct {
		input    string
		op       NeoOpCode
		vars     map[string]any
		expected string
	}{
		{`concat(a, b)`, NeoOpConcat2, map[string]any{"a": "A", "b": "B"}, "AB"},
		{`concat(b, "x")`, NeoOpConcatGC, map[string]any{"b": "B"}, "Bx"},
		{`concat("x", b)`, NeoOpConcatCG, map[string]any{"b": "B"}, "xB"},
	}

	for _, tt := range tests {
		engine, err := NewEngineVMNeo(tt.input)
		if err != nil {
			t.Fatalf("NewEngineVMNeo(%q) error: %v", tt.input, err)
		}

		found := false
		for _, inst := range engine.neoBytecode.Instructions {
			if inst.Op == tt.op {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q: compiler did not emit %v", tt.input, tt.op)
			continue
		}

		got, err := engine.ExecuteWithContext(&neoTestContext{vars: tt.vars})
		if err != nil {
			t.Errorf("ExecuteWithContext(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("ExecuteWithContext(%q) = %v, want %v", tt.input, got, tt.expected)
		}

		gotMap, err := engine.Execute(tt.vars)
		if err != nil || gotMap != tt.expected {
			t.Errorf("Execute(%q) = %v, err=%v, want %v", tt.input, gotMap, err, tt.expected)
		}
	}
}
