// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"math"
	"strconv"
	"sync"
)

type compilationValue struct {
	isConst  bool
	val      Value
	isString bool
}

type NeoCompiler struct {
	lexer     *Lexer
	curToken  Token
	peekToken Token
	
	instructions []neoInstruction
	constants    []Value
	
	intConsts    map[uint64]int32
	strConsts    map[string]int32
	floatConsts  map[uint64]int32
	boolConsts   [2]int32
	hasNilConst  bool
	nilConstIdx  int32

	discard bool // New: discard emitted instructions
	errors  []string
}

var neoCompilerPool = sync.Pool{
	New: func() any {
		return &NeoCompiler{
			intConsts:    make(map[uint64]int32),
			strConsts:    make(map[string]int32),
			floatConsts:  make(map[uint64]int32),
			boolConsts:   [2]int32{-1, -1},
			instructions: make([]neoInstruction, 0, 128),
			constants:    make([]Value, 0, 32),
		}
	},
}

func NewNeoCompiler(input string) *NeoCompiler {
	l := lexerPool.Get().(*Lexer)
	l.Reset(input)
	c := neoCompilerPool.Get().(*NeoCompiler)
	c.lexer = l
	c.nextToken()
	c.nextToken()
	return c
}

func (c *NeoCompiler) Close() {
	if c.lexer != nil {
		lexerPool.Put(c.lexer)
		c.lexer = nil
	}
	c.instructions = c.instructions[:0]
	c.constants = c.constants[:0]
	for k := range c.intConsts { delete(c.intConsts, k) }
	for k := range c.strConsts { delete(c.strConsts, k) }
	for k := range c.floatConsts { delete(c.floatConsts, k) }
	c.boolConsts = [2]int32{-1, -1}
	c.hasNilConst = false
	c.errors = nil
	c.discard = false
	neoCompilerPool.Put(c)
}

func (c *NeoCompiler) nextToken() {
	c.curToken = c.peekToken
	c.peekToken = c.lexer.NextToken()
}

func (c *NeoCompiler) Compile() (*NeoBytecode, error) {
	defer c.Close()
	val, err := c.parseExpression(LOWEST)
	if err != nil {
		return nil, err
	}
	
	if val.isConst {
		c.emitPush(val.val)
	}
	
	if len(c.errors) > 0 {
		return nil, fmt.Errorf("compile errors: %v", c.errors)
	}
	
	c.emit(NeoOpReturn, 0)
	
	finalInsts := make([]neoInstruction, len(c.instructions))
	copy(finalInsts, c.instructions)
	finalConsts := make([]Value, len(c.constants))
	copy(finalConsts, c.constants)
	
	return &NeoBytecode{
		Instructions: finalInsts,
		Constants:    finalConsts,
	}, nil
}

func (c *NeoCompiler) parseExpression(precedence int) (compilationValue, error) {
	prefix := c.getPrefixFn(c.curToken.Type)
	if prefix == nil {
		return compilationValue{}, fmt.Errorf("no prefix parsing function for %s", c.curToken.Type)
	}
	
	left, err := prefix()
	if err != nil {
		return compilationValue{}, err
	}
	
	for c.peekToken.Type != TokenEOF && precedence < c.peekPrecedence() {
		infix := c.getInfixFn(c.peekToken.Type)
		if infix == nil {
			return left, nil
		}
		
		c.nextToken()
		left, err = infix(left)
		if err != nil {
			return compilationValue{}, err
		}
	}
	
	return left, nil
}

func (c *NeoCompiler) curPrecedence() int {
	return getPrecedence(c.curToken.Type)
}

func (c *NeoCompiler) peekPrecedence() int {
	return getPrecedence(c.peekToken.Type)
}

func (c *NeoCompiler) getPrefixFn(t TokenType) func() (compilationValue, error) {
	switch t {
	case TokenIdent: return c.parseIdentifier
	case TokenNumber: return c.parseNumberLiteral
	case TokenString: return c.parseStringLiteral
	case TokenTrue, TokenFalse: return c.parseBooleanLiteral
	case TokenBang, TokenMinus: return c.parsePrefixExpression
	case TokenLParen: return c.parseGroupedExpression
	case TokenIf: return c.parseIfExpression
	default: return nil
	}
}

