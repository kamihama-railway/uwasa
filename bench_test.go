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

type benchContext struct {
	vars map[string]any
}

func (c *benchContext) Get(name string) any { return c.vars[name] }
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

func BenchmarkComplexExpression(b *testing.B) {
	input := `if (a + b) * (c - d) > 100 && e == "test" then f = 1`
	// Note: currently '*' is not implemented in Parser, let's use what's available
	input = `if (a + b) - (c - d) > 100 && e == "test" then f = 1`
	engine, _ := NewEngine(input)
	vars := map[string]any{
		"a": 50, "b": 60, "c": 10, "d": 5, "e": "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Execute(vars)
	}
}
