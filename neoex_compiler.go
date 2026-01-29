// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"math"
	"strconv"
)

type compilationValue struct {
	isConst bool
	val     Value
}

type NeoExCompiler struct {
	lexer     *Lexer
	curToken  Token
	peekToken Token

	instructions []neoInstruction
	constants    []Value
	constMap     map[any]int32

	errors []string
}

func NewNeoExCompiler(input string) *NeoExCompiler {
	l := NewLexer(input)
	c := &NeoExCompiler{
		lexer:    l,
		constMap: make(map[any]int32),
	}
	c.nextToken()
	c.nextToken()
	return c
}

func (c *NeoExCompiler) nextToken() {
	c.curToken = c.peekToken
	c.peekToken = c.lexer.NextToken()
}

func (c *NeoExCompiler) Compile() (*NeoBytecode, error) {
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

func (c *NeoExCompiler) parseExpression(precedence int) (compilationValue, error) {
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

func (c *NeoExCompiler) curPrecedence() int {
	return getPrecedence(c.curToken.Type)
}

func (c *NeoExCompiler) peekPrecedence() int {
	return getPrecedence(c.peekToken.Type)
}

func (c *NeoExCompiler) getPrefixFn(t TokenType) func() (compilationValue, error) {
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

func (c *NeoExCompiler) getInfixFn(t TokenType) func(compilationValue) (compilationValue, error) {
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

func (c *NeoExCompiler) parseIdentifier() (compilationValue, error) {
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

func (c *NeoExCompiler) parseNumberLiteral() (compilationValue, error) {
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

func (c *NeoExCompiler) parseStringLiteral() (compilationValue, error) {
	return compilationValue{isConst: true, val: Value{Type: ValString, Str: c.curToken.Literal}}, nil
}

func (c *NeoExCompiler) parseBooleanLiteral() (compilationValue, error) {
	val := uint64(0)
	if c.curToken.Type == TokenTrue { val = 1 }
	return compilationValue{isConst: true, val: Value{Type: ValBool, Num: val}}, nil
}

func (c *NeoExCompiler) parsePrefixExpression() (compilationValue, error) {
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

func (c *NeoExCompiler) parseGroupedExpression() (compilationValue, error) {
	c.nextToken()
	val, err := c.parseExpression(LOWEST)
	if err != nil { return compilationValue{}, err }
	if c.peekToken.Type != TokenRParen {
		return compilationValue{}, fmt.Errorf("expected ), got %s", c.peekToken.Type)
	}
	c.nextToken()
	return val, nil
}

func (c *NeoExCompiler) parseInfixExpression(left compilationValue) (compilationValue, error) {
	op := c.curToken.Literal
	precedence := c.curPrecedence()

	if op == "&&" {
		if left.isConst {
			if !isValTruthy(left.val) {
				return left, nil
			}
			c.nextToken()
			return c.parseExpression(precedence)
		}

		jumpFalse := c.emit(NeoOpJumpIfFalse, 0)
		c.nextToken()
		right, err := c.parseExpression(precedence)
		if err != nil { return compilationValue{}, err }

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
			if isValTruthy(left.val) {
				return left, nil
			}
			c.nextToken()
			return c.parseExpression(precedence)
		}

		jumpTrue := c.emit(NeoOpJumpIfTrue, 0)
		c.nextToken()
		right, err := c.parseExpression(precedence)
		if err != nil { return compilationValue{}, err }

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

	c.nextToken()
	right, err := c.parseExpression(precedence)
	if err != nil { return compilationValue{}, err }

	if left.isConst && right.isConst {
		res, ok := c.foldInfix(left.val, right.val, op)
		if ok {
			return compilationValue{isConst: true, val: res}, nil
		}
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
	case "+": c.emit(NeoOpAdd, 0)
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

func (c *NeoExCompiler) foldInfix(l, r Value, op string) (Value, bool) {
	switch op {
	case "+":
		if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num + r.Num}, true }
		if l.Type == ValString && r.Type == ValString { return Value{Type: ValString, Str: l.Str + r.Str}, true }
		lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
		if okL && okR { return Value{Type: ValFloat, Num: math.Float64bits(lf + rf)}, true }
	case "-":
		if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num - r.Num}, true }
		lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
		if okL && okR { return Value{Type: ValFloat, Num: math.Float64bits(lf - rf)}, true }
	case "*":
		if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num * r.Num}, true }
		lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
		if okL && okR { return Value{Type: ValFloat, Num: math.Float64bits(lf * rf)}, true }
	case "/":
		if (r.Type == ValInt && r.Num == 0) || (r.Type == ValFloat && math.Float64frombits(r.Num) == 0) {
			c.errors = append(c.errors, "division by zero")
			return Value{}, false
		}
		if l.Type == ValInt && r.Type == ValInt { return Value{Type: ValInt, Num: l.Num / r.Num}, true }
		lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
		if okL && okR { return Value{Type: ValFloat, Num: math.Float64bits(lf / rf)}, true }
	case "==":
		res := false
		if l.Type == r.Type {
			switch l.Type {
			case ValInt, ValFloat, ValBool: res = l.Num == r.Num
			case ValString: res = l.Str == r.Str
			case ValNil: res = true
			}
		} else {
			lf, okL := valToFloat64(l); rf, okR := valToFloat64(r)
			if okL && okR { res = lf == rf }
		}
		return Value{Type: ValBool, Num: boolToUint64(res)}, true
	}
	return Value{}, false
}

func (c *NeoExCompiler) parseAssignExpression(left compilationValue) (compilationValue, error) {
	if left.isConst {
		return compilationValue{}, fmt.Errorf("left side of assignment must be an identifier")
	}
	lastInst := c.instructions[len(c.instructions)-1]
	if lastInst.Op != NeoOpGetGlobal {
		return compilationValue{}, fmt.Errorf("left side of assignment must be an identifier")
	}
	identIdx := lastInst.Arg
	c.instructions = c.instructions[:len(c.instructions)-1]

	c.nextToken()
	val, err := c.parseExpression(ASSIGN)
	if err != nil { return compilationValue{}, err }

	if val.isConst {
		c.emitPush(val.val)
	}

	c.emit(NeoOpSetGlobal, identIdx)
	return compilationValue{isConst: false}, nil
}

func (c *NeoExCompiler) parseCallExpression(left compilationValue) (compilationValue, error) {
	if left.isConst {
		return compilationValue{}, fmt.Errorf("function call must be on an identifier")
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
		if err != nil { return compilationValue{}, err }
		if val.isConst { c.emitPush(val.val) }
		numArgs++

		for c.peekToken.Type == TokenComma {
			c.nextToken()
			c.nextToken()
			val, err := c.parseExpression(LOWEST)
			if err != nil { return compilationValue{}, err }
			if val.isConst { c.emitPush(val.val) }
			numArgs++
		}
	}

	if c.peekToken.Type != TokenRParen {
		return compilationValue{}, fmt.Errorf("expected ), got %s", c.peekToken.Type)
	}
	c.nextToken()

	funcName := c.constants[funcNameIdx].Str
	if funcName == "concat" {
		c.emit(NeoOpConcat, int32(numArgs))
	} else {
		c.emit(NeoOpCall, funcNameIdx | int32(numArgs << 16))
	}
	return compilationValue{isConst: false}, nil
}

func (c *NeoExCompiler) parseIfExpression() (compilationValue, error) {
	c.nextToken()
	cond, err := c.parseExpression(LOWEST)
	if err != nil { return compilationValue{}, err }

	if c.peekToken.Type == TokenThen {
		c.nextToken()
		c.nextToken()

		if cond.isConst {
			if isValTruthy(cond.val) {
				return c.parseExpression(LOWEST)
			} else {
				_, err = c.parseExpression(LOWEST)
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
		first := true

		for {
			if !first {
				if c.curToken.Type == TokenIf {
					c.nextToken()
					cond, err = c.parseExpression(LOWEST)
					if err != nil { return compilationValue{}, err }
				} else if c.curToken.Type == TokenIs {
					cond = compilationValue{isConst: true, val: Value{Type: ValBool, Num: 1}}
				}
			}
			first = false

			if c.peekToken.Type != TokenIs {
				return compilationValue{}, fmt.Errorf("expected is after if condition")
			}
			c.nextToken()
			c.nextToken()

			var jumpFalse int
			if cond.isConst {
				if !isValTruthy(cond.val) {
					_, err = c.parseExpression(LOWEST)
					if err != nil { return compilationValue{}, err }
					goto handleElse
				}
				cons, err := c.parseExpression(LOWEST)
				if err != nil { return compilationValue{}, err }
				if cons.isConst { c.emitPush(cons.val) }

				for c.peekToken.Type == TokenElse {
					c.nextToken()
					if c.peekToken.Type == TokenIf {
						c.nextToken(); c.nextToken(); c.parseExpression(LOWEST)
						c.nextToken(); c.nextToken(); c.parseExpression(LOWEST)
					} else if c.peekToken.Type == TokenIs {
						c.nextToken(); c.nextToken(); c.parseExpression(LOWEST)
					}
				}
				break
			} else {
				jumpFalse = c.emit(NeoOpJumpIfFalse, 0)
				cons, err := c.parseExpression(LOWEST)
				if err != nil { return compilationValue{}, err }
				if cons.isConst { c.emitPush(cons.val) }
				jumpEndTargets = append(jumpEndTargets, c.emit(NeoOpJump, 0))
				c.patch(jumpFalse, int32(len(c.instructions)))
			}

		handleElse:
			if c.peekToken.Type != TokenElse {
				c.emitPush(Value{Type: ValNil})
				break
			}
			c.nextToken()

			if c.peekToken.Type == TokenIf {
				c.nextToken()
				c.nextToken()
				continue
			}

			if c.peekToken.Type == TokenIs {
				c.nextToken()
				c.nextToken()
				alt, err := c.parseExpression(LOWEST)
				if err != nil { return compilationValue{}, err }
				if alt.isConst { c.emitPush(alt.val) }
				break
			}

			return compilationValue{}, fmt.Errorf("expected if or is after else")
		}

		for _, target := range jumpEndTargets {
			c.patch(target, int32(len(c.instructions)))
		}
		return compilationValue{isConst: false}, nil
	}

	return compilationValue{}, fmt.Errorf("expected then or is after if condition")
}

func (c *NeoExCompiler) emit(op NeoOpCode, arg int32) int {
	c.instructions = append(c.instructions, neoInstruction{Op: op, Arg: arg})
	return len(c.instructions) - 1
}

func (c *NeoExCompiler) emitPush(v Value) int {
	return c.emit(NeoOpPush, c.addConstant(v))
}

func (c *NeoExCompiler) patch(pos int, arg int32) {
	c.instructions[pos].Arg = arg
}

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

func (c *NeoExCompiler) addConstant(v Value) int32 {
	var key any
	switch v.Type {
	case ValInt: key = int64(v.Num)
	case ValFloat: key = math.Float64frombits(v.Num)
	case ValBool: key = v.Num != 0
	case ValString: key = v.Str
	case ValNil: key = nil
	}
	if idx, ok := c.constMap[key]; ok {
		return idx
	}
	idx := int32(len(c.constants))
	c.constants = append(c.constants, v)
	c.constMap[key] = idx
	return idx
}

func (c *NeoExCompiler) peephole() {
	if len(c.instructions) < 2 {
		return
	}

	newInsts := make([]neoInstruction, 0, len(c.instructions))
	oldToNew := make([]int, len(c.instructions)+1)

	for i := 0; i < len(c.instructions); i++ {
		oldToNew[i] = len(newInsts)
		inst := c.instructions[i]

		if i+3 < len(c.instructions) &&
			inst.Op == NeoOpGetGlobal &&
			c.instructions[i+1].Op == NeoOpPush &&
			(c.instructions[i+2].Op == NeoOpEqual || c.instructions[i+2].Op == NeoOpGreater || c.instructions[i+2].Op == NeoOpLess) &&
			c.instructions[i+3].Op == NeoOpJumpIfFalse {

			gIdx := inst.Arg
			cIdx := c.instructions[i+1].Arg
			jTarget := c.instructions[i+3].Arg

			if gIdx < 1024 && cIdx < 1024 && jTarget < 4096 && c.instructions[i+2].Op == NeoOpEqual {
				fusedArg := (gIdx << 22) | (cIdx << 12) | jTarget
				newInsts = append(newInsts, neoInstruction{Op: NeoOpFusedCompareGlobalConstJumpIfFalse, Arg: fusedArg})
				oldToNew[i+1] = len(newInsts) - 1
				oldToNew[i+2] = len(newInsts) - 1
				oldToNew[i+3] = len(newInsts) - 1
				i += 3
				continue
			}
		}

		if i+2 < len(c.instructions) &&
			inst.Op == NeoOpGetGlobal &&
			c.instructions[i+1].Op == NeoOpPush {

			gIdx := inst.Arg
			cIdx := c.instructions[i+1].Arg

			if gIdx < 65536 && cIdx < 65536 {
				if c.instructions[i+2].Op == NeoOpAdd {
					newInsts = append(newInsts, neoInstruction{Op: NeoOpAddGlobal, Arg: (cIdx << 16) | gIdx})
					oldToNew[i+1] = len(newInsts) - 1
					oldToNew[i+2] = len(newInsts) - 1
					i += 2
					continue
				}

				op := NeoOpCode(0)
				switch c.instructions[i+2].Op {
				case NeoOpEqual: op = NeoOpEqualGlobalConst
				case NeoOpGreater: op = NeoOpGreaterGlobalConst
				case NeoOpLess: op = NeoOpLessGlobalConst
				}

				if op != 0 {
					newInsts = append(newInsts, neoInstruction{Op: op, Arg: (gIdx << 16) | cIdx})
					oldToNew[i+1] = len(newInsts) - 1
					oldToNew[i+2] = len(newInsts) - 1
					i += 2
					continue
				}
			}
		}

		if i+2 < len(c.instructions) &&
			inst.Op == NeoOpGetGlobal &&
			c.instructions[i+1].Op == NeoOpGetGlobal &&
			c.instructions[i+2].Op == NeoOpAdd {

			g1Idx := inst.Arg
			g2Idx := c.instructions[i+1].Arg
			if g1Idx < 65536 && g2Idx < 65536 {
				newInsts = append(newInsts, neoInstruction{Op: NeoOpAddGlobalGlobal, Arg: (g1Idx << 16) | g2Idx})
				oldToNew[i+1] = len(newInsts) - 1
				oldToNew[i+2] = len(newInsts) - 1
				i += 2
				continue
			}
		}

		if i+1 < len(c.instructions) &&
			inst.Op == NeoOpGetGlobal {

			gIdx := inst.Arg
			jTarget := c.instructions[i+1].Arg

			if gIdx < 65536 && jTarget < 65536 {
				op := NeoOpCode(0)
				switch c.instructions[i+1].Op {
				case NeoOpJumpIfFalse: op = NeoOpGetGlobalJumpIfFalse
				case NeoOpJumpIfTrue: op = NeoOpGetGlobalJumpIfTrue
				}

				if op != 0 {
					newInsts = append(newInsts, neoInstruction{Op: op, Arg: (gIdx << 16) | jTarget})
					oldToNew[i+1] = len(newInsts) - 1
					i += 1
					continue
				}
			}
		}

		newInsts = append(newInsts, inst)
	}
	oldToNew[len(c.instructions)] = len(newInsts)

	for i := range newInsts {
		switch newInsts[i].Op {
		case NeoOpJump, NeoOpJumpIfFalse, NeoOpJumpIfTrue:
			newInsts[i].Arg = int32(oldToNew[newInsts[i].Arg])
		case NeoOpFusedCompareGlobalConstJumpIfFalse:
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

	c.instructions = newInsts
}
