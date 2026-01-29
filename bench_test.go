package uwasa

import (
	"testing"
)

func BenchmarkEvaluator(b *testing.B) {
	input := "1 + 2 * 3 / 4 - 5"
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	ctx := NewMapContext(make(map[string]any))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Eval(program, ctx)
	}
}

func BenchmarkVM(b *testing.B) {
	input := "1 + 2 * 3 / 4 - 5"
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	comp := NewVMCompiler()
	bc, _ := comp.Compile(program)
	ctx := NewMapContext(make(map[string]any))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunVM(bc, ctx)
	}
}

func BenchmarkEngine(b *testing.B) {
	input := "1 + 2 * 3 / 4 - 5"
	eng, _ := NewEngine(input)
	vars := make(map[string]any)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.Execute(vars)
	}
}
