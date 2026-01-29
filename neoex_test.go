// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"testing"
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
		{"if a > 10 is \"high\" else is \"low\"", map[string]any{"a": int64(15)}, "high"},
		{"if a > 10 is \"high\" else is \"low\"", map[string]any{"a": int64(5)}, "low"},
		{"if a > 10 then b = 100", map[string]any{"a": int64(15), "b": int64(0)}, int64(100)},
		{"concat(\"a\", \"b\", c)", map[string]any{"c": "d"}, "abd"},
		{"a == 10 && b == 20", map[string]any{"a": int64(10), "b": int64(20)}, true},
		{"a == 10 || b == 20", map[string]any{"a": int64(10), "b": int64(0)}, true},
		{"!true", nil, false},
		{"-10 + 20", nil, int64(10)},
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
