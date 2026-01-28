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
}

type Engine struct {
	program Expression
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
	return &Engine{program: optimized.(Expression)}, nil
}

func (e *Engine) Execute(vars map[string]any) (any, error) {
	ctx := NewMapContext(vars)
	defer func() {
		ctx.vars = nil
		contextPool.Put(ctx)
	}()
	return Eval(e.program, ctx)
}

func (e *Engine) ExecuteWithContext(ctx Context) (any, error) {
	return Eval(e.program, ctx)
}
