// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"testing"
)

func TestUwasaEngine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]any
		expected any
	}{
		{
			"Basic arithmetic",
			"1 + 2 * 3",
			nil,
			int64(7),
		},
		{
			"Parentheses",
			"(1 + 2) * 3",
			nil,
			int64(9),
		},
		{
			"Variables",
			"a + b",
			map[string]any{"a": 10, "b": 20},
			int64(30),
		},
		{
			"If...is...else",
			"if a == 0 is \"yes\" else is \"no\"",
			map[string]any{"a": 0},
			"yes",
		},
		{
			"Nested If",
			`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`,
			map[string]any{"a": 1},
			"ok",
		},
		{
			"Deep Nested If",
			`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`,
			map[string]any{"a": 2},
			"bad",
		},
		{
			"Pre-condition true",
			`if a == 0 then b + 10`,
			map[string]any{"a": 0, "b": 5},
			int64(15),
		},
		{
			"Pre-condition false",
			`if a == 0 then b + 10`,
			map[string]any{"a": 1, "b": 5},
			nil,
		},
		{
			"Assignment",
			`b = b + 10`,
			map[string]any{"b": 5},
			int64(15),
		},
		{
			"Mixed types comparison",
			`if a == 0.0`,
			map[string]any{"a": 0},
			true,
		},
		{
			"String concatenation",
			`"hello " + "world"`,
			nil,
			"hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(tt.input)
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}
			result, err := engine.Execute(tt.vars)
			if err != nil {
				t.Fatalf("failed to execute: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestAssignmentSideEffect(t *testing.T) {
	engine, _ := NewEngine(`b = b + 10`)
	vars := map[string]any{"b": 5}
	engine.Execute(vars)
	if vars["b"] != int64(15) {
		t.Errorf("expected vars[b] to be 15, got %v", vars["b"])
	}
}

func TestEngineConcurrency(t *testing.T) {
	input := `if a == 0 is "yes" else is "no"`
	engine, _ := NewEngine(input)

	const workers = 100
	const iterations = 1000
	done := make(chan bool)

	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				vars := map[string]any{"a": j % 2}
				result, err := engine.Execute(vars)
				if err != nil {
					t.Errorf("concurrent execute failed: %v", err)
					return
				}
				expected := "yes"
				if j%2 != 0 {
					expected = "no"
				}
				if result != expected {
					t.Errorf("concurrent result mismatch: expected %v, got %v", expected, result)
					return
				}
			}
			done <- true
		}()
	}

	for i := 0; i < workers; i++ {
		<-done
	}
}
