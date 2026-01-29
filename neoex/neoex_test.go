package neoex

import (
	"testing"
	"github.com/kamihama-railway/uwasa/types"
)

func TestNeoEx(t *testing.T) {
	tests := []struct {
		input    string
		vars     map[string]any
		expected any
	}{
		{"1 + 2 * 3", nil, int64(7)},
		{"(1 + 2) * 3", nil, int64(9)},
		{"a + b", map[string]any{"a": int64(10), "b": int64(20)}, int64(30)},
		{"if a > 10 is \"big\" else is \"small\"", map[string]any{"a": int64(15)}, "big"},
		{"a = 10", map[string]any{"a": int64(0)}, int64(10)},
		{"concat(\"hello\", \" \", name)", map[string]any{"name": "world"}, "hello world"},
		{"a == 10 && b == 20", map[string]any{"a": int64(10), "b": int64(20)}, true},
		{"a == 10 || b == 20", map[string]any{"a": int64(5), "b": int64(20)}, true},
		{"!a", map[string]any{"a": false}, true},
		{"-a + 5", map[string]any{"a": int64(10)}, int64(-5)},
	}

	for _, tt := range tests {
		c := NewCompiler(tt.input)
		bc, err := c.Compile()
		if err != nil {
			t.Errorf("input %s: compile error: %v", tt.input, err)
			continue
		}

		ctx := types.NewMapContext(tt.vars)
		got, err := Run(bc, ctx)
		if err != nil {
			t.Errorf("input %s: execute error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.input, tt.expected, got)
		}
	}
}
