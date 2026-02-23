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

	constMapInt    map[int64]int32
	constMapFloat  map[uint64]int32
	constMapBool   map[bool]int32
	constMapString map[string]int32
	constMapOther  map[any]int32

	discard bool // New: discard emitted instructions
	errors  []string
}

var neoCompilerPool = sync.Pool{
	New: func() any {
		return &NeoCompiler{
			constMapInt:    make(map[int64]int32),
			constMapFloat:  make(map[uint64]int32),
			constMapBool:   make(map[bool]int32),
			constMapString: make(map[string]int32),
			constMapOther:  make(map[any]int32),
		}
	},
}

func NewNeoCompiler(input string) *NeoCompiler {
	l := lexerPool.Get().(*Lexer)
	l.Reset(input)
	c := neoCompilerPool.Get().(*NeoCompiler)
	c.lexer = l
	c.Reset()
	return c
}

func (c *NeoCompiler) Reset() {
	c.instructions = c.instructions[:0]
	c.constants = c.constants[:0]
	for k := range c.constMapInt {
		delete(c.constMapInt, k)
	}
	for k := range c.constMapFloat {
		delete(c.constMapFloat, k)
	}
	for k := range c.constMapBool {
		delete(c.constMapBool, k)
	}
	for k := range c.constMapString {
		delete(c.constMapString, k)
	}
	for k := range c.constMapOther {
		delete(c.constMapOther, k)
	}
	c.errors = c.errors[:0]
	c.discard = false
	c.nextToken()
	c.nextToken()
}

func (c *NeoCompiler) Close() {
	lexerPool.Put(c.lexer)
	c.lexer = nil
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

	c.peephole()
	c.emit(NeoOpReturn, 0)

	return &NeoBytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
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
	case TokenIdent:
		return c.parseIdentifier
	case TokenNumber:
		return c.parseNumberLiteral
	case TokenString:
		return c.parseStringLiteral
	case TokenTrue, TokenFalse:
		return c.parseBooleanLiteral
	case TokenBang, TokenMinus:
		return c.parsePrefixExpression
	case TokenLParen:
		return c.parseGroupedExpression
	case TokenIf:
		return c.parseIfExpression
	default:
		return nil
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
	case TokenDot:
		return c.parseMemberCallExpression
	case TokenSequence:
		return c.parseSequenceExpression
	default:
		return nil
	}
}

func (c *NeoCompiler) parseSequenceExpression(left compilationValue) (compilationValue, error) {
	if left.isConst {
		c.emitPush(left.val)
	}
	c.emit(NeoOpPop, 0)
	c.nextToken()
	return c.parseExpression(SEQUENCE)
}

