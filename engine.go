// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"github.com/kamihama-railway/uwasa/rvm"
)

type Engine struct {
	program          Node
	bytecode         *RenderedBytecode
	registerBytecode *rvm.RegisterBytecode
	opts             EngineOptions
	isConstant       bool
	constantResult   any
}

type OptimizationLevel int

const (
	OptNone OptimizationLevel = iota
	OptBasic
)

type EngineOptions struct {
	OptimizationLevel OptimizationLevel
	UseVM              bool
	UseRecompiler      bool
}

func NewEngine(input string) (*Engine, error) {
	return NewEngineWithOptions(input, EngineOptions{
		OptimizationLevel: OptBasic,
		UseVM:              true,
		UseRecompiler:      false,
	})
}

// NewEngineVM is a convenience function that creates an engine with VM enabled.
func NewEngineVM(input string) (*Engine, error) {
	return NewEngineWithOptions(input, EngineOptions{
		OptimizationLevel: OptBasic,
		UseVM:              true,
		UseRecompiler:      false,
	})
}

func NewEngineWithOptions(input string, opts EngineOptions) (*Engine, error) {
	l := NewLexer(input)
	defer LexerPool.Put(l)
	p := NewParser(l)
	defer ParserPool.Put(p)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return nil, fmt.Errorf("parse errors: %v", p.Errors())
	}

	e := &Engine{
		program: program,
		opts:    opts,
	}

	if opts.OptimizationLevel >= OptBasic {
		e.program = Fold(e.program)
	}

	if opts.UseRecompiler {
		re := NewRecompiler()
		optimized, err := re.Optimize(e.program)
		if err != nil {
			return nil, err
		}
		e.program = optimized
	}

	// Constant path optimization
	if program != nil {
		switch n := e.program.(type) {
		case *NumberLiteral:
			e.isConstant = true
			if n.IsInt {
				e.constantResult = n.Int64Value
			} else {
				e.constantResult = n.Float64Value
			}
		case *StringLiteral:
			e.isConstant = true
			e.constantResult = n.Value
		case *BooleanLiteral:
			e.isConstant = true
			e.constantResult = n.Value
		}
	} else {
		// Nil program
		e.isConstant = true
		e.constantResult = nil
	}

	if opts.UseVM && !e.isConstant {
		c := NewVMCompiler()
		bc, err := c.Compile(e.program)
		if err != nil {
			return nil, err
		}
		e.bytecode = bc
	}

	return e, nil
}

func (e *Engine) UseRegisterVM() error {
	if e.isConstant {
		return nil
	}
	c := rvm.NewRegisterCompiler()
	bc, err := c.Compile(e.program)
	if err != nil {
		return err
	}
	e.registerBytecode = bc
	e.bytecode = nil // Clear stack bytecode
	return nil
}

func (e *Engine) Execute(vars map[string]any) (any, error) {
	if e.isConstant {
		return e.constantResult, nil
	}

	ctx := NewMapContext(vars)
	defer func() {
		ctx.Vars = nil
		MapContextPool.Put(ctx)
	}()
	if e.registerBytecode != nil {
		return rvm.RunRegisterVM(e.registerBytecode, ctx)
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
		return rvm.RunRegisterVM(e.registerBytecode, ctx)
	}
	if e.bytecode != nil {
		return RunVM(e.bytecode, ctx)
	}
	return Eval(e.program, ctx)
}