func (c *NeoCompiler) getInfixFn(t TokenType) func(compilationValue) (compilationValue, error) {
	switch t {
	case TokenPlus, TokenMinus, TokenAsterisk, TokenSlash, TokenPercent,
		TokenEq, TokenGt, TokenLt, TokenGe, TokenLe, TokenAnd, TokenOr:
		return c.parseInfixExpression
	case TokenAssign:
		return c.parseAssignExpression
	case TokenLParen:
		return c.parseCallExpression
	default:
		return nil
	}
}

func (c *NeoCompiler) parseIdentifier() (compilationValue, error) {
	c.emit(NeoOpGetGlobal, c.addConstant(Value{Type: ValString, Str: c.curToken.Literal}))
	return compilationValue{isConst: false}, nil
}

func neoContainsDot(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return true
		}
	}
	return false
}

func (c *NeoCompiler) parseNumberLiteral() (compilationValue, error) {
	v, err := strconv.ParseFloat(c.curToken.Literal, 64)
	if err != nil {
		return compilationValue{}, err
	}
	var val Value
	if !neoContainsDot(c.curToken.Literal) {
		val = Value{Type: ValInt, Num: uint64(int64(v))}
	} else {
		val = Value{Type: ValFloat, Num: math.Float64bits(v)}
	}
	return compilationValue{isConst: true, val: val}, nil
}

func (c *NeoCompiler) parseStringLiteral() (compilationValue, error) {
	return compilationValue{isConst: true, val: Value{Type: ValString, Str: c.curToken.Literal}, isString: true}, nil
}

func (c *NeoCompiler) parseBooleanLiteral() (compilationValue, error) {
	val := uint64(0)
	if c.curToken.Type == TokenTrue { val = 1 }
	return compilationValue{isConst: true, val: Value{Type: ValBool, Num: val}}, nil
}

func (c *NeoCompiler) parsePrefixExpression() (compilationValue, error) {
	op := c.curToken.Literal
	c.nextToken()
	right, err := c.parseExpression(PREFIX)
	if err != nil { return compilationValue{}, err }
	
	if right.isConst {
		if op == "-" {
			if right.val.Type == ValInt {
				return compilationValue{isConst: true, val: Value{Type: ValInt, Num: uint64(-int64(right.val.Num))}}, nil
			} else if right.val.Type == ValFloat {
				fv := math.Float64frombits(right.val.Num)
				return compilationValue{isConst: true, val: Value{Type: ValFloat, Num: math.Float64bits(-fv)}}, nil
			}
		} else if op == "!" {
			return compilationValue{isConst: true, val: Value{Type: ValBool, Num: boolToUint64(!isValTruthy(right.val))}}, nil
		}
	}
	
	if right.isConst {
		c.emitPush(right.val)
	}
	
	if op == "-" {
		c.emit(NeoOpPush, c.addConstant(Value{Type: ValInt, Num: 0}))
		c.emit(NeoOpSub, 0)
	} else if op == "!" {
		c.emit(NeoOpNot, 0)
	}
	return compilationValue{isConst: false}, nil
}

func (c *NeoCompiler) parseGroupedExpression() (compilationValue, error) {
	c.nextToken()
	val, err := c.parseExpression(LOWEST)
	if err != nil { return compilationValue{}, err }
	if c.peekToken.Type != TokenRParen {
		return compilationValue{}, fmt.Errorf("expected ), got %s", c.peekToken.Type)
	}
	c.nextToken()
	return val, nil
}

func (c *NeoCompiler) peekTokenIsLiteral() bool {
	t := c.peekToken.Type
	return t == TokenNumber || t == TokenString || t == TokenTrue || t == TokenFalse
}

