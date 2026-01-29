// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"testing"
)

func BenchmarkNeoExVM_Execute(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngineVMNeo(input)
	vars := map[string]any{"a": 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}

func BenchmarkNeoExVM_MixedTypeArithmetic(b *testing.B) {
	input := "a + b"
	vars := map[string]any{"a": int64(10), "b": 2.5}
	engine, _ := NewEngineVMNeo(input)
	ctx := NewMapContext(vars)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.ExecuteWithContext(ctx)
	}
}

func BenchmarkNeoExVM_StringConcatenation(b *testing.B) {
	input := `"hello " + "world " + name`
	vars := map[string]any{"name": "uwasa"}
	engine, _ := NewEngineVMNeo(input)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(vars)
	}
}
