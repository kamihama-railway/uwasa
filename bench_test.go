package uwasa

import (
	"testing"
)

func BenchmarkLexer(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	for i := 0; i < b.N; i++ {
		l := NewLexer(input)
		for tok := l.NextToken(); tok.Type != TokenEOF; tok = l.NextToken() {
		}
		lexerPool.Put(l)
	}
}

func BenchmarkParser(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	for i := 0; i < b.N; i++ {
		l := NewLexer(input)
		p := NewParser(l)
		p.ParseProgram()
		parserPool.Put(p)
		lexerPool.Put(l)
	}
}

func BenchmarkEngineExecute(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngine(input)
	vars := map[string]any{"a": 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}

func BenchmarkEngineExecute_VM(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngineVM(input)
	vars := map[string]any{"a": 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}

func BenchmarkEngineExecute_OptNone(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptNone})
	vars := map[string]any{"a": 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}

func BenchmarkEngineExecute_OptBasic(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptBasic})
	vars := map[string]any{"a": 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}

func BenchmarkEngineExecute_Recompiled(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptBasic, UseRecompiler: true})
	vars := map[string]any{"a": 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}

type benchContext struct {
	vars map[string]any
}

func (c *benchContext) Get(name string) (any, bool) {
	val, exists := c.vars[name]
	return val, exists
}
func (c *benchContext) Set(name string, value any) error {
	c.vars[name] = value
	return nil
}

func BenchmarkEngineExecuteWithContext(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngine(input)
	ctx := &benchContext{vars: map[string]any{"a": 1}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.ExecuteWithContext(ctx)
	}
}

func BenchmarkEngineExecuteWithContext_VM(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngineVM(input)
	ctx := &benchContext{vars: map[string]any{"a": 1}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.ExecuteWithContext(ctx)
	}
}

func BenchmarkEngineExecuteWithContext_OptNone(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptNone})
	ctx := &benchContext{vars: map[string]any{"a": 1}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.ExecuteWithContext(ctx)
	}
}

func BenchmarkEngineExecuteWithContext_OptBasic(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptBasic})
	ctx := &benchContext{vars: map[string]any{"a": 1}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.ExecuteWithContext(ctx)
	}
}

func BenchmarkEngineExecuteWithContext_Recompiled(b *testing.B) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`
	engine, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptBasic, UseRecompiler: true})
	ctx := &benchContext{vars: map[string]any{"a": 1}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.ExecuteWithContext(ctx)
	}
}

func BenchmarkComplexExpression(b *testing.B) {
	input := `if (a + b) * (c - d) > 100 && e == "test" then f = 1`
	engine, _ := NewEngine(input)
	vars := map[string]any{
		"a": int64(50), "b": int64(60), "c": int64(10), "d": int64(5), "e": "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}

func BenchmarkComplexExpression_VM(b *testing.B) {
	input := `if (a + b) * (c - d) > 100 && e == "test" then f = 1`
	engine, _ := NewEngineVM(input)
	vars := map[string]any{
		"a": int64(50), "b": int64(60), "c": int64(10), "d": int64(5), "e": "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}

func BenchmarkComplexExpression_OptNone(b *testing.B) {
	input := `if (a + b) * (c - d) > 100 && e == "test" then f = 1`
	engine, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptNone})
	vars := map[string]any{
		"a": int64(50), "b": int64(60), "c": int64(10), "d": int64(5), "e": "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}

func BenchmarkComplexExpression_OptBasic(b *testing.B) {
	input := `if (a + b) * (c - d) > 100 && e == "test" then f = 1`
	engine, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptBasic})
	vars := map[string]any{
		"a": int64(50), "b": int64(60), "c": int64(10), "d": int64(5), "e": "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}

func BenchmarkComplexExpression_Recompiled(b *testing.B) {
	input := `if (a + b) * (c - d) > 100 && e == "test" then f = 1`
	engine, _ := NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptBasic, UseRecompiler: true})
	vars := map[string]any{
		"a": int64(50), "b": int64(60), "c": int64(10), "d": int64(5), "e": "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}