func (c *NeoCompiler) parseInfixExpression(left compilationValue) (compilationValue, error) {
	op := c.curToken.Literal
	precedence := c.curPrecedence()

	if op == "+" && left.isString {
		lastIdx := len(c.instructions) - 1
		canFuse := lastIdx >= 0 && c.instructions[lastIdx].Op == NeoOpConcat
		var nArgs int32
		if canFuse {
			nArgs = c.instructions[lastIdx].Arg
			c.instructions = c.instructions[:lastIdx]
		}
		if left.isConst && !c.peekTokenIsLiteral() {
			c.emitPush(left.val)
			left.isConst = false
		}
		c.nextToken()
		right, err := c.parseExpression(precedence)
		if err != nil { return compilationValue{}, err }
		if left.isConst && right.isConst {
			res, ok := c.foldInfix(left.val, right.val, op)
			if ok { return compilationValue{isConst: true, val: res, isString: true}, nil }
		}
		if left.isConst { c.emitPush(left.val) }
		if right.isConst { c.emitPush(right.val) }
		if canFuse { c.emit(NeoOpConcat, nArgs+1) } else { c.emit(NeoOpConcat, 2) }
		return compilationValue{isConst: false, isString: true}, nil
	}

	if op == "&&" {
		if left.isConst {
			if isValTruthy(left.val) {
				c.nextToken()
				return c.parseExpression(precedence)
			} else {
				oldDiscard := c.discard
				c.discard = true
				c.nextToken()
				c.parseExpression(precedence)
				c.discard = oldDiscard
				return left, nil
			}
		}
		jumpFalse := c.emit(NeoOpJumpIfFalse, 0)
		c.nextToken()
		right, err := c.parseExpression(precedence)
		if err != nil { return compilationValue{}, err }
		if right.isConst { c.emitPush(right.val) }
		c.emit(NeoOpNot, 0)
		c.emit(NeoOpNot, 0)
		jumpEnd := c.emit(NeoOpJump, 0)
		c.patch(jumpFalse, int32(len(c.instructions)))
		c.emit(NeoOpPush, c.addConstant(Value{Type: ValBool, Num: 0}))
		c.patch(jumpEnd, int32(len(c.instructions)))
		return compilationValue{isConst: false}, nil
	}

	if op == "||" {
		if left.isConst {
			if !isValTruthy(left.val) {
				c.nextToken()
				return c.parseExpression(precedence)
			} else {
				oldDiscard := c.discard
				c.discard = true
				c.nextToken()
				c.parseExpression(precedence)
				c.discard = oldDiscard
				return left, nil
			}
		}
		jumpTrue := c.emit(NeoOpJumpIfTrue, 0)
		c.nextToken()
		right, err := c.parseExpression(precedence)
		if err != nil { return compilationValue{}, err }
		if right.isConst { c.emitPush(right.val) }
		c.emit(NeoOpNot, 0)
		c.emit(NeoOpNot, 0)
		jumpEnd := c.emit(NeoOpJump, 0)
		c.patch(jumpTrue, int32(len(c.instructions)))
		c.emit(NeoOpPush, c.addConstant(Value{Type: ValBool, Num: 1}))
		c.patch(jumpEnd, int32(len(c.instructions)))
		return compilationValue{isConst: false}, nil
	}

	if left.isConst && !c.peekTokenIsLiteral() {
		c.emitPush(left.val)
		left.isConst = false
	}
	c.nextToken()
	right, err := c.parseExpression(precedence)
	if err != nil { return compilationValue{}, err }
	if left.isConst && right.isConst {
		res, ok := c.foldInfix(left.val, right.val, op)
		if ok { return compilationValue{isConst: true, val: res}, nil }
	}
	if (left.isString || right.isString) && op == "+" {
		if left.isConst { c.emitPush(left.val) }
		if right.isConst { c.emitPush(right.val) }
		c.emit(NeoOpConcat, 2)
		return compilationValue{isConst: false, isString: true}, nil
	}

	// Algebraic Simplifications
	if op == "+" {
		if left.isConst && neoIsZero(left.val) { return right, nil }
		if right.isConst && neoIsZero(right.val) { return left, nil }
	} else if op == "-" {
		if right.isConst && neoIsZero(right.val) { return left, nil }
	} else if op == "*" {
		if left.isConst {
			if neoIsZero(left.val) { return left, nil }
			if neoIsOne(left.val) { return right, nil }
		}
		if right.isConst {
			if neoIsZero(right.val) { return right, nil }
			if neoIsOne(right.val) { return left, nil }
		}
	} else if op == "/" {
		if right.isConst && neoIsOne(right.val) { return left, nil }
	}

	if left.isConst { c.emitPush(left.val) }
	if right.isConst { c.emitPush(right.val) }

	switch op {
	case "+":
		if left.isString || right.isString {
			lastIdx := len(c.instructions) - 1
			if lastIdx >= 0 && c.instructions[lastIdx].Op == NeoOpConcat {
				nArgs := c.instructions[lastIdx].Arg
				c.instructions = c.instructions[:lastIdx]
				if right.isConst { c.emitPush(right.val) }
				c.emit(NeoOpConcat, nArgs+1)
				return compilationValue{isConst: false, isString: true}, nil
			}
			if left.isConst { c.emitPush(left.val) }
			if right.isConst { c.emitPush(right.val) }
			c.emit(NeoOpConcat, 2)
			return compilationValue{isConst: false, isString: true}, nil
		}
		c.emit(NeoOpAdd, 0)
	case "-": c.emit(NeoOpSub, 0)
	case "*": c.emit(NeoOpMul, 0)
	case "/": c.emit(NeoOpDiv, 0)
	case "%": c.emit(NeoOpMod, 0)
	case "==": c.emit(NeoOpEqual, 0)
	case ">": c.emit(NeoOpGreater, 0)
	case "<": c.emit(NeoOpLess, 0)
	case ">=": c.emit(NeoOpGreaterEqual, 0)
	case "<=": c.emit(NeoOpLessEqual, 0)
	}
	return compilationValue{isConst: false}, nil
}

