// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"testing"
)

func BenchmarkNeoEx(b *testing.B) {
	input := `if (score >= 90 || attendance > 0.9) && status == "active" then bonus = 100`
	vars := map[string]any{
		"score":      95.0,
		"attendance": 0.8,
		"status":     "active",
	}

	b.Run("NeoEx-Execute", func(b *testing.B) {
		engine, _ := NewEngineVMNeo(input)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			engine.Execute(vars)
		}
	})

	b.Run("VM-Execute", func(b *testing.B) {
		engine, _ := NewEngineVM(input)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			engine.Execute(vars)
		}
	})

	b.Run("NeoEx-Compile", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewEngineVMNeo(input)
		}
	})

	b.Run("VM-Compile", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewEngineVM(input)
		}
	})

	b.Run("NeoEx-ExecuteWithContext", func(b *testing.B) {
		engine, _ := NewEngineVMNeo(input)
		ctx := NewMapContext(vars)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			engine.ExecuteWithContext(ctx)
		}
	})

	b.Run("VM-ExecuteWithContext", func(b *testing.B) {
		engine, _ := NewEngineVM(input)
		ctx := NewMapContext(vars)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			engine.ExecuteWithContext(ctx)
		}
	})
}