func (c *NeoCompiler) parseMemberCallExpression(left compilationValue) (compilationValue, error) {
	if left.isConst {
		return compilationValue{}, fmt.Errorf("member call subject must be an identifier")
	}

	lastInst := c.instructions[len(c.instructions)-1]
	if lastInst.Op != NeoOpGetGlobal {
		return compilationValue{}, fmt.Errorf("member call subject must be an identifier")
	}

	c.nextToken() // cur is identifier (method name)
	if c.curToken.Type != TokenIdent {
		return compilationValue{}, fmt.Errorf("expected method name after '.'")
	}
	method := c.curToken.Literal

	if c.peekToken.Type != TokenLParen {
		return compilationValue{}, fmt.Errorf("expected '(' after method name")
	}
	c.nextToken() // cur is '('

	numArgs := 0
	var firstArgConst bool
	var firstArgVal Value
	if c.peekToken.Type != TokenRParen {
		c.nextToken()
		val, err := c.parseExpression(LOWEST)
		if err != nil {
			return compilationValue{}, err
		}
		firstArgConst = val.isConst
		firstArgVal = val.val
		numArgs++
		for c.peekToken.Type == TokenComma {
			if firstArgConst {
				c.emitPush(firstArgVal)
				firstArgConst = false
			}
			c.nextToken()
			c.nextToken()
			val, err = c.parseExpression(LOWEST)
			if err != nil {
				return compilationValue{}, err
			}
			if val.isConst {
				c.emitPush(val.val)
			}
			numArgs++
		}
	}
	if c.peekToken.Type != TokenRParen {
		return compilationValue{}, fmt.Errorf("expected ')', got %s", c.peekToken.Type)
	}
	c.nextToken()

	switch method {
	case "get":
		if numArgs != 1 {
			return compilationValue{}, fmt.Errorf("get expects 1 argument")
		}
		if firstArgConst && firstArgVal.Type == ValString {
			c.emit(NeoOpMapGetConst, c.addConstant(firstArgVal))
		} else {
			if firstArgConst {
				c.emitPush(firstArgVal)
			}
			c.emit(NeoOpMapGet, 0)
		}
	case "set":
		if firstArgConst {
			c.emitPush(firstArgVal)
		}
		if numArgs != 2 {
			return compilationValue{}, fmt.Errorf("set expects 2 arguments")
		}
		c.emit(NeoOpMapSet, 0)
	case "has":
		if firstArgConst {
			c.emitPush(firstArgVal)
			firstArgConst = false
		}
		if numArgs != 1 {
			return compilationValue{}, fmt.Errorf("has expects 1 argument")
		}
		c.emit(NeoOpMapHas, 0)
	case "del":
		if firstArgConst {
			c.emitPush(firstArgVal)
			firstArgConst = false
		}
		if numArgs != 1 {
			return compilationValue{}, fmt.Errorf("del expects 1 argument")
		}
		c.emit(NeoOpMapDel, 0)
	default:
		return compilationValue{}, fmt.Errorf("unknown map method: %s", method)
	}

	return compilationValue{isConst: false}, nil
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
	var val Value
	if !neoContainsDot(c.curToken.Literal) {
		v, err := strconv.ParseInt(c.curToken.Literal, 0, 64)
		if err != nil {
			// Fallback to float if it's too big for int64?
			// But usually we want exact int64.
			v_f, err_f := strconv.ParseFloat(c.curToken.Literal, 64)
			if err_f != nil {
				return compilationValue{}, err_f
			}
			val = Value{Type: ValFloat, Num: math.Float64bits(v_f)}
		} else {
			val = Value{Type: ValInt, Num: uint64(v)}
		}
	} else {
		v, err := strconv.ParseFloat(c.curToken.Literal, 64)
		if err != nil {
			return compilationValue{}, err
		}
		val = Value{Type: ValFloat, Num: math.Float64bits(v)}
	}
	return compilationValue{isConst: true, val: val}, nil
}

func (c *NeoCompiler) parseStringLiteral() (compilationValue, error) {
	return compilationValue{isConst: true, val: Value{Type: ValString, Str: c.curToken.Literal}, isString: true}, nil
}

func (c *NeoCompiler) parseBooleanLiteral() (compilationValue, error) {
	val := uint64(0)
	if c.curToken.Type == TokenTrue {
		val = 1
	}
	return compilationValue{isConst: true, val: Value{Type: ValBool, Num: val}}, nil
}