func (c *NeoCompiler) foldInfix(l, r Value, op string) (Value, bool) {
	switch op {
	case "+":
		if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num + r.Num}, true }
		if l.Type == ValString && r.Type == ValString { return Value{Type: ValString, Str: l.Str + r.Str}, true }
		if (l.Type == ValInt || l.Type == ValFloat) && (r.Type == ValInt || r.Type == ValFloat) {
			lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
			return Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}, true
		}
	case "-":
		if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num - r.Num}, true }
		if (l.Type == ValInt || l.Type == ValFloat) && (r.Type == ValInt || r.Type == ValFloat) {
			lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
			return Value{Type: ValFloat, Num: math.Float64bits(lf - rf)}, true
		}
	case "*":
		if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num * r.Num}, true }
		if (l.Type == ValInt || l.Type == ValFloat) && (r.Type == ValInt || r.Type == ValFloat) {
			lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
			return Value{Type: ValFloat, Num: math.Float64bits(lf * rf)}, true
		}
	case "/":
		if (r.Type == ValInt && r.Num == 0) || (r.Type == ValFloat && math.Float64frombits(r.Num) == 0) {
			c.errors = append(c.errors, "division by zero")
			return Value{}, false
		}
		if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num / r.Num}, true }
		if (l.Type == ValInt || l.Type == ValFloat) && (r.Type == ValInt || r.Type == ValFloat) {
			lf, _ := valToFloat64(l); rf, _ := valToFloat64(r)
			return Value{Type: ValFloat, Num: math.Float64bits(lf / rf)}, true
		}
	case "%":
		if r.Type == ValInt && r.Num == 0 { c.errors = append(c.errors, "division by zero"); return Value{}, false }
		if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num % r.Num}, true }
	case "==": return Value{Type: ValBool, Num: boolToUint64(c.compare(l, r) == 0)}, true
	case ">": return Value{Type: ValBool, Num: boolToUint64(c.compare(l, r) > 0)}, true
	case "<": return Value{Type: ValBool, Num: boolToUint64(c.compare(l, r) < 0)}, true
	case ">=": return Value{Type: ValBool, Num: boolToUint64(c.compare(l, r) >= 0)}, true
	case "<=": return Value{Type: ValBool, Num: boolToUint64(c.compare(l, r) <= 0)}, true
	}
	return Value{}, false
}

