// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package parser

import (
	"fmt"
	"strconv"
	"sync"
	"github.com/kamihama-railway/uwasa/lexer"
	"github.com/kamihama-railway/uwasa/ast"
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
	PRODUCT
	PREFIX
	CALL
)

func getPrecedence(t lexer.TokenType) int {
	switch t {
	case lexer.TokenAssign:
		return ASSIGN
	case lexer.TokenOr:
		return OR
	case lexer.TokenAnd:
		return AND
	case lexer.TokenEq:
		return EQUALS
	case lexer.TokenGt, lexer.TokenLt, lexer.TokenGe, lexer.TokenLe:
		return LESSGREATER
	case lexer.TokenPlus, lexer.TokenMinus:
		return SUM
	case lexer.TokenAsterisk, lexer.TokenSlash, lexer.TokenPercent:
		return PRODUCT
	case lexer.TokenLParen:
		return CALL
	default:
		return LOWEST
	}
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  lexer.Token
	peekToken lexer.Token

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

func (p *Parser) init() {
	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)

	p.registerPrefix(lexer.TokenIdent, p.parseIdentifier)
	p.registerPrefix(lexer.TokenNumber, p.parseNumberLiteral)
	p.registerPrefix(lexer.TokenString, p.parseStringLiteral)
	p.registerPrefix(lexer.TokenTrue, p.parseBooleanLiteral)
	p.registerPrefix(lexer.TokenFalse, p.parseBooleanLiteral)
	p.registerPrefix(lexer.TokenBang, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenMinus, p.parsePrefixExpression)
	p.registerPrefix(lexer.TokenLParen, p.parseGroupedExpression)
	p.registerPrefix(lexer.TokenIf, p.parseIfExpression)

	p.registerInfix(lexer.TokenPlus, p.parseInfixExpression)
	p.registerInfix(lexer.TokenMinus, p.parseInfixExpression)
	p.registerInfix(lexer.TokenSlash, p.parseInfixExpression)
	p.registerInfix(lexer.TokenAsterisk, p.parseInfixExpression)
	p.registerInfix(lexer.TokenPercent, p.parseInfixExpression)
	p.registerInfix(lexer.TokenEq, p.parseInfixExpression)
	p.registerInfix(lexer.TokenGt, p.parseInfixExpression)
	p.registerInfix(lexer.TokenLt, p.parseInfixExpression)
	p.registerInfix(lexer.TokenGe, p.parseInfixExpression)
	p.registerInfix(lexer.TokenLe, p.parseInfixExpression)
	p.registerInfix(lexer.TokenAnd, p.parseInfixExpression)
	p.registerInfix(lexer.TokenOr, p.parseInfixExpression)
	p.registerInfix(lexer.TokenLParen, p.parseCallExpression)
	p.registerInfix(lexer.TokenAssign, p.parseAssignExpression)
}

var ParserPool = sync.Pool{
	New: func() any {
		p := &Parser{}
		p.init()
		return p
	},
}

func NewParser(l *lexer.Lexer) *Parser {
	p := ParserPool.Get().(*Parser)
	p.Reset(l)
	return p
}

func (p *Parser) Reset(l *lexer.Lexer) {
	p.l = l
	p.errors = p.errors[:0]

	p.nextToken()
	p.nextToken()
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() ast.Expression {
	return p.parseExpression(LOWEST)
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(lexer.TokenEOF) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Value: p.curToken.Literal}
}

func (p *Parser) parseNumberLiteral() ast.Expression {
	lit := &ast.NumberLiteral{}
	if val, err := strconv.ParseInt(p.curToken.Literal, 10, 64); err == nil {
		lit.Int64Value = val
		lit.IsInt = true
	} else if val, err := strconv.ParseFloat(p.curToken.Literal, 64); err == nil {
		lit.Float64Value = val
		lit.IsInt = false
	} else {
		p.errors = append(p.errors, fmt.Sprintf("could not parse %q as number", p.curToken.Literal))
		return nil
	}
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{Value: p.curTokenIs(lexer.TokenTrue)}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Operator: p.curToken.Literal,
	}
	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)
	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TokenRParen) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.TokenIs) {
		p.nextToken()
		p.nextToken()
		expression.Consequence = p.parseExpression(LOWEST)
		expression.IsThen = false
		expression.IsSimple = false
	} else if p.peekTokenIs(lexer.TokenThen) {
		p.nextToken()
		p.nextToken()
		expression.Consequence = p.parseExpression(LOWEST)
		expression.IsThen = true
		expression.IsSimple = false
	} else {
		expression.IsSimple = true
		return expression
	}

	if p.peekTokenIs(lexer.TokenElse) {
		p.nextToken()
		if p.peekTokenIs(lexer.TokenIs) {
			p.nextToken()
		}
		p.nextToken()
		expression.Alternative = p.parseExpression(LOWEST)
	}

	return expression
}

func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	ident, ok := left.(*ast.Identifier)
	if !ok {
		p.errors = append(p.errors, "left side of assignment must be an identifier")
		return nil
	}
	expression := &ast.AssignExpression{Name: ident}
	p.nextToken()
	expression.Value = p.parseExpression(LOWEST)
	return expression
}

func (p *Parser) parseCallExpression(left ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Function: left}
	exp.Arguments = p.parseExpressionList(lexer.TokenRParen)
	return exp
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.TokenComma) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
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

func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) curPrecedence() int {
	return getPrecedence(p.curToken.Type)
}

func (p *Parser) peekPrecedence() int {
	return getPrecedence(p.peekToken.Type)
}
