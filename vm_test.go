// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"testing"
)

func TestVMExecution(t *testing.T) {
	tests := []struct {
		input    string
		expected any
		vars     map[string]any
	}{
		{"1 + 2 * 3", int64(7), nil},
		{"(1 + 2) * 3", int64(9), nil},
		{"a + b", int64(15), map[string]any{"a": int64(5), "b": int64(10)}},
		{"if a > 10 is 1 else is 0", int64(1), map[string]any{"a": int64(15)}},
		{"if a > 10 is 1 else is 0", int64(0), map[string]any{"a": int64(5)}},
		{"a = 100", int64(100), map[string]any{"a": int64(0)}},
		{"true && false", false, nil},
		{"true || false", true, nil},
		{"concat(\"hello \", name)", "hello world", map[string]any{"name": "world"}},
		{"if true then 1", int64(1), nil},
		{"if false then 1", nil, nil},
		{"if a == 10 then b = 20 else is b = 30", int64(20), map[string]any{"a": int64(10), "b": int64(0)}},
		{"a && b", true, map[string]any{"a": true, "b": true}},
		{"a && b", false, map[string]any{"a": true, "b": false}},
		{"a || b", true, map[string]any{"a": false, "b": true}},
		{"a || b", false, map[string]any{"a": false, "b": false}},
		{`if (a + b) * (c - d) > 100 && e == "test" then f = 1`, int64(1), map[string]any{
			"a": int64(50), "b": int64(60), "c": int64(10), "d": int64(5), "e": "test",
		}},
	}

	for _, tt := range tests {
		engine, err := NewEngineWithOptions(tt.input, EngineOptions{UseVM: true, OptimizationLevel: OptBasic})
		if err != nil {
			t.Errorf("input %s: NewEngineWithOptions failed: %v", tt.input, err)
			continue
		}

		got, err := engine.Execute(tt.vars)
		if err != nil {
			t.Errorf("input %s: Execute failed: %v", tt.input, err)
			continue
		}

		if got != tt.expected {
			t.Errorf("input %s: expected %v, got %v", tt.input, tt.expected, got)
		}

		if tt.input == "a = 100" {
			// check if variable was set in context
			// engine.Execute creates a new context, so we might need ExecuteWithContext to verify side effects
			ctx := NewMapContext(tt.vars)
			engine.ExecuteWithContext(ctx)
			val, _ := ctx.Get("a")
			if val != int64(100) {
				t.Errorf("assignment failed: expected 100, got %v", val)
			}
		}
	}
}