func (c *NeoCompiler) compare(l, r Value) int {
	if l.Type == r.Type {
		switch l.Type {
		case ValInt:
			if int64(l.Num) < int64(r.Num) { return -1 }
			if int64(l.Num) > int64(r.Num) { return 1 }
			return 0
		case ValFloat:
			lf := math.Float64frombits(l.Num); rf := math.Float64frombits(r.Num)
			if lf < rf { return -1 }
			if lf > rf { return 1 }
			return 0
		case ValString:
			if l.Str < r.Str { return -1 }
			if l.Str > r.Str { return 1 }
			return 0
		case ValBool:
			if l.Num < r.Num { return -1 }
			if l.Num > r.Num { return 1 }
			return 0
		case ValNil: return 0
		}
	}
	lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
	if okL && okR {
		if lf < rf { return -1 }
		if lf > rf { return 1 }
		return 0
	}
	return 0
}

func (c *NeoCompiler) parseAssignExpression(left compilationValue) (compilationValue, error) {
	if left.isConst { return compilationValue{}, fmt.Errorf("left side of assignment must be an identifier") }
	if c.discard {
		c.nextToken()
		_, err := c.parseExpression(ASSIGN)
		return compilationValue{isConst: false}, err
	}
	lastInst := c.instructions[len(c.instructions)-1]
	if lastInst.Op != NeoOpGetGlobal { return compilationValue{}, fmt.Errorf("left side of assignment must be an identifier") }
	identIdx := lastInst.Arg
	c.instructions = c.instructions[:len(c.instructions)-1]
	c.nextToken()
	val, err := c.parseExpression(ASSIGN)
	if err != nil { return compilationValue{}, err }
	if val.isConst { c.emitPush(val.val) }
	c.emit(NeoOpSetGlobal, identIdx)
	return compilationValue{isConst: false}, nil
}

func (c *NeoCompiler) parseCallExpression(left compilationValue) (compilationValue, error) {
	if left.isConst { return compilationValue{}, fmt.Errorf("function call must be on an identifier") }
	if c.discard {
		numArgs := 0
		if c.peekToken.Type != TokenRParen {
			c.nextToken(); c.parseExpression(LOWEST); numArgs++
			for c.peekToken.Type == TokenComma { c.nextToken(); c.nextToken(); c.parseExpression(LOWEST); numArgs++ }
		}
		if c.peekToken.Type != TokenRParen { return compilationValue{}, fmt.Errorf("expected ), got %s", c.peekToken.Type) }
		c.nextToken(); return compilationValue{isConst: false}, nil
	}
	lastInst := c.instructions[len(c.instructions)-1]
	if lastInst.Op != NeoOpGetGlobal { return compilationValue{}, fmt.Errorf("function call must be on an identifier") }
	funcNameIdx := lastInst.Arg
	c.instructions = c.instructions[:len(c.instructions)-1]
	numArgs := 0
	if c.peekToken.Type != TokenRParen {
		c.nextToken(); val, err := c.parseExpression(LOWEST)
		if err != nil { return compilationValue{}, err }
		if val.isConst { c.emitPush(val.val) }
		numArgs++
		for c.peekToken.Type == TokenComma {
			c.nextToken(); c.nextToken(); val, err = c.parseExpression(LOWEST)
			if err != nil { return compilationValue{}, err }
			if val.isConst { c.emitPush(val.val) }
			numArgs++
		}
	}
	if c.peekToken.Type != TokenRParen { return compilationValue{}, fmt.Errorf("expected ), got %s", c.peekToken.Type) }
	c.nextToken()
	funcName := c.constants[funcNameIdx].Str
	if funcName == "concat" {
		if numArgs == 2 { c.emit(NeoOpConcat2, 0) } else { c.emit(NeoOpConcat, int32(numArgs)) }
	} else { c.emit(NeoOpCall, funcNameIdx | int32(numArgs << 16)) }
	return compilationValue{isConst: false}, nil
}

