package uwasa

import (
	"testing"
)

func TestConstantFolding(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2", "3"},
		{"10 - 5", "5"},
		{"2 * 3", "6"},
		{"10 / 2", "5"},
		{"10 % 3", "1"},
		{"1 + 2 * 3", "7"},
		{"(1 + 2) * 3", "9"},
		{"true && false", "false"},
		{"true || false", "true"},
		{"1 == 1", "true"},
		{"1 > 2", "false"},
		{"if true is 1 else is 2", "1"},
		{"if false is 1 else is 2", "2"},
		{"if 1 == 1 is " + `"yes"` + " else is " + `"no"`, "yes"},
	}

	for _, tt := range tests {
		t.Logf("Testing input: %s", tt.input)
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()
		folded := Fold(program)

		if folded == nil {
			t.Errorf("input %q: folded is nil", tt.input)
			continue
		}

		if folded.String() != tt.expected {
			t.Errorf("input %q: expected folded string %q, got %q", tt.input, tt.expected, folded.String())
		}

		// Ensure it's folded to a literal if possible
		switch tt.expected {
		case "true", "false":
			if _, ok := folded.(*BooleanLiteral); !ok {
				t.Errorf("input %q: expected *BooleanLiteral, got %T", tt.input, folded)
			}
		case "yes", "no", "1", "2", "3", "5", "6", "7", "9":
			// numbers or strings
			if _, ok := folded.(*NumberLiteral); !ok {
				if _, ok := folded.(*StringLiteral); !ok {
					t.Errorf("input %q: expected literal, got %T", tt.input, folded)
				}
			}
		}
	}
}