func (c *NeoCompiler) parsePrefixExpression() (compilationValue, error) {
	op := c.curToken.Literal
	c.nextToken()
	right, err := c.parseExpression(PREFIX)
	if err != nil {
		return compilationValue{}, err
	}

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
	if err != nil {
		return compilationValue{}, err
	}
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
		if err != nil {
			return compilationValue{}, err
		}
		if left.isConst && right.isConst {
			res, ok := c.foldInfix(left.val, right.val, op)
			if ok {
				return compilationValue{isConst: true, val: res, isString: true}, nil
			}
		}
		if left.isConst {
			c.emitPush(left.val)
		}
		if right.isConst {
			c.emitPush(right.val)
		}
		if canFuse {
			c.emit(NeoOpConcat, nArgs+1)
		} else {
			c.emit(NeoOpConcat, 2)
		}
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
		if err != nil {
			return compilationValue{}, err
		}
		if right.isConst {
			c.emitPush(right.val)
		}
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
		if err != nil {
			return compilationValue{}, err
		}
		if right.isConst {
			c.emitPush(right.val)
		}
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
	if err != nil {
		return compilationValue{}, err
	}
	if left.isConst && right.isConst {
		res, ok := c.foldInfix(left.val, right.val, op)
		if ok {
			return compilationValue{isConst: true, val: res}, nil
		}
	}
	if (left.isString || right.isString) && op == "+" {
		if left.isConst {
			c.emitPush(left.val)
		}
		if right.isConst {
			c.emitPush(right.val)
		}
		c.emit(NeoOpConcat, 2)
		return compilationValue{isConst: false, isString: true}, nil
	}

	// Algebraic Simplifications
	if op == "+" {
		if left.isConst && neoIsZero(left.val) {
			return right, nil
		}
		if right.isConst && neoIsZero(right.val) {
			return left, nil
		}
	} else if op == "-" {
		if right.isConst && neoIsZero(right.val) {
			return left, nil
		}
	} else if op == "*" {
		if left.isConst {
			if neoIsZero(left.val) {
				return left, nil
			}
			if neoIsOne(left.val) {
				return right, nil
			}
		}
		if right.isConst {
			if neoIsZero(right.val) {
				return right, nil
			}
			if neoIsOne(right.val) {
				return left, nil
			}
		}
	} else if op == "/" {
		if right.isConst && neoIsOne(right.val) {
			return left, nil
		}
	}

	if left.isConst {
		c.emitPush(left.val)
	}
	if right.isConst {
		c.emitPush(right.val)
	}

	switch op {
	case "+":
		if left.isString || right.isString {
			lastIdx := len(c.instructions) - 1
			if lastIdx >= 0 && c.instructions[lastIdx].Op == NeoOpConcat {
				nArgs := c.instructions[lastIdx].Arg
				c.instructions = c.instructions[:lastIdx]
				if right.isConst {
					c.emitPush(right.val)
				}
				c.emit(NeoOpConcat, nArgs+1)
				return compilationValue{isConst: false, isString: true}, nil
			}
			if left.isConst {
				c.emitPush(left.val)
			}
			if right.isConst {
				c.emitPush(right.val)
			}
			c.emit(NeoOpConcat, 2)
			return compilationValue{isConst: false, isString: true}, nil
		}
		c.emit(NeoOpAdd, 0)
	case "-":
		c.emit(NeoOpSub, 0)
	case "*":
		c.emit(NeoOpMul, 0)
	case "/":
		c.emit(NeoOpDiv, 0)
	case "%":
		c.emit(NeoOpMod, 0)
	case "==":
		c.emit(NeoOpEqual, 0)
	case ">":
		c.emit(NeoOpGreater, 0)
	case "<":
		c.emit(NeoOpLess, 0)
	case ">=":
		c.emit(NeoOpGreaterEqual, 0)
	case "<=":
		c.emit(NeoOpLessEqual, 0)
	}
	return compilationValue{isConst: false}, nil
}

func (c *NeoCompiler) foldInfix(l, r Value, op string) (Value, bool) {
	switch op {
	case "+":
		if l.Type == ValInt && r.Type == ValInt {
			return Value{Type: ValInt, Num: l.Num + r.Num}, true
		}
		if l.Type == ValString && r.Type == ValString {
			return Value{Type: ValString, Str: l.Str + r.Str}, true
		}
		if (l.Type == ValInt || l.Type == ValFloat) && (r.Type == ValInt || r.Type == ValFloat) {
			lf, _ := valToFloat64(l)
			rf, _ := valToFloat64(r)
			return Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}, true
		}
	case "-":
		if l.Type == ValInt && r.Type == ValInt {
			return Value{Type: ValInt, Num: l.Num - r.Num}, true
		}
		if (l.Type == ValInt || l.Type == ValFloat) && (r.Type == ValInt || r.Type == ValFloat) {
			lf, _ := valToFloat64(l)
			rf, _ := valToFloat64(r)
			return Value{Type: ValFloat, Num: math.Float64bits(lf - rf)}, true
		}
	case "*":
		if l.Type == ValInt && r.Type == ValInt {
			return Value{Type: ValInt, Num: l.Num * r.Num}, true
		}
		if (l.Type == ValInt || l.Type == ValFloat) && (r.Type == ValInt || r.Type == ValFloat) {
			lf, _ := valToFloat64(l)
			rf, _ := valToFloat64(r)
			return Value{Type: ValFloat, Num: math.Float64bits(lf * rf)}, true
		}
	case "/":
		if (r.Type == ValInt && r.Num == 0) || (r.Type == ValFloat && math.Float64frombits(r.Num) == 0) {
			return Value{}, false
		}
		if l.Type == ValInt && r.Type == ValInt {
			return Value{Type: ValInt, Num: l.Num / r.Num}, true
		}
		if (l.Type == ValInt || l.Type == ValFloat) && (r.Type == ValInt || r.Type == ValFloat) {
			lf, _ := valToFloat64(l)
			rf, _ := valToFloat64(r)
			return Value{Type: ValFloat, Num: math.Float64bits(lf / rf)}, true
		}
	case "%":
		if r.Type == ValInt && r.Num == 0 {
			return Value{}, false
		}
		if l.Type == ValInt && r.Type == ValInt {
			return Value{Type: ValInt, Num: l.Num % r.Num}, true
		}
	case "==":
		return Value{Type: ValBool, Num: boolToUint64(c.compare(l, r) == 0)}, true
	case ">":
		return Value{Type: ValBool, Num: boolToUint64(c.compare(l, r) > 0)}, true
	case "<":
		return Value{Type: ValBool, Num: boolToUint64(c.compare(l, r) < 0)}, true
	case ">=":
		return Value{Type: ValBool, Num: boolToUint64(c.compare(l, r) >= 0)}, true
	case "<=":
		return Value{Type: ValBool, Num: boolToUint64(c.compare(l, r) <= 0)}, true
	}
	return Value{}, false
}

