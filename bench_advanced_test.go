package uwasa

import (
	"testing"
)

func benchmarkEngine(b *testing.B, input string, vars map[string]any, opt OptimizationLevel) {
	opts := EngineOptions{OptimizationLevel: opt}
	engine, err := NewEngineWithOptions(input, opts)
	if err != nil {
		b.Fatalf("NewEngine error: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(vars)
	}
}

func BenchmarkOptimizationComparison(b *testing.B) {
	scenarios := []struct {
		name  string
		input string
		vars  map[string]any
	}{
		{
			"IntAddSub_Const",
			"1 + 2 - 3 + 4 - 5 + 6 - 7 + 8 - 9 + 10",
			nil,
		},
		{
			"IntMixed_Const",
			"1 + 2 * 3 - 4 / 2 + 5 % 3",
			nil,
		},
		{
			"FloatAddSub_Const",
			"1.1 + 2.2 - 3.3 + 4.4 - 5.5 + 6.6",
			nil,
		},
		{
			"IntAddSub_Vars",
			"a + b - c + d - e + f",
			map[string]any{
				"a": int64(1), "b": int64(2), "c": int64(3),
				"d": int64(4), "e": int64(5), "f": int64(6),
			},
		},
		{
			"IntMixed_Vars",
			"a + b * c - d / e",
			map[string]any{
				"a": int64(1), "b": int64(2), "c": int64(3),
				"d": int64(4), "e": int64(2),
			},
		},
		{
			"BoolShortCircuit_Const",
			"false && (a + b * c / d > e)",
			map[string]any{
				"a": int64(1), "b": int64(2), "c": int64(3),
				"d": int64(4), "e": int64(5),
			},
		},
		{
			"AlgebraicSimplification",
			"a + 0 + b * 1 + c - c",
			map[string]any{
				"a": int64(10), "b": int64(20), "c": int64(30),
			},
		},
		{
			"DeeplyNested_Unoptimized",
			"((((a + 1) + 1) + 1) + 1) + 1",
			map[string]any{"a": int64(0)},
		},
		{
			"StringConcat_Long",
			`s1 + " " + s2 + " " + s3 + " " + s4`,
			map[string]any{
				"s1": "the", "s2": "quick", "s3": "brown", "s4": "fox",
			},
		},
	}

	configs := []struct {
		name string
		opts EngineOptions
	}{
		{"None", EngineOptions{OptimizationLevel: OptNone}},
		{"Basic", EngineOptions{OptimizationLevel: OptBasic}},
		{"Recompiled", EngineOptions{OptimizationLevel: OptBasic, UseRecompiler: true}},
	}

	for _, sc := range scenarios {
		for _, cfg := range configs {
			b.Run(sc.name+"/"+cfg.name, func(b *testing.B) {
				engine, err := NewEngineWithOptions(sc.input, cfg.opts)
				if err != nil {
					b.Fatalf("NewEngine error: %v", err)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = engine.Execute(sc.vars)
				}
			})
		}
		b.Run(sc.name+"/VM", func(b *testing.B) {
			engine, err := NewEngineVM(sc.input)
			if err != nil {
				b.Fatalf("NewEngineVM error: %v", err)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = engine.Execute(sc.vars)
			}
		})
		b.Run(sc.name+"/RegisterVM", func(b *testing.B) {
			engine, err := NewEngineVMWithOptions(sc.input, EngineOptions{UseRegisterVM: true, OptimizationLevel: OptBasic})
			if err != nil {
				b.Fatalf("NewEngineRegisterVM error: %v", err)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = engine.Execute(sc.vars)
			}
		})
	}
}

func BenchmarkMixedTypeArithmetic(b *testing.B) {
	input := "a + b"
	vars := map[string]any{"a": int64(10), "b": 2.5}

	b.Run("IntFloat", func(b *testing.B) {
		benchmarkEngine(b, input, vars, OptBasic)
	})
	b.Run("IntFloat_VM", func(b *testing.B) {
		engine, _ := NewEngineVM(input)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = engine.Execute(vars)
		}
	})
	b.Run("IntFloat_RegisterVM", func(b *testing.B) {
		engine, _ := NewEngineVMWithOptions(input, EngineOptions{UseRegisterVM: true})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = engine.Execute(vars)
		}
	})
}

func BenchmarkConcatBuiltin(b *testing.B) {
	input := `concat(s1, " ", s2, " ", s3, " ", s4)`
	vars := map[string]any{
		"s1": "the", "s2": "quick", "s3": "brown", "s4": "fox",
	}

	b.Run("Variables", func(b *testing.B) {
		benchmarkEngine(b, input, vars, OptBasic)
	})
	b.Run("Variables_VM", func(b *testing.B) {
		engine, _ := NewEngineVM(input)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = engine.Execute(vars)
		}
	})
	b.Run("Variables_RegisterVM", func(b *testing.B) {
		engine, _ := NewEngineVMWithOptions(input, EngineOptions{UseRegisterVM: true})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = engine.Execute(vars)
		}
	})
}

func BenchmarkStringConcatenation(b *testing.B) {
	input := `"hello " + "world" + name`
	vars := map[string]any{"name": "uwasa"}

	b.Run("ConstStrings", func(b *testing.B) {
		benchmarkEngine(b, input, vars, OptBasic)
	})
	b.Run("ConstStrings_VM", func(b *testing.B) {
		engine, _ := NewEngineVM(input)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = engine.Execute(vars)
		}
	})
	b.Run("ConstStrings_RegisterVM", func(b *testing.B) {
		engine, _ := NewEngineVMWithOptions(input, EngineOptions{UseRegisterVM: true})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = engine.Execute(vars)
		}
	})
}