func (c *NeoCompiler) parseIfExpression() (compilationValue, error) {
	c.nextToken(); cond, err := c.parseExpression(LOWEST)
	if err != nil { return compilationValue{}, err }
	if c.peekToken.Type == TokenThen {
		c.nextToken(); c.nextToken()
		if cond.isConst {
			if isValTruthy(cond.val) { return c.parseExpression(LOWEST) } else {
				oldDiscard := c.discard; c.discard = true; _, err = c.parseExpression(LOWEST); c.discard = oldDiscard
				return compilationValue{isConst: true, val: Value{Type: ValNil}}, err
			}
		}
		jumpFalse := c.emit(NeoOpJumpIfFalse, 0)
		cons, err := c.parseExpression(LOWEST)
		if err != nil { return compilationValue{}, err }
		if cons.isConst { c.emitPush(cons.val) }
		c.patch(jumpFalse, int32(len(c.instructions)))
		return compilationValue{isConst: false}, nil
	}
	if c.peekToken.Type == TokenIs {
		var jumpEndTargets []int
		for {
			if c.peekToken.Type != TokenIs { return compilationValue{}, fmt.Errorf("expected is after if condition, got %s", c.peekToken.Type) }
			c.nextToken(); c.nextToken(); var jumpFalse int; var tookBranch bool
			if cond.isConst {
				if isValTruthy(cond.val) {
					cons, err := c.parseExpression(LOWEST); if err != nil { return compilationValue{}, err }
					if cons.isConst { c.emitPush(cons.val) }; tookBranch = true
				} else { oldDiscard := c.discard; c.discard = true; c.parseExpression(LOWEST); c.discard = oldDiscard }
			} else {
				jumpFalse = c.emit(NeoOpJumpIfFalse, 0)
				cons, err := c.parseExpression(LOWEST); if err != nil { return compilationValue{}, err }
				if cons.isConst { c.emitPush(cons.val) }
				jumpEndTargets = append(jumpEndTargets, c.emit(NeoOpJump, 0)); c.patch(jumpFalse, int32(len(c.instructions)))
			}
			if tookBranch {
				for c.peekToken.Type == TokenElse {
					c.nextToken()
					if c.peekToken.Type == TokenIf {
						c.nextToken(); c.nextToken(); oldDiscard := c.discard; c.discard = true; c.parseExpression(LOWEST)
						if c.peekToken.Type == TokenIs { c.nextToken(); c.nextToken(); c.parseExpression(LOWEST) }
						c.discard = oldDiscard
					} else if c.peekToken.Type == TokenIs {
						c.nextToken(); c.nextToken(); oldDiscard := c.discard; c.discard = true; c.parseExpression(LOWEST); c.discard = oldDiscard; break
					}
				}
				break
			}
			if c.peekToken.Type != TokenElse { c.emitPush(Value{Type: ValNil}); break }
			c.nextToken()
			if c.peekToken.Type == TokenIf { c.nextToken(); c.nextToken(); cond, err = c.parseExpression(LOWEST); if err != nil { return compilationValue{}, err }
				continue
			}
			if c.peekToken.Type == TokenIs {
				c.nextToken(); c.nextToken(); alt, err := c.parseExpression(LOWEST); if err != nil { return compilationValue{}, err }
				if alt.isConst { c.emitPush(alt.val) }; break
			}
			return compilationValue{}, fmt.Errorf("expected if or is after else, got %s", c.peekToken.Type)
		}
		for _, target := range jumpEndTargets { c.patch(target, int32(len(c.instructions))) }
		return compilationValue{isConst: false}, nil
	}
	return compilationValue{}, fmt.Errorf("expected then or is after if condition, got %s", c.peekToken.Type)
}

