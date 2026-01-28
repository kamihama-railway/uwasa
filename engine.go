// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
)

type OptimizationLevel int

const (
	OptNone OptimizationLevel = iota
	OptBasic
)

type EngineOptions struct {
	OptimizationLevel OptimizationLevel
	UseRecompiler     bool
	UseVM             bool
}

type Engine struct {
	program  Expression
	bytecode *Bytecode
	rendered *RenderedBytecode
}

func NewEngine(input string) (*Engine, error) {
	return NewEngineWithOptions(input, EngineOptions{OptimizationLevel: OptBasic})
}

func NewEngineWithOptions(input string, opts EngineOptions) (*Engine, error) {
	l := NewLexer(input)
	defer lexerPool.Put(l)
	p := NewParser(l)
	defer parserPool.Put(p)

	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return nil, fmt.Errorf("parser errors: %v", p.Errors())
	}

	var optimized Node = program
	if opts.OptimizationLevel >= OptBasic {
		optimized = Fold(optimized)
	}

	if opts.UseRecompiler {
		re := NewRecompiler()
		var err error
		optimized, err = re.Optimize(optimized)
		if err != nil {
			return nil, err
		}
	}

	if optimized == nil {
		return &Engine{program: nil}, nil
	}

	var bc *Bytecode
	var rendered *RenderedBytecode
	if opts.UseVM {
		comp := NewCompiler()
		err := comp.Compile(optimized)
		if err != nil {
			return nil, err
		}
		bc = comp.Bytecode()
		rendered = bc.Render()
	}

	return &Engine{
		program:  optimized.(Expression),
		bytecode: bc,
		rendered: rendered,
	}, nil
}

func (e *Engine) Execute(vars map[string]any) (any, error) {
	ctx := NewMapContext(vars)
	defer func() {
		ctx.vars = nil
		contextPool.Put(ctx)
	}()
	return e.ExecuteWithContext(ctx)
}

func (e *Engine) ExecuteWithContext(ctx Context) (any, error) {
	if e.rendered != nil {
		// Fast path for constant-only programs
		if len(e.rendered.Instructions) == 1 && e.rendered.Instructions[0].op == OpConstant {
			return e.rendered.Constants[e.rendered.Instructions[0].arg1].ToAny(), nil
		}
		return RunVM(e.rendered, ctx)
	}
	return Eval(e.program, ctx)
}
