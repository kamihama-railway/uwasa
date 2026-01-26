// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"strconv"
	"sync"
)

const (
	_ int = iota
	LOWEST
	ASSIGN
	OR
	AND
	EQUALS
	LESSGREATER
	SUM
	PREFIX
)

func getPrecedence(t TokenType) int {
	switch t {
	case TokenAssign:
		return ASSIGN
	case TokenOr:
		return OR
	case TokenAnd:
		return AND
	case TokenEq:
		return EQUALS
	case TokenGt, TokenLt, TokenGe, TokenLe:
		return LESSGREATER
	case TokenPlus, TokenMinus:
		return SUM
	default:
		return LOWEST
	}
}

type Parser struct {
	l      *Lexer
	curTok Token
	peekTok Token
	errors []string

	prefixParseFns map[TokenType]prefixParseFn
	infixParseFns  map[TokenType]infixParseFn
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

var parserPool = sync.Pool{
	New: func() any {
		p := &Parser{}
		p.prefixParseFns = make(map[TokenType]prefixParseFn)
		p.infixParseFns = make(map[TokenType]infixParseFn)

		p.registerPrefix(TokenIdent, p.parseIdentifier)
		p.registerPrefix(TokenNumber, p.parseNumberLiteral)
		p.registerPrefix(TokenString, p.parseStringLiteral)
		p.registerPrefix(TokenTrue, p.parseBooleanLiteral)
		p.registerPrefix(TokenFalse, p.parseBooleanLiteral)
		p.registerPrefix(TokenMinus, p.parsePrefixExpression)
		p.registerPrefix(TokenLParen, p.parseGroupedExpression)
		p.registerPrefix(TokenIf, p.parseIfExpression)

		p.registerInfix(TokenOr, p.parseInfixExpression)
		p.registerInfix(TokenAnd, p.parseInfixExpression)
		p.registerInfix(TokenEq, p.parseInfixExpression)
		p.registerInfix(TokenGt, p.parseInfixExpression)
		p.registerInfix(TokenLt, p.parseInfixExpression)
		p.registerInfix(TokenGe, p.parseInfixExpression)
		p.registerInfix(TokenLe, p.parseInfixExpression)
		p.registerInfix(TokenPlus, p.parseInfixExpression)
		p.registerInfix(TokenMinus, p.parseInfixExpression)
		p.registerInfix(TokenAssign, p.parseAssignExpression)

		return p
	},
}

func NewParser(l *Lexer) *Parser {
	p := parserPool.Get().(*Parser)
	p.Reset(l)
	return p
}

func (p *Parser) Reset(l *Lexer) {
	p.l = l
	p.errors = p.errors[:0]
	p.nextToken()
	p.nextToken()
}

func (p *Parser) nextToken() {
	p.curTok = p.peekTok
	p.peekTok = p.l.NextToken()
}

func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.curTok.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curTok.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(TokenEOF) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekTok.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) registerPrefix(tokenType TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) peekPrecedence() int {
	return getPrecedence(p.peekTok.Type)
}

func (p *Parser) curPrecedence() int {
	return getPrecedence(p.curTok.Type)
}

func (p *Parser) parseIdentifier() Expression {
	return &Identifier{Value: p.curTok.Literal}
}

func (p *Parser) parseNumberLiteral() Expression {
	val, err := strconv.ParseFloat(p.curTok.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as float64", p.curTok.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	return &NumberLiteral{Value: val}
}

func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Value: p.curTok.Literal}
}

func (p *Parser) parseBooleanLiteral() Expression {
	return &BooleanLiteral{Value: p.curTokenIs(TokenTrue)}
}

func (p *Parser) parsePrefixExpression() Expression {
	expression := &PrefixExpression{
		Operator: p.curTok.Literal,
	}
	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)
	return expression
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	expression := &InfixExpression{
		Operator: p.curTok.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)
	return expression
}

func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(TokenRParen) {
		return nil
	}
	return exp
}

func (p *Parser) parseAssignExpression(left Expression) Expression {
	ident, ok := left.(*Identifier)
	if !ok {
		p.errors = append(p.errors, "left side of assignment must be an identifier")
		return nil
	}
	expression := &AssignExpression{Name: ident}
	p.nextToken()
	expression.Value = p.parseExpression(LOWEST)
	return expression
}

func (p *Parser) parseIfExpression() Expression {
	expression := &IfExpression{}
	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if p.peekTokenIs(TokenIs) {
		p.nextToken() // cur is 'is'
		p.nextToken() // move to expression after 'is'
		expression.Consequence = p.parseExpression(LOWEST)
		expression.IsThen = false

		if p.peekTokenIs(TokenElse) {
			p.nextToken() // cur is 'else'
			if p.peekTokenIs(TokenIf) {
				p.nextToken() // cur is 'if'
				expression.Alternative = p.parseIfExpression()
			} else if p.peekTokenIs(TokenIs) {
				p.nextToken() // cur is 'is'
				p.nextToken() // move to expression after 'is'
				expression.Alternative = p.parseExpression(LOWEST)
			} else {
				// Handle case "else is ..." without explicit "is" token?
				// Spec says "else is "bad""
				p.errors = append(p.errors, "expected 'if' or 'is' after 'else'")
			}
		}
	} else if p.peekTokenIs(TokenThen) {
		p.nextToken() // cur is 'then'
		p.nextToken() // move to expression after 'then'
		expression.Consequence = p.parseExpression(LOWEST)
		expression.IsThen = true
		// No 'else' mentioned for 'then' in spec, but we could support it
	} else {
		// Simple 'if <cond>'
		expression.IsSimple = true
	}

	return expression
}

func (p *Parser) peekTokenIs(t TokenType) bool {
	return p.peekTok.Type == t
}

func (p *Parser) curTokenIs(t TokenType) bool {
	return p.curTok.Type == t
}

func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekTok.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) noPrefixParseFnError(t TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) ParseProgram() Expression {
	return p.parseExpression(LOWEST)
}
