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
			"Simple condition true",
			`if a == 0 && b >= 1`,
			map[string]any{"a": 0, "b": 1},
			true,
		},
		{
			"Simple condition false",
			`if a == 0 && b >= 1`,
			map[string]any{"a": 0, "b": 0},
			false,
		},
		{
			"Multi-layer branch 1",
			`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`,
			map[string]any{"a": 0},
			"yes",
		},
		{
			"Multi-layer branch 2",
			`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`,
			map[string]any{"a": 1},
			"ok",
		},
		{
			"Multi-layer branch 3",
			`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`,
			map[string]any{"a": 2},
			"bad",
		},
		{
			"Pre-condition true",
			`if a == 0 then b + 10`,
			map[string]any{"a": 0, "b": 5},
			15.0,
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
			15.0,
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
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAssignmentSideEffect(t *testing.T) {
	engine, _ := NewEngine(`b = b + 10`)
	vars := map[string]any{"b": 5}
	engine.Execute(vars)
	if vars["b"] != 15.0 {
		t.Errorf("expected vars[b] to be 15.0, got %v", vars["b"])
	}
}

func TestEngineConcurrency(t *testing.T) {
	input := `if a == 0 is "yes" else is "no"`
	engine, _ := NewEngine(input)

	const workers = 100
	const iterations = 1000
	done := make(chan bool)

	for i := 0; i < workers; i++ {
		go func(id int) {
			for j := 0; j < iterations; j++ {
				val := j % 2
				vars := map[string]any{"a": val}
				expected := "no"
				if val == 0 {
					expected = "yes"
				}

				result, err := engine.Execute(vars)
				if err != nil {
					t.Errorf("worker %d: execute error: %v", id, err)
					return
				}
				if result != expected {
					t.Errorf("worker %d: expected %v, got %v", id, expected, result)
					return
				}
			}
			done <- true
		}(i)
	}

	for i := 0; i < workers; i++ {
		<-done
	}
}
