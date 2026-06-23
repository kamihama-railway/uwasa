// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"testing"
)

func TestVM(t *testing.T) {
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
		engine, err := NewEngineVM(tt.input)
		if err != nil {
			t.Errorf("input %s: NewEngine error: %v", tt.input, err)
			continue
		}
		got, err := engine.Execute(tt.vars)
		if err != nil {
			t.Errorf("input %s: Execute error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("%s: expected %v (%T), got %v (%T)", tt.input, tt.expected, tt.expected, got, got)
		}
	}
}

func TestVM_ShortCircuit(t *testing.T) {
	// Test if side effects are skipped using expressions
	// Note: uwasa doesn't support multiple statements separated by semicolon in a single ParseProgram call

	// If false, then (a=2) should not execute. Result should be false.
	input := "false && (a = 2)"
	engine, _ := NewEngineVM(input)
	vars := map[string]any{"a": int64(0)}
	got, _ := engine.Execute(vars)
	if got != false {
		t.Errorf("Short-circuit && result failed: expected false, got %v", got)
	}
	if vars["a"] != int64(0) {
		t.Errorf("Short-circuit && side effect failed: expected 0, got %v", vars["a"])
	}

	// If true, then (a=2) should not execute. Result should be true.
	input2 := "true || (a = 2)"
	engine2, _ := NewEngineVM(input2)
	vars2 := map[string]any{"a": int64(0)}
	got2, _ := engine2.Execute(vars2)
	if got2 != true {
		t.Errorf("Short-circuit || result failed: expected true, got %v", got2)
	}
	if vars2["a"] != int64(0) {
		t.Errorf("Short-circuit || side effect failed: expected 0, got %v", vars2["a"])
	}
}

func TestVMStackOverflow(t *testing.T) {
	// 1. Test basic stack overflow (many nested additions with identifier)
	depth := 70
	expr := "a"
	for range depth {
		expr = "a + (" + expr + ")"
	}
	engine, err := NewEngineVMWithOptions(expr, EngineOptions{OptimizationLevel: OptNone})
	if err != nil {
		t.Fatalf("NewEngineVM failed: %v", err)
	}
	_, err = engine.Execute(map[string]any{"a": 1})
	if err == nil || err.Error() != "VM stack overflow" {
		t.Errorf("Expected stack overflow error, got: %v", err)
	}

	// 2. Test stack overflow with OpAddGlobal (optimized path)
	expr2 := "(a+1)"
	for range depth {
		expr2 = "(a+1) + (" + expr2 + ")"
	}
	engine2, err := NewEngineVMWithOptions(expr2, EngineOptions{OptimizationLevel: OptBasic})
	if err != nil {
		t.Fatalf("NewEngineVM failed: %v", err)
	}
	_, err = engine2.Execute(map[string]any{"a": 1})
	if err == nil || err.Error() != "VM stack overflow" {
		t.Errorf("Expected stack overflow error, got: %v", err)
	}
}

func TestVM_FusedStringConcat(t *testing.T) {
	// Test if OpAddGlobal correctly handles strings instead of falling back to float 0
	input := "name + \"!\""
	engine, err := NewEngineVM(input)
	if err != nil {
		t.Fatalf("NewEngineVM failed: %v", err)
	}

	vars := map[string]any{"name": "uwasa"}
	got, err := engine.Execute(vars)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expected := "uwasa!"
	if got != expected {
		t.Errorf("Expected %q, got %q (this likely indicates OpAddGlobal failed to handle strings)", expected, got)
	}

	// Test OpAddGlobalGlobal
	input2 := "a + b"
	engine2, _ := NewEngineVM(input2)
	vars2 := map[string]any{"a": "hello ", "b": "world"}
	got2, _ := engine2.Execute(vars2)
	if got2 != "hello world" {
		t.Errorf("Expected %q, got %q (OpAddGlobalGlobal failed for strings)", "hello world", got2)
	}
}

func TestVMNewSyntax(t *testing.T) {
	tests := []struct {
		input    string
		vars     map[string]any
		expected any
	}{
		{`m.set("a", 1) => m.get("a")`, map[string]any{"m": make(map[string]any)}, int64(1)},
		{`m.set("a", 1) => m.has("a")`, map[string]any{"m": make(map[string]any)}, true},
		{`m.set("a", 1) => m.del("a") => m.has("a")`, map[string]any{"m": make(map[string]any)}, false},
		{`if m.has("a") then m.get("a") else is "none"`, map[string]any{"m": map[string]any{"a": "ok"}}, "ok"},
		{`a = 1 => a = a + 1 => a`, map[string]any{"a": int64(0)}, int64(2)},
		{`m.set("x", 10) => m.set("y", 20) => m.get("x") + m.get("y")`, map[string]any{"m": make(map[string]any)}, int64(30)},
	}

	for i, tt := range tests {
		engine, err := NewEngineVM(tt.input)
		if err != nil {
			t.Errorf("test[%d] %q NewEngineVM error: %v", i, tt.input, err)
			continue
		}
		got, err := engine.Execute(tt.vars)
		if err != nil {
			t.Errorf("test[%d] %q Execute error: %v", i, tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("test[%d] %q expected %v (%T), got %v (%T)", i, tt.input, tt.expected, tt.expected, got, got)
		}
	}
}