func (c *NeoCompiler) compare(l, r Value) int {
	if l.Type == r.Type {
		switch l.Type {
		case ValInt:
			if int64(l.Num) < int64(r.Num) {
				return -1
			}
			if int64(l.Num) > int64(r.Num) {
				return 1
			}
			return 0
		case ValFloat:
			lf := math.Float64frombits(l.Num)
			rf := math.Float64frombits(r.Num)
			if lf < rf {
				return -1
			}
			if lf > rf {
				return 1
			}
			return 0
		case ValString:
			if l.Str < r.Str {
				return -1
			}
			if l.Str > r.Str {
				return 1
			}
			return 0
		case ValBool:
			if l.Num < r.Num {
				return -1
			}
			if l.Num > r.Num {
				return 1
			}
			return 0
		case ValNil:
			return 0
		}
	}
	lf, okL := valToFloat64(l)
	rf, okR := valToFloat64(r)
	if okL && okR {
		if lf < rf {
			return -1
		}
		if lf > rf {
			return 1
		}
		return 0
	}
	return 0
}

func (c *NeoCompiler) parseAssignExpression(left compilationValue) (compilationValue, error) {
	if left.isConst {
		return compilationValue{}, fmt.Errorf("left side of assignment must be an identifier")
	}
	if c.discard {
		c.nextToken()
		_, err := c.parseExpression(ASSIGN)
		return compilationValue{isConst: false}, err
	}
	lastInst := c.instructions[len(c.instructions)-1]
	if lastInst.Op != NeoOpGetGlobal {
		return compilationValue{}, fmt.Errorf("left side of assignment must be an identifier")
	}
	identIdx := lastInst.Arg
	c.instructions = c.instructions[:len(c.instructions)-1]
	c.nextToken()
	val, err := c.parseExpression(ASSIGN)
	if err != nil {
		return compilationValue{}, err
	}
	if val.isConst {
		c.emitPush(val.val)
	}
	c.emit(NeoOpSetGlobal, identIdx)
	return compilationValue{isConst: false}, nil
}

func (c *NeoCompiler) parseCallExpression(left compilationValue) (compilationValue, error) {
	if left.isConst {
		return compilationValue{}, fmt.Errorf("function call must be on an identifier")
	}
	if c.discard {
		numArgs := 0
		if c.peekToken.Type != TokenRParen {
			c.nextToken()
			c.parseExpression(LOWEST)
			numArgs++
			for c.peekToken.Type == TokenComma {
				c.nextToken()
				c.nextToken()
				c.parseExpression(LOWEST)
				numArgs++
			}
		}
		if c.peekToken.Type != TokenRParen {
			return compilationValue{}, fmt.Errorf("expected ), got %s", c.peekToken.Type)
		}
		c.nextToken()
		return compilationValue{isConst: false}, nil
	}
	lastInst := c.instructions[len(c.instructions)-1]
	if lastInst.Op != NeoOpGetGlobal {
		return compilationValue{}, fmt.Errorf("function call must be on an identifier")
	}
	funcNameIdx := lastInst.Arg
	c.instructions = c.instructions[:len(c.instructions)-1]
	numArgs := 0
	if c.peekToken.Type != TokenRParen {
		c.nextToken()
		val, err := c.parseExpression(LOWEST)
		if err != nil {
			return compilationValue{}, err
		}
		if val.isConst {
			c.emitPush(val.val)
		}
		numArgs++
		for c.peekToken.Type == TokenComma {
			c.nextToken()
			c.nextToken()
			val, err = c.parseExpression(LOWEST)
			if err != nil {
				return compilationValue{}, err
			}
			if val.isConst {
				c.emitPush(val.val)
			}
			numArgs++
		}
	}
	if c.peekToken.Type != TokenRParen {
		return compilationValue{}, fmt.Errorf("expected ), got %s", c.peekToken.Type)
	}
	c.nextToken()
	funcName := c.constants[funcNameIdx].Str
	if funcName == "concat" {
		if numArgs == 2 {
			c.emit(NeoOpConcat2, 0)
		} else {
			c.emit(NeoOpConcat, int32(numArgs))
		}
	} else {
		c.emit(NeoOpCall, funcNameIdx|int32(numArgs<<16))
	}
	return compilationValue{isConst: false}, nil
}

