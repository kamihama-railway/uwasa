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
	UseRegisterVM     bool // Experimental: use register-based VM
}

type Engine struct {
	program          Expression
	bytecode         *RenderedBytecode
	registerBytecode *RegisterBytecode
	neoBytecode      *NeoBytecode
	constantResult   any
	isConstant       bool
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
		return &Engine{program: nil, isConstant: true}, nil
	}

	engine := &Engine{program: optimized.(Expression)}

	switch n := optimized.(type) {
	case *NumberLiteral, *StringLiteral, *BooleanLiteral:
		val, _ := Eval(n, nil)
		engine.constantResult = val
		engine.isConstant = true
	}

	return engine, nil
}

func NewEngineVMNeo(input string) (*Engine, error) {
	c := NewNeoCompiler(input)
	bc, err := c.Compile()
	if err != nil {
		return nil, err
	}
	// Constant detection
	if len(bc.Instructions) == 2 && bc.Instructions[0].Op == NeoOpPush && bc.Instructions[1].Op == NeoOpReturn {
		return &Engine{constantResult: bc.Constants[bc.Instructions[0].Arg].ToInterface(), isConstant: true}, nil
	}
	return &Engine{neoBytecode: bc}, nil
}

func NewEngineVM(input string) (*Engine, error) {
	return NewEngineVMWithOptions(input, EngineOptions{OptimizationLevel: OptBasic})
}

func NewEngineVMWithOptions(input string, opts EngineOptions) (*Engine, error) {
	l := NewLexer(input)
	defer lexerPool.Put(l)
	p := NewParser(l)
	defer parserPool.Put(p)

	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return nil, fmt.Errorf("parser errors: %v", p.Errors())
	}

	if opts.UseRegisterVM {
		c := NewRegisterCompiler()
		// For now, register VM compiler doesn't have the full optimized pipeline like VMCompiler
		// But we can manually fold
		var optimized Node = program
		if opts.OptimizationLevel >= OptBasic {
			optimized = Fold(optimized)
		}
		bc, err := c.Compile(optimized)
		if err != nil {
			return nil, err
		}
		// If the resulting bytecode is just returning a single constant, optimize it
		if bc != nil && len(bc.Instructions) == 2 && bc.Instructions[0].Op == ROpLoadConst && bc.Instructions[1].Op == ROpReturn {
			return &Engine{constantResult: bc.Constants[bc.Instructions[0].Arg].ToInterface(), isConstant: true}, nil
		}
		return &Engine{registerBytecode: bc}, nil
	}

	c := NewVMCompiler()
	// VMCompiler will handle its own optimization levels internally
	bc, err := c.CompileOptimized(program, opts)
	if err != nil {
		return nil, err
	}

	// If the resulting bytecode is just pushing a single constant, optimize it
	if bc != nil && len(bc.Instructions) == 1 && bc.Instructions[0].Op == OpPush {
		return &Engine{constantResult: bc.Constants[bc.Instructions[0].Arg].ToInterface(), isConstant: true}, nil
	}

	return &Engine{bytecode: bc}, nil
}

func (e *Engine) Execute(vars map[string]any) (any, error) {
	if e.isConstant {
		return e.constantResult, nil
	}

	if e.neoBytecode != nil {
		return RunNeoVMWithMap(e.neoBytecode, vars)
	}

	ctx := NewMapContext(vars)
	defer func() {
		ctx.vars = nil
		contextPool.Put(ctx)
	}()
	if e.registerBytecode != nil {
		return RunRegisterVM(e.registerBytecode, ctx)
	}
	if e.bytecode != nil {
		return RunVM(e.bytecode, ctx)
	}
	return Eval(e.program, ctx)
}

func (e *Engine) ExecuteWithContext(ctx Context) (any, error) {
	if e.isConstant {
		return e.constantResult, nil
	}

	if e.registerBytecode != nil {
		return RunRegisterVM(e.registerBytecode, ctx)
	}
	if e.neoBytecode != nil {
		return RunNeoVM(e.neoBytecode, ctx)
	}
	if e.bytecode != nil {
		return RunVM(e.bytecode, ctx)
	}
	return Eval(e.program, ctx)
}
