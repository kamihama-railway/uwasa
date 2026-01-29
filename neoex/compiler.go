// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package neoex

import (
	"fmt"
	"math"
	"strconv"
	"github.com/kamihama-railway/uwasa/lexer"
	"github.com/kamihama-railway/uwasa/types"
)

type Compiler struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token

	instructions []Instruction
	constants    []types.Value
	constMap     map[any]int32
}

func NewCompiler(input string) *Compiler {
	c := &Compiler{
		l:        lexer.NewLexer(input),
		constMap: make(map[any]int32),
	}
	c.nextToken()
	c.nextToken()
	return c
}

func (c *Compiler) nextToken() {
	c.curToken = c.peekToken
	c.peekToken = c.l.NextToken()
}

func (c *Compiler) Compile() (*Bytecode, error) {
	reg, err := c.parseExpression(LOWEST, 0)
	if err != nil {
		return nil, err
	}
	c.emit(OpReturn, 0, reg, 0, 0)
	return &Bytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
	}, nil
}

const (
	LOWEST int = iota
	ASSIGN
	OR
	AND
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	PREFIX
	CALL
)

var precedences = map[lexer.TokenType]int{
	lexer.TokenAssign:   ASSIGN,
	lexer.TokenOr:       OR,
	lexer.TokenAnd:      AND,
	lexer.TokenEq:       EQUALS,
	lexer.TokenGt:       LESSGREATER,
	lexer.TokenLt:       LESSGREATER,
	lexer.TokenGe:       LESSGREATER,
	lexer.TokenLe:       LESSGREATER,
	lexer.TokenPlus:     SUM,
	lexer.TokenMinus:    SUM,
	lexer.TokenAsterisk: PRODUCT,
	lexer.TokenSlash:    PRODUCT,
	lexer.TokenPercent:  PRODUCT,
	lexer.TokenLParen:   CALL,
}