func (c *NeoCompiler) parseIfExpression() (compilationValue, error) {
	c.nextToken()
	cond, err := c.parseExpression(LOWEST)
	if err != nil {
		return compilationValue{}, err
	}
	if c.peekToken.Type == TokenThen {
		c.nextToken()
		c.nextToken()
		if cond.isConst {
			if isValTruthy(cond.val) {
				return c.parseExpression(LOWEST)
			} else {
				oldDiscard := c.discard
				c.discard = true
				_, err = c.parseExpression(LOWEST)
				c.discard = oldDiscard
				return compilationValue{isConst: true, val: Value{Type: ValNil}}, err
			}
		}
		jumpFalse := c.emit(NeoOpJumpIfFalse, 0)
		cons, err := c.parseExpression(LOWEST)
		if err != nil {
			return compilationValue{}, err
		}
		if cons.isConst {
			c.emitPush(cons.val)
		}
		jumpEnd := c.emit(NeoOpJump, 0)
		c.patch(jumpFalse, int32(len(c.instructions)))
		c.emitPush(Value{Type: ValNil})
		c.patch(jumpEnd, int32(len(c.instructions)))
		return compilationValue{isConst: false}, nil
	}
	if c.peekToken.Type == TokenIs {
		var jumpEndTargets []int
		for {
			if c.peekToken.Type != TokenIs {
				return compilationValue{}, fmt.Errorf("expected is after if condition, got %s", c.peekToken.Type)
			}
			c.nextToken()
			c.nextToken()
			var jumpFalse int
			var tookBranch bool
			if cond.isConst {
				if isValTruthy(cond.val) {
					cons, err := c.parseExpression(LOWEST)
					if err != nil {
						return compilationValue{}, err
					}
					if cons.isConst {
						c.emitPush(cons.val)
					}
					tookBranch = true
				} else {
					oldDiscard := c.discard
					c.discard = true
					c.parseExpression(LOWEST)
					c.discard = oldDiscard
				}
			} else {
				jumpFalse = c.emit(NeoOpJumpIfFalse, 0)
				cons, err := c.parseExpression(LOWEST)
				if err != nil {
					return compilationValue{}, err
				}
				if cons.isConst {
					c.emitPush(cons.val)
				}
				jumpEndTargets = append(jumpEndTargets, c.emit(NeoOpJump, 0))
				c.patch(jumpFalse, int32(len(c.instructions)))
			}
			if tookBranch {
				for c.peekToken.Type == TokenElse {
					c.nextToken()
					if c.peekToken.Type == TokenIf {
						c.nextToken()
						c.nextToken()
						oldDiscard := c.discard
						c.discard = true
						c.parseExpression(LOWEST)
						if c.peekToken.Type == TokenIs {
							c.nextToken()
							c.nextToken()
							c.parseExpression(LOWEST)
						}
						c.discard = oldDiscard
					} else if c.peekToken.Type == TokenIs {
						c.nextToken()
						c.nextToken()
						oldDiscard := c.discard
						c.discard = true
						c.parseExpression(LOWEST)
						c.discard = oldDiscard
						break
					}
				}
				break
			}
			if c.peekToken.Type != TokenElse {
				c.emitPush(Value{Type: ValNil})
				break
			}
			c.nextToken()
			if c.peekToken.Type == TokenIf {
				c.nextToken()
				c.nextToken()
				cond, err = c.parseExpression(LOWEST)
				if err != nil {
					return compilationValue{}, err
				}
				continue
			}
			if c.peekToken.Type == TokenIs {
				c.nextToken()
				c.nextToken()
				alt, err := c.parseExpression(LOWEST)
				if err != nil {
					return compilationValue{}, err
				}
				if alt.isConst {
					c.emitPush(alt.val)
				}
				break
			}
			return compilationValue{}, fmt.Errorf("expected if or is after else, got %s", c.peekToken.Type)
		}
		for _, target := range jumpEndTargets {
			c.patch(target, int32(len(c.instructions)))
		}
		return compilationValue{isConst: false}, nil
	}
	// Simple if <cond> -> returns bool
	return compilationValue{isConst: false}, nil
}