func (c *NeoCompiler) emit(op NeoOpCode, arg int32) int {
	if c.discard { return -1 }

	n := len(c.instructions)
	if n >= 1 {
		last := &c.instructions[n-1]

		// Online fusions
		switch op {
		case NeoOpJumpIfFalse:
			if last.Op == NeoOpEqualGlobalConst {
				gIdx := last.Arg >> 16; cIdx := last.Arg & 0xFFFF
				if gIdx < 1024 && cIdx < 1024 && arg < 4096 {
					last.Op = NeoOpFusedCompareGlobalConstJumpIfFalse
					last.Arg = (gIdx << 22) | (cIdx << 12) | arg
					return n - 1
				}
			} else if last.Op == NeoOpGreaterGlobalConst {
				gIdx := last.Arg >> 16; cIdx := last.Arg & 0xFFFF
				if gIdx < 1024 && cIdx < 1024 && arg < 4096 {
					last.Op = NeoOpFusedGreaterGlobalConstJumpIfFalse
					last.Arg = (gIdx << 22) | (cIdx << 12) | arg
					return n - 1
				}
			} else if last.Op == NeoOpLessGlobalConst {
				gIdx := last.Arg >> 16; cIdx := last.Arg & 0xFFFF
				if gIdx < 1024 && cIdx < 1024 && arg < 4096 {
					last.Op = NeoOpFusedLessGlobalConstJumpIfFalse
					last.Arg = (gIdx << 22) | (cIdx << 12) | arg
					return n - 1
				}
			}
		case NeoOpAdd, NeoOpSub, NeoOpMul, NeoOpDiv, NeoOpEqual, NeoOpGreater, NeoOpLess:
			if last.Op == NeoOpPush {
				opC := NeoOpCode(0)
				switch op {
				case NeoOpAdd: opC = NeoOpAddC
				case NeoOpSub: opC = NeoOpSubC
				case NeoOpMul: opC = NeoOpMulC
				case NeoOpDiv: opC = NeoOpDivC
				case NeoOpEqual: opC = NeoOpEqualC
				case NeoOpGreater: opC = NeoOpGreaterC
				case NeoOpLess: opC = NeoOpLessC
				}
				if opC != 0 {
					last.Op = opC
					return n - 1
				}
			}
		case NeoOpJumpIfTrue:
			if last.Op == NeoOpGetGlobal {
				if last.Arg < 65536 && arg < 65536 {
					last.Op = NeoOpGetGlobalJumpIfTrue
					last.Arg = (last.Arg << 16) | arg
					return n - 1
				}
			}
		}

		if n >= 2 {
			prev := &c.instructions[n-2]
			// 3-instruction fusions: GetG + Push + Op
			if prev.Op == NeoOpGetGlobal && last.Op == NeoOpPush {
				opGC := NeoOpCode(0)
				switch op {
				case NeoOpAdd: opGC = NeoOpAddGC
				case NeoOpSub: opGC = NeoOpSubGC
				case NeoOpMul: opGC = NeoOpMulGC
				case NeoOpDiv: opGC = NeoOpDivGC
				case NeoOpEqual: opGC = NeoOpEqualGlobalConst
				case NeoOpGreater: opGC = NeoOpGreaterGlobalConst
				case NeoOpLess: opGC = NeoOpLessGlobalConst
				}
				if opGC != 0 && prev.Arg < 65536 && last.Arg < 65536 {
					prev.Op = opGC
					prev.Arg = (prev.Arg << 16) | last.Arg
					c.instructions = c.instructions[:n-1]
					return n - 2
				}
			}
			// 3-instruction fusion: Push + Push + Op (Constant Fold)
			if prev.Op == NeoOpPush && last.Op == NeoOpPush {
				c1 := c.constants[prev.Arg]; c2 := c.constants[last.Arg]
				var opStr string
				switch op {
				case NeoOpAdd: opStr = "+"
				case NeoOpSub: opStr = "-"
				case NeoOpMul: opStr = "*"
				case NeoOpDiv: opStr = "/"
				case NeoOpEqual: opStr = "=="
				case NeoOpGreater: opStr = ">"
				case NeoOpLess: opStr = "<"
				}
				if opStr != "" {
					res, ok := c.foldInfix(c1, c2, opStr)
					if ok {
						prev.Arg = c.addConstant(res)
						c.instructions = c.instructions[:n-1]
						return n - 2
					}
				}
			}
			// 3-instruction fusions: Push + GetG + Op
			if prev.Op == NeoOpPush && last.Op == NeoOpGetGlobal {
				opCG := NeoOpCode(0)
				switch op {
				case NeoOpAdd: opCG = NeoOpAddConstGlobal
				case NeoOpSub: opCG = NeoOpSubCG
				case NeoOpMul: opCG = NeoOpMulCG
				case NeoOpDiv: opCG = NeoOpDivCG
				}
				if opCG != 0 && last.Arg < 65536 && prev.Arg < 65536 {
					prev.Op = opCG
					prev.Arg = (last.Arg << 16) | prev.Arg
					c.instructions = c.instructions[:n-1]
					return n - 2
				}
			}
			// 3-instruction fusions: GetG + GetG + Op
			if prev.Op == NeoOpGetGlobal && last.Op == NeoOpGetGlobal {
				opGG := NeoOpCode(0)
				switch op {
				case NeoOpAdd: opGG = NeoOpAddGlobalGlobal
				case NeoOpSub: opGG = NeoOpSubGlobalGlobal
				case NeoOpMul: opGG = NeoOpMulGlobalGlobal
				}
				if opGG != 0 && prev.Arg < 65536 && last.Arg < 65536 {
					prev.Op = opGG
					prev.Arg = (prev.Arg << 16) | last.Arg
					c.instructions = c.instructions[:n-1]
					return n - 2
				}
			}
		}
	}

	c.instructions = append(c.instructions, neoInstruction{Op: op, Arg: arg})
	return len(c.instructions) - 1
}

