package uwasa

import (
	"math"
	"testing"
)

func TestEnrichedCoverage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]any
		expected any
	}{
		{
			"Nested If in Condition (True)",
			"if (if a > 0 then true) is \"ok\" else is \"no\"",
			map[string]any{"a": 1},
			"ok",
		},
		{
			"Nested If in Condition (False/Nil)",
			"if (if a > 0 then true) is \"ok\" else is \"no\"",
			map[string]any{"a": -1},
			"no",
		},
		{
			"Deeply Nested If-Else Chain",
			"if a == 1 is 10 else if a == 2 is 20 else if a == 3 is 30 else is 40",
			map[string]any{"a": 3},
			int64(30),
		},
		{
			"Nested Function Calls",
			"concat(\"A\", concat(\"B\", concat(\"C\", \"D\")))",
			nil,
			"ABCD",
		},
		{
			"Complex Precedence and Logic",
			"(a + 5) * 2 > 20 && (s = \"passed\") == \"passed\"",
			map[string]any{"a": int64(6), "s": ""},
			true,
		},
		{
			"Conditional Assignment in Branch",
			"if score >= 60 is (status = \"pass\") else is (status = \"fail\")",
			map[string]any{"score": int64(70), "status": ""},
			"pass",
		},
		{
			"Mixed Type Comparisons (Float/Int)",
			"if a > 10.5 is \"float\" else is \"int\"",
			map[string]any{"a": int64(11)},
			"float",
		},
		{
			"Nil handling with If-Simple",
			"if (if a > 100) is \"never\" else is \"safe\"",
			map[string]any{"a": 10},
			"safe",
		},
		{
			"Short Circuit in Nested Expression",
			"if (a == 0) || (b = 100) > 50 is \"done\" else is \"error\"",
			map[string]any{"a": 0, "b": 1},
			"done",
		},
	}

	runTests := func(t *testing.T, engineType string) {
		for _, tt := range tests {
			t.Run(engineType+"_"+tt.name, func(t *testing.T) {
				var engine *Engine
				var err error
				if engineType == "Standard" {
					engine, err = NewEngineVM(tt.input)
				} else {
					engine, err = NewEngineVMNeo(tt.input)
				}

				if err != nil {
					t.Fatalf("Compile failed: %v", err)
				}

				// Re-copy vars to avoid side-effect pollution between Standard and NeoEx if they share map
				testVars := make(map[string]any)
				for k, v := range tt.vars {
					testVars[k] = v
				}

				got, err := engine.Execute(testVars)
				if err != nil {
					t.Fatalf("Execute failed: %v", err)
				}
				if got != tt.expected {
					t.Errorf("got %v (%T), want %v (%T)", got, got, tt.expected, tt.expected)
				}
			})
		}
	}

	runTests(t, "Standard")
	runTests(t, "NeoEx")
}

func TestEdgeCases(t *testing.T) {
	edgeTests := []struct {
		name     string
		input    string
		vars     map[string]any
		expected any
	}{
		{"Divide by Zero (Float)", "1.0 / 0.0", nil, math.Inf(1)},
		{"Empty Concat", "concat()", nil, ""},
		{"Assign to non-existent", "new_var = 10", map[string]any{}, int64(10)},
		{"Large Number Ops", "9223372036854775807 - 1", nil, int64(9223372036854775806)},
		{"Nested Bangs", "!!!!true", nil, true},
		{"Modulo with Variables", "a % b", map[string]any{"a": 10, "b": 3}, int64(1)},
	}

	for _, tt := range edgeTests {
		t.Run("NeoEx_Edge_"+tt.name, func(t *testing.T) {
			engine, _ := NewEngineVMNeo(tt.input)
			got, _ := engine.Execute(tt.vars)
			if got != tt.expected {
				// Handle 0.0/0.0 special cases if needed, but for now strict equal
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}