func (c *NeoCompiler) emit(op NeoOpCode, arg int32) int {
	if c.discard {
		return -1
	}

	n := len(c.instructions)

	// 3rd-order patterns (GC, CG, GG) and 2nd-order (C)
	// We skip Jump patterns here because they require knowing the target range,
	// which is not known during emit (patched later). Jumps are handled in peephole.
	switch op {
	case NeoOpAdd, NeoOpSub, NeoOpMul, NeoOpDiv, NeoOpEqual, NeoOpGreater, NeoOpLess, NeoOpConcat2:
		if n >= 2 {
			i1 := c.instructions[n-2]
			i2 := c.instructions[n-1]
			if i1.Op == NeoOpGetGlobal && i2.Op == NeoOpPush {
				if i1.Arg < 65536 && i2.Arg < 65536 {
					newOp := NeoOpCode(0)
					switch op {
					case NeoOpAdd:
						newOp = NeoOpAddGC
					case NeoOpSub:
						newOp = NeoOpSubGC
					case NeoOpMul:
						newOp = NeoOpMulGC
					case NeoOpDiv:
						newOp = NeoOpDivGC
					case NeoOpEqual:
						newOp = NeoOpEqualGlobalConst
					case NeoOpGreater:
						newOp = NeoOpGreaterGlobalConst
					case NeoOpLess:
						newOp = NeoOpLessGlobalConst
					case NeoOpConcat2:
						newOp = NeoOpConcatGC
					}
					if newOp != 0 {
						c.instructions = c.instructions[:n-2]
						return c.emit(newOp, (i1.Arg<<16)|i2.Arg)
					}
				}
			}
			if i1.Op == NeoOpPush && i2.Op == NeoOpGetGlobal {
				if i1.Arg < 65536 && i2.Arg < 65536 {
					newOp := NeoOpCode(0)
					switch op {
					case NeoOpAdd:
						newOp = NeoOpAddConstGlobal
					case NeoOpSub:
						newOp = NeoOpSubCG
					case NeoOpMul:
						newOp = NeoOpMulCG
					case NeoOpDiv:
						newOp = NeoOpDivCG
					case NeoOpEqual:
						newOp = NeoOpEqualGlobalConst
					case NeoOpConcat2:
						newOp = NeoOpConcatCG
					}
					if newOp != 0 {
						c.instructions = c.instructions[:n-2]
						return c.emit(newOp, (i2.Arg<<16)|i1.Arg)
					}
				}
			}
			if i1.Op == NeoOpGetGlobal && i2.Op == NeoOpGetGlobal {
				if i1.Arg < 65536 && i2.Arg < 65536 {
					newOp := NeoOpCode(0)
					switch op {
					case NeoOpAdd:
						newOp = NeoOpAddGlobalGlobal
					case NeoOpSub:
						newOp = NeoOpSubGlobalGlobal
					case NeoOpMul:
						newOp = NeoOpMulGlobalGlobal
					}
					if newOp != 0 {
						c.instructions = c.instructions[:n-2]
						return c.emit(newOp, (i1.Arg<<16)|i2.Arg)
					}
				}
			}
			// Constant folding in emit (fallback)
			if i1.Op == NeoOpPush && i2.Op == NeoOpPush {
				c1 := c.constants[i1.Arg]
				c2 := c.constants[i2.Arg]
				res, ok := c.foldInfix(c1, c2, opToString(op))
				if ok {
					c.instructions = c.instructions[:n-2]
					return c.emitPush(res)
				}
			}
		}
		// 2nd-order (OpC)
		if n >= 1 {
			prev := c.instructions[n-1]
			if prev.Op == NeoOpPush {
				newOp := NeoOpCode(0)
				switch op {
				case NeoOpAdd:
					newOp = NeoOpAddC
				case NeoOpSub:
					newOp = NeoOpSubC
				case NeoOpMul:
					newOp = NeoOpMulC
				case NeoOpDiv:
					newOp = NeoOpDivC
				case NeoOpEqual:
					newOp = NeoOpEqualC
				case NeoOpGreater:
					newOp = NeoOpGreaterC
				case NeoOpLess:
					newOp = NeoOpLessC
				}
				if newOp != 0 {
					c.instructions[n-1] = neoInstruction{Op: newOp, Arg: prev.Arg}
					return n - 1
				}
			}
		}
	}

	// Final check for ADD following CONCAT
	if op == NeoOpAdd && n > 0 && c.instructions[n-1].Op == NeoOpConcat {
		c.instructions[n-1].Arg++
		return n - 1
	}

	c.instructions = append(c.instructions, neoInstruction{Op: op, Arg: arg})
	return len(c.instructions) - 1
}