func (c *Compiler) curPrecedence() int {
	if p, ok := precedences[c.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (c *Compiler) peekPrecedence() int {
	if p, ok := precedences[c.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (c *Compiler) parseExpression(precedence int, reg uint8) (uint8, error) {
	if reg >= 250 {
		return 0, fmt.Errorf("register limit exceeded")
	}
	leftReg := reg
	var err error

	switch c.curToken.Type {
	case lexer.TokenIdent:
		c.emit(OpGetGlobal, leftReg, 0, 0, c.addConstant(types.Value{Type: types.ValString, Str: c.curToken.Literal}))
	case lexer.TokenNumber:
		val := c.curToken.Literal
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			c.emit(OpLoadConst, leftReg, 0, 0, c.addConstant(types.Value{Type: types.ValInt, Num: uint64(i)}))
		} else if f, err := strconv.ParseFloat(val, 64); err == nil {
			c.emit(OpLoadConst, leftReg, 0, 0, c.addConstant(types.Value{Type: types.ValFloat, Num: math.Float64bits(f)}))
		}
	case lexer.TokenString:
		c.emit(OpLoadConst, leftReg, 0, 0, c.addConstant(types.Value{Type: types.ValString, Str: c.curToken.Literal}))
	case lexer.TokenTrue:
		c.emit(OpLoadConst, leftReg, 0, 0, c.addConstant(types.Value{Type: types.ValBool, Num: 1}))
	case lexer.TokenFalse:
		c.emit(OpLoadConst, leftReg, 0, 0, c.addConstant(types.Value{Type: types.ValBool, Num: 0}))
	case lexer.TokenLParen:
		c.nextToken()
		leftReg, err = c.parseExpression(LOWEST, reg)
		if err != nil {
			return 0, err
		}
		if !c.expectPeek(lexer.TokenRParen) {
			return 0, fmt.Errorf("expected )")
		}
	case lexer.TokenMinus:
		c.nextToken()
		c.emit(OpLoadConst, leftReg, 0, 0, c.addConstant(types.Value{Type: types.ValInt, Num: 0}))
		rReg, err := c.parseExpression(PREFIX, reg+1)
		if err != nil {
			return 0, err
		}
		c.emit(OpSub, leftReg, leftReg, rReg, 0)
	case lexer.TokenBang:
		c.nextToken()
		rReg, err := c.parseExpression(PREFIX, reg)
		if err != nil {
			return 0, err
		}
		c.emit(OpNot, leftReg, rReg, 0, 0)
	case lexer.TokenIf:
		return c.parseIfExpression(reg)
	default:
		return 0, fmt.Errorf("unexpected token %v", c.curToken)
	}

	for !c.peekTokenIs(lexer.TokenEOF) && precedence < c.peekPrecedence() {
		c.nextToken()
		leftReg, err = c.parseInfixExpression(leftReg)
		if err != nil {
			return 0, err
		}
	}

	return leftReg, nil
}

func (c *Compiler) peekTokenIs(t lexer.TokenType) bool {
	return c.peekToken.Type == t
}

func (c *Compiler) expectPeek(t lexer.TokenType) bool {
	if c.peekTokenIs(t) {
		c.nextToken()
		return true
	}
	return false
}

func (c *Compiler) parseInfixExpression(leftReg uint8) (uint8, error) {
	op := c.curToken.Type
	prec := c.curPrecedence()

	if op == lexer.TokenAnd {
		jumpFalse := c.emit(OpJumpIfFalse, 0, leftReg, 0, 0)
		c.nextToken()
		_, err := c.parseExpression(prec, leftReg)
		if err != nil {
			return 0, err
		}
		c.emit(OpNot, leftReg, leftReg, 0, 0)
		c.emit(OpNot, leftReg, leftReg, 0, 0)
		jumpEnd := c.emit(OpJump, 0, 0, 0, 0)
		c.patch(jumpFalse, int32(len(c.instructions)))
		c.emit(OpLoadConst, leftReg, 0, 0, c.addConstant(types.Value{Type: types.ValBool, Num: 0}))
		c.patch(jumpEnd, int32(len(c.instructions)))
		return leftReg, nil
	}

	if op == lexer.TokenOr {
		jumpTrue := c.emit(OpJumpIfTrue, 0, leftReg, 0, 0)
		c.nextToken()
		_, err := c.parseExpression(prec, leftReg)
		if err != nil {
			return 0, err
		}
		c.emit(OpNot, leftReg, leftReg, 0, 0)
		c.emit(OpNot, leftReg, leftReg, 0, 0)
		jumpEnd := c.emit(OpJump, 0, 0, 0, 0)
		c.patch(jumpTrue, int32(len(c.instructions)))
		c.emit(OpLoadConst, leftReg, 0, 0, c.addConstant(types.Value{Type: types.ValBool, Num: 1}))
		c.patch(jumpEnd, int32(len(c.instructions)))
		return leftReg, nil
	}

	if op == lexer.TokenLParen {
		lastInst := &c.instructions[len(c.instructions)-1]
		if lastInst.Op != OpGetGlobal {
			return 0, fmt.Errorf("calling non-identifier not supported")
		}
		nameConstIdx := lastInst.Arg
		c.instructions = c.instructions[:len(c.instructions)-1]

		numArgs := 0
		if !c.peekTokenIs(lexer.TokenRParen) {
			for {
				c.nextToken()
				_, err := c.parseExpression(LOWEST, leftReg+uint8(numArgs))
				if err != nil {
					return 0, err
				}
				numArgs++
				if c.peekTokenIs(lexer.TokenComma) {
					c.nextToken()
				} else {
					break
				}
			}
		}
		if !c.expectPeek(lexer.TokenRParen) {
			return 0, fmt.Errorf("expected )")
		}

		name := c.constants[nameConstIdx].Str
		if name == "concat" {
			c.emit(OpConcat, leftReg, leftReg, uint8(numArgs), 0)
		} else {
			c.emit(OpCall, leftReg, leftReg, uint8(numArgs), nameConstIdx)
		}
		return leftReg, nil
	}

	if op == lexer.TokenAssign {
		lastInst := &c.instructions[len(c.instructions)-1]
		if lastInst.Op != OpGetGlobal {
			return 0, fmt.Errorf("left side of assignment must be an identifier")
		}
		nameConstIdx := lastInst.Arg
		c.instructions = c.instructions[:len(c.instructions)-1]

		c.nextToken()
		valReg, err := c.parseExpression(LOWEST, leftReg)
		if err != nil {
			return 0, err
		}
		c.emit(OpSetGlobal, 0, valReg, 0, nameConstIdx)
		return valReg, nil
	}

	// Peephole optimization for Fused Instructions
	if op == lexer.TokenEq || op == lexer.TokenPlus {
		lastIdx := len(c.instructions) - 1
		if lastIdx >= 0 {
			lastInst := &c.instructions[lastIdx]
			if lastInst.Op == OpGetGlobal && lastInst.Dest == leftReg {
				if c.peekToken.Type == lexer.TokenNumber || c.peekToken.Type == lexer.TokenString ||
					c.peekToken.Type == lexer.TokenTrue || c.peekToken.Type == lexer.TokenFalse {

					c.nextToken()
					var val types.Value
					switch c.curToken.Type {
					case lexer.TokenNumber:
						if i, err := strconv.ParseInt(c.curToken.Literal, 10, 64); err == nil {
							val = types.Value{Type: types.ValInt, Num: uint64(i)}
						} else if f, err := strconv.ParseFloat(c.curToken.Literal, 64); err == nil {
							val = types.Value{Type: types.ValFloat, Num: math.Float64bits(f)}
						}
					case lexer.TokenString:
						val = types.Value{Type: types.ValString, Str: c.curToken.Literal}
					case lexer.TokenTrue:
						val = types.Value{Type: types.ValBool, Num: 1}
					case lexer.TokenFalse:
						val = types.Value{Type: types.ValBool, Num: 0}
					}

					nameConstIdx := lastInst.Arg
					valConstIdx := c.addConstant(val)

					fusedOp := OpEqualGlobalConst
					if op == lexer.TokenPlus {
						fusedOp = OpAddGlobalConst
					}

					lastInst.Op = fusedOp
					lastInst.Arg = (nameConstIdx << 16) | (valConstIdx & 0xFFFF)
					return leftReg, nil
				}
			}
		}
	}

	c.nextToken()
	rightReg, err := c.parseExpression(prec, leftReg+1)
	if err != nil {
		return 0, err
	}

	var opcode OpCode
	switch op {
	case lexer.TokenPlus:
		opcode = OpAdd
	case lexer.TokenMinus:
		opcode = OpSub
	case lexer.TokenAsterisk:
		opcode = OpMul
	case lexer.TokenSlash:
		opcode = OpDiv
	case lexer.TokenPercent:
		opcode = OpMod
	case lexer.TokenEq:
		opcode = OpEqual
	case lexer.TokenGt:
		opcode = OpGreater
	case lexer.TokenLt:
		opcode = OpLess
	case lexer.TokenGe:
		opcode = OpGreaterEqual
	case lexer.TokenLe:
		opcode = OpLessEqual
	default:
		return 0, fmt.Errorf("unknown infix operator %v", op)
	}

	c.emit(opcode, leftReg, leftReg, rightReg, 0)
	return leftReg, nil
}

func (c *Compiler) parseIfExpression(reg uint8) (uint8, error) {
	c.nextToken()

	fusedGetGlobal := false
	var nameConstIdx int32
	if c.curToken.Type == lexer.TokenIdent && (c.peekTokenIs(lexer.TokenIs) || c.peekTokenIs(lexer.TokenThen)) {
		nameConstIdx = c.addConstant(types.Value{Type: types.ValString, Str: c.curToken.Literal})
		fusedGetGlobal = true
	} else {
		_, err := c.parseExpression(LOWEST, reg)
		if err != nil {
			return 0, err
		}
	}

	if c.peekTokenIs(lexer.TokenIs) || c.peekTokenIs(lexer.TokenThen) {
		isThen := c.peekTokenIs(lexer.TokenThen)
		c.nextToken()

		var jumpFalse int
		if fusedGetGlobal {
			jumpFalse = c.emit(OpGetGlobalJumpIfFalse, 0, 0, 0, nameConstIdx<<16)
		} else {
			jumpFalse = c.emit(OpJumpIfFalse, 0, reg, 0, 0)
		}

		c.nextToken()
		_, err := c.parseExpression(LOWEST, reg)
		if err != nil {
			return 0, err
		}

		jumpEnd := c.emit(OpJump, 0, 0, 0, 0)
		c.patch(jumpFalse, int32(len(c.instructions)))

		if c.peekTokenIs(lexer.TokenElse) {
			c.nextToken()
			if !isThen && c.peekTokenIs(lexer.TokenIs) {
				c.nextToken()
			}
			c.nextToken()
			_, err := c.parseExpression(LOWEST, reg)
			if err != nil {
				return 0, err
			}
		} else {
			c.emit(OpLoadConst, reg, 0, 0, c.addConstant(types.Value{Type: types.ValNil}))
		}
		c.patch(jumpEnd, int32(len(c.instructions)))
	}
	return reg, nil
}

func (c *Compiler) addConstant(v types.Value) int32 {
	var key any
	switch v.Type {
	case types.ValInt:
		key = int64(v.Num)
	case types.ValFloat:
		key = math.Float64frombits(v.Num)
	case types.ValBool:
		key = v.Num != 0
	case types.ValString:
		key = v.Str
	case types.ValNil:
		key = nil
	}
	if idx, ok := c.constMap[key]; ok {
		return idx
	}
	idx := int32(len(c.constants))
	c.constants = append(c.constants, v)
	c.constMap[key] = idx
	return idx
}

func (c *Compiler) emit(op OpCode, dest, src1, src2 uint8, arg int32) int {
	c.instructions = append(c.instructions, Instruction{Op: op, Dest: dest, Src1: src1, Src2: src2, Arg: arg})
	return len(c.instructions) - 1
}

func (c *Compiler) patch(pos int, arg int32) {
	if c.instructions[pos].Op == OpGetGlobalJumpIfFalse {
		c.instructions[pos].Arg |= (arg & 0xFFFF)
	} else {
		c.instructions[pos].Arg = arg
	}
}
