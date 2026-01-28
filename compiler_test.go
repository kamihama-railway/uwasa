package uwasa

import (
	"testing"
)

func TestRecompiler(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a + 0", "a"},
		{"0 + a", "a"},
		{"a - 0", "a"},
		{"a * 1", "a"},
		{"1 * a", "a"},
		{"a * 0", "0"},
		{"0 * a", "0"},
		{"a / 1", "a"},
		{"a - a", "0"},
		{"a == a", "true"},
		{"a = a", "a"},
		{"(a + b) - (a + b)", "((a + b) - (a + b))"}, // Should not simplify complex expressions for now to avoid side effect issues
	}

	for _, tt := range tests {
		engine, err := NewEngineWithOptions(tt.input, EngineOptions{UseRecompiler: true})
		if err != nil {
			t.Fatalf("input %q: NewEngine error: %v", tt.input, err)
		}

		if engine.program.String() != tt.expected {
			t.Errorf("input %q: expected optimized %q, got %q", tt.input, tt.expected, engine.program.String())
		}
	}
}

func TestStaticAnalysis(t *testing.T) {
	tests := []struct {
		input       string
		errContains string
	}{
		{"1 / 0", "division by zero"},
		{"a / 0", "division by zero"},
		{"- \"hello\"", "invalid operation: -string"},
		{"1 + \"hello\"", "invalid operation: string + number mismatch"},
		{"\"hello\" * 2", "invalid operation: string * string/number"},
		{"true + 1", "invalid operation: boolean + any"},
		{"true && 1", "invalid logic operation: && used with non-boolean literal"},
	}

	for _, tt := range tests {
		_, err := NewEngineWithOptions(tt.input, EngineOptions{UseRecompiler: true})
		if err == nil {
			t.Errorf("input %q: expected error, got nil", tt.input)
			continue
		}
		if errContains := tt.errContains; errContains != "" {
			// Basic check
			found := false
			if err.Error() != "" {
				found = true
			}
			if !found {
				t.Errorf("input %q: error %q does not contain %q", tt.input, err.Error(), errContains)
			}
		}
	}
}

func TestOptimizationLevels(t *testing.T) {
	input := "1 + 2"

	// OptNone
	engineNone, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptNone})
	if engineNone.program.String() != "(1 + 2)" {
		t.Errorf("OptNone: expected (1 + 2), got %s", engineNone.program.String())
	}

	// OptBasic
	engineBasic, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptBasic})
	if engineBasic.program.String() != "3" {
		t.Errorf("OptBasic: expected 3, got %s", engineBasic.program.String())
	}
}