func opToString(op NeoOpCode) string {
	switch op {
	case NeoOpAdd:
		return "+"
	case NeoOpSub:
		return "-"
	case NeoOpMul:
		return "*"
	case NeoOpDiv:
		return "/"
	case NeoOpEqual:
		return "=="
	case NeoOpGreater:
		return ">"
	case NeoOpLess:
		return "<"
	case NeoOpGreaterEqual:
		return ">="
	case NeoOpLessEqual:
		return "<="
	case NeoOpMod:
		return "%"
	}
	return ""
}

func (c *NeoCompiler) emitPush(v Value) int { return c.emit(NeoOpPush, c.addConstant(v)) }

func (c *NeoCompiler) patch(pos int, arg int32) { c.instructions[pos].Arg = arg }

func neoIsZero(v Value) bool {
	switch v.Type {
	case ValInt:
		return v.Num == 0
	case ValFloat:
		return math.Float64frombits(v.Num) == 0
	}
	return false
}

func neoIsOne(v Value) bool {
	switch v.Type {
	case ValInt:
		return v.Num == 1
	case ValFloat:
		return math.Float64frombits(v.Num) == 1
	}
	return false
}

func (c *NeoCompiler) addConstant(v Value) int32 {
	// Optimization: check first few constants linearly to avoid map overhead for tiny expressions
	n := len(c.constants)
	if n > 0 {
		start := 0
		if n > 16 {
			start = n - 16
		}
		for i := n - 1; i >= start; i-- {
			if c.constants[i].Type == v.Type {
				if v.Type == ValString {
					if c.constants[i].Str == v.Str {
						return int32(i)
					}
				} else if c.constants[i].Num == v.Num {
					return int32(i)
				}
			}
		}
	}

	switch v.Type {
	case ValInt:
		if idx, ok := c.constMapInt[int64(v.Num)]; ok {
			return idx
		}
	case ValFloat:
		if idx, ok := c.constMapFloat[v.Num]; ok {
			return idx
		}
	case ValBool:
		if idx, ok := c.constMapBool[v.Num != 0]; ok {
			return idx
		}
	case ValString:
		if idx, ok := c.constMapString[v.Str]; ok {
			return idx
		}
	case ValNil:
		if idx, ok := c.constMapOther[nil]; ok {
			return idx
		}
	case ValMap:
		if idx, ok := c.constMapOther[v.Ptr]; ok {
			return idx
		}
	}

	idx := int32(len(c.constants))
	c.constants = append(c.constants, v)

	switch v.Type {
	case ValInt:
		c.constMapInt[int64(v.Num)] = idx
	case ValFloat:
		c.constMapFloat[v.Num] = idx
	case ValBool:
		c.constMapBool[v.Num != 0] = idx
	case ValString:
		c.constMapString[v.Str] = idx
	case ValNil, ValMap:
		key := v.Ptr
		if v.Type == ValNil {
			key = nil
		}
		c.constMapOther[key] = idx
	}
	return idx
}