func (c *NeoCompiler) emitPush(v Value) int { return c.emit(NeoOpPush, c.addConstant(v)) }

func (c *NeoCompiler) patch(pos int, arg int32) { c.instructions[pos].Arg = arg }

func neoIsZero(v Value) bool {
	switch v.Type {
	case ValInt: return v.Num == 0
	case ValFloat: return math.Float64frombits(v.Num) == 0
	}
	return false
}

func neoIsOne(v Value) bool {
	switch v.Type {
	case ValInt: return v.Num == 1
	case ValFloat: return math.Float64frombits(v.Num) == 1
	}
	return false
}

func (c *NeoCompiler) addConstant(v Value) int32 {
	// Linear search for small number of constants to avoid map overhead
	if len(c.constants) < 16 {
		for i, cv := range c.constants {
			if cv.Type == v.Type {
				if cv.Type == ValString {
					if cv.Str == v.Str { return int32(i) }
				} else if cv.Num == v.Num {
					return int32(i)
				}
			}
		}
	}

	switch v.Type {
	case ValInt:
		if idx, ok := c.intConsts[v.Num]; ok { return idx }
		idx := int32(len(c.constants)); c.constants = append(c.constants, v); c.intConsts[v.Num] = idx
		return idx
	case ValFloat:
		if idx, ok := c.floatConsts[v.Num]; ok { return idx }
		idx := int32(len(c.constants)); c.constants = append(c.constants, v); c.floatConsts[v.Num] = idx
		return idx
	case ValString:
		if idx, ok := c.strConsts[v.Str]; ok { return idx }
		idx := int32(len(c.constants)); c.constants = append(c.constants, v); c.strConsts[v.Str] = idx
		return idx
	case ValBool:
		if c.boolConsts[v.Num] != -1 { return c.boolConsts[v.Num] }
		idx := int32(len(c.constants)); c.constants = append(c.constants, v); c.boolConsts[v.Num] = idx
		return idx
	case ValNil:
		if c.hasNilConst { return c.nilConstIdx }
		idx := int32(len(c.constants)); c.constants = append(c.constants, v); c.hasNilConst = true; c.nilConstIdx = idx
		return idx
	}
	return -1
}

