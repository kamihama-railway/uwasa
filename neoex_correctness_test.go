// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"math"
	"testing"
)

func TestNeoExVM_Correctness(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]any
		expected any
	}{
		{"Int Add", "a + b", map[string]any{"a": int64(10), "b": int64(20)}, int64(30)},
		{"Int Sub", "a - b", map[string]any{"a": int64(50), "b": int64(20)}, int64(30)},
		{"Int Mul", "a * b", map[string]any{"a": int64(10), "b": int64(20)}, int64(200)},
		{"Int Div", "a / b", map[string]any{"a": int64(100), "b": int64(2)}, int64(50)},
		{"Float Add", "a + b", map[string]any{"a": 10.5, "b": 20.5}, 31.0},
		{"Mixed Add", "a + b", map[string]any{"a": int64(10), "b": 20.5}, 30.5},
		{"String Concat", "a + b", map[string]any{"a": "hello ", "b": "world"}, "hello world"},
		{"Greater", "a > b", map[string]any{"a": int64(20), "b": int64(10)}, true},
		{"Less", "a < b", map[string]any{"a": int64(5), "b": int64(10)}, true},
		{"Equal", "a == b", map[string]any{"a": "test", "b": "test"}, true},
		{"Logical And", "a > 0 && b > 0", map[string]any{"a": 1, "b": 1}, true},
		{"Logical Or", "a > 10 || b > 0", map[string]any{"a": 1, "b": 1}, true},
		{"Not", "!a", map[string]any{"a": false}, true},
		{"Fused EQGC", "a == 10", map[string]any{"a": int64(10)}, true},
		{"Fused ADDGC", "a + 10", map[string]any{"a": int64(10)}, int64(20)},
		{"Fused ADDGG", "a + b", map[string]any{"a": int64(10), "b": int64(20)}, int64(30)},
		{"Complex", "(a + b) * (c - d) / e", map[string]any{"a": 10, "b": 20, "c": 30, "d": 10, "e": 2}, int64(300)},
		{"Const Global Sub", "100 - a", map[string]any{"a": int64(30)}, int64(70)},
		{"Const Global Div", "100 / a", map[string]any{"a": int64(2)}, int64(50)},
		{"Infinity Div (Var)", "1 / a", map[string]any{"a": 0}, math.Inf(1)},
		{"Map Set and Get", "m.set(\"a\", 1) => m.get(\"a\")", map[string]any{"m": make(map[string]any)}, int64(1)},
		{"Map Has", "m.set(\"a\", 1) => m.has(\"a\")", map[string]any{"m": make(map[string]any)}, true},
		{"Map Del", "m.set(\"a\", 1) => m.del(\"a\") => m.has(\"a\")", map[string]any{"m": make(map[string]any)}, false},
		{"Sequence Side Effect", "a = 1 => a = a + 1 => a", map[string]any{"a": int64(0)}, int64(2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngineVMNeo(tt.input)
			if err != nil {
				t.Fatalf("NewEngineVMNeo failed: %v", err)
			}
			got, err := engine.Execute(tt.vars)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}
			if got != tt.expected {
				t.Errorf("Execute() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNeoExVM_StackOverflow(t *testing.T) {
	// Stack is 64
	input := "a"
	for range 70 {
		input = "a + (" + input + ")"
	}
	engine, err := NewEngineVMNeo(input)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	_, err = engine.Execute(map[string]any{"a": int64(1)})
	if err == nil {
		t.Error("Expected stack overflow error, got nil")
	}
}