var oldToNewPool = sync.Pool{
	New: func() any { return make([]int, 0, 128) },
}
var newInstsPool = sync.Pool{
	New: func() any { return make([]neoInstruction, 0, 128) },
}

func (c *NeoCompiler) peephole() {
	if len(c.instructions) < 2 {
		return
	}

	// Single pass for jump fusions, as others are handled online in emit.
	// This reduces complexity and allocations while maintaining safety.
	newInsts := newInstsPool.Get().([]neoInstruction)[:0]
	oldToNew := oldToNewPool.Get().([]int)[:0]
	if cap(oldToNew) < len(c.instructions)+1 {
		oldToNew = make([]int, 0, len(c.instructions)+1)
	}

	for i := 0; i < len(c.instructions); i++ {
		oldToNew = append(oldToNew, len(newInsts))
		inst := c.instructions[i]

		// Catch remaining jump fusions (FCG, GGJ)
		if i+1 < len(c.instructions) {
			next := c.instructions[i+1]
			if next.Op == NeoOpJumpIfFalse {
				jTarget := next.Arg
				if jTarget < 4096 {
					switch inst.Op {
					case NeoOpEqualGlobalConst, NeoOpGreaterGlobalConst, NeoOpLessGlobalConst:
						gIdx := inst.Arg >> 16
						cIdx := inst.Arg & 0xFFFF
						if gIdx < 1024 && cIdx < 1024 {
							newOp := NeoOpCode(0)
							switch inst.Op {
							case NeoOpEqualGlobalConst:
								newOp = NeoOpFusedCompareGlobalConstJumpIfFalse
							case NeoOpGreaterGlobalConst:
								newOp = NeoOpFusedGreaterGlobalConstJumpIfFalse
							case NeoOpLessGlobalConst:
								newOp = NeoOpFusedLessGlobalConstJumpIfFalse
							}
							newInsts = append(newInsts, neoInstruction{Op: newOp, Arg: (gIdx << 22) | (cIdx << 12) | jTarget})
							oldToNew = append(oldToNew, len(newInsts)-1)
							i++
							continue
						}
					case NeoOpGetGlobal:
						if inst.Arg < 65536 && jTarget < 65536 {
							newInsts = append(newInsts, neoInstruction{Op: NeoOpGetGlobalJumpIfFalse, Arg: (inst.Arg << 16) | jTarget})
							oldToNew = append(oldToNew, len(newInsts)-1)
							i++
							continue
						}
					}
				}
			} else if next.Op == NeoOpJumpIfTrue {
				jTarget := next.Arg
				if inst.Op == NeoOpGetGlobal && inst.Arg < 65536 && jTarget < 65536 {
					newInsts = append(newInsts, neoInstruction{Op: NeoOpGetGlobalJumpIfTrue, Arg: (inst.Arg << 16) | jTarget})
					oldToNew = append(oldToNew, len(newInsts)-1)
					i++
					continue
				}
			}
		}
		newInsts = append(newInsts, inst)
	}
	oldToNew = append(oldToNew, len(newInsts))

	// Update jump targets
	for i := range newInsts {
		switch newInsts[i].Op {
		case NeoOpJump, NeoOpJumpIfFalse, NeoOpJumpIfTrue:
			newInsts[i].Arg = int32(oldToNew[newInsts[i].Arg])
		case NeoOpFusedCompareGlobalConstJumpIfFalse, NeoOpFusedGreaterGlobalConstJumpIfFalse, NeoOpFusedLessGlobalConstJumpIfFalse:
			gIdx := (newInsts[i].Arg >> 22) & 0x3FF
			cIdx := (newInsts[i].Arg >> 12) & 0x3FF
			jTarget := newInsts[i].Arg & 0xFFF
			newInsts[i].Arg = (gIdx << 22) | (cIdx << 12) | int32(oldToNew[jTarget])
		case NeoOpGetGlobalJumpIfFalse, NeoOpGetGlobalJumpIfTrue:
			gIdx := newInsts[i].Arg >> 16
			jTarget := newInsts[i].Arg & 0xFFFF
			newInsts[i].Arg = (gIdx << 16) | int32(oldToNew[jTarget])
		}
	}

	c.instructions = make([]neoInstruction, len(newInsts))
	copy(c.instructions, newInsts)
	newInstsPool.Put(newInsts)
	oldToNewPool.Put(oldToNew)
}
