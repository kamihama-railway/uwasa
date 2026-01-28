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
