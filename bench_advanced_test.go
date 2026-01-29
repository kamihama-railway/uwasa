package uwasa

import (
	"testing"
)

func BenchmarkComplexExpressionVM(b *testing.B) {
	input := "if (x + y) * z > 100 then (x + y) * z else 0"
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	comp := NewVMCompiler()
	bc, _ := comp.Compile(program)

	vars := map[string]any{
		"x": int64(10),
		"y": int64(20),
		"z": int64(5),
	}
	ctx := NewMapContext(vars)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunVM(bc, ctx)
	}
}

func BenchmarkComplexExpressionAST(b *testing.B) {
	input := "if (x + y) * z > 100 then (x + y) * z else 0"
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	vars := map[string]any{
		"x": int64(10),
		"y": int64(20),
		"z": int64(5),
	}
	ctx := NewMapContext(vars)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Eval(program, ctx)
	}
}
