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
	ASSIGN      // =
	OR          // ||
	AND         // &&
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
)

var precedences = map[TokenType]int{
	TokenAssign:   ASSIGN,
	TokenEq:       EQUALS,
	TokenNotEq:    EQUALS,
	TokenLt:       LESSGREATER,
	TokenGt:       LESSGREATER,
	TokenLe:       LESSGREATER,
	TokenGe:       LESSGREATER,
	TokenAnd:      AND,
	TokenOr:       OR,
	TokenPlus:     SUM,
	TokenMinus:    SUM,
	TokenSlash:    PRODUCT,
	TokenAsterisk: PRODUCT,
	TokenPercent:  PRODUCT,
	TokenLParen:   CALL,
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

type Parser struct {
	l      *Lexer
	errors []string

	curToken  Token
	peekToken Token

	prefixParseFns map[TokenType]prefixParseFn
	infixParseFns  map[TokenType]infixParseFn
}

var ParserPool = sync.Pool{
	New: func() any {
		return &Parser{}
	},
}

func NewParser(l *Lexer) *Parser {
	p := ParserPool.Get().(*Parser)
	p.l = l
	p.errors = []string{}

	p.prefixParseFns = make(map[TokenType]prefixParseFn)
	p.registerPrefix(TokenIdent, p.parseIdentifier)
	p.registerPrefix(TokenNumber, p.parseNumberLiteral)
	p.registerPrefix(TokenString, p.parseStringLiteral)
	p.registerPrefix(TokenTrue, p.parseBooleanLiteral)
	p.registerPrefix(TokenFalse, p.parseBooleanLiteral)
	p.registerPrefix(TokenMinus, p.parsePrefixExpression)
	p.registerPrefix(TokenBang, p.parsePrefixExpression)
	p.registerPrefix(TokenLParen, p.parseGroupedExpression)
	p.registerPrefix(TokenIf, p.parseIfExpression)

	p.infixParseFns = make(map[TokenType]infixParseFn)
	p.registerInfix(TokenPlus, p.parseInfixExpression)
	p.registerInfix(TokenMinus, p.parseInfixExpression)
	p.registerInfix(TokenSlash, p.parseInfixExpression)
	p.registerInfix(TokenAsterisk, p.parseInfixExpression)
	p.registerInfix(TokenPercent, p.parseInfixExpression)
	p.registerInfix(TokenEq, p.parseInfixExpression)
	p.registerInfix(TokenNotEq, p.parseInfixExpression)
	p.registerInfix(TokenLt, p.parseInfixExpression)
	p.registerInfix(TokenGt, p.parseInfixExpression)
	p.registerInfix(TokenLe, p.parseInfixExpression)
	p.registerInfix(TokenGe, p.parseInfixExpression)
	p.registerInfix(TokenAnd, p.parseInfixExpression)
	p.registerInfix(TokenOr, p.parseInfixExpression)
	p.registerInfix(TokenAssign, p.parseAssignExpression)
	p.registerInfix(TokenLParen, p.parseCallExpression)

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) ParseProgram() Expression {
	if p.curToken.Type == TokenEOF {
		return nil
	}
	return p.parseExpression(LOWEST)
}

func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.errors = append(p.errors, fmt.Sprintf("no prefix parse function for %s found", p.curToken.Type))
		return nil
	}
	leftExp := prefix()

	for p.peekToken.Type != TokenEOF && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() Expression {
	return &Identifier{Value: p.curToken.Literal}
}

func (p *Parser) parseNumberLiteral() Expression {
	lit := &NumberLiteral{}
	if i, err := strconv.ParseInt(p.curToken.Literal, 10, 64); err == nil {
		lit.Int64Value = i
		lit.IsInt = true
	} else if f, err := strconv.ParseFloat(p.curToken.Literal, 64); err == nil {
		lit.Float64Value = f
		lit.IsInt = false
	} else {
		p.errors = append(p.errors, fmt.Sprintf("could not parse %q as number", p.curToken.Literal))
		return nil
	}
	return lit
}

func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() Expression {
	return &BooleanLiteral{Value: p.curToken.Type == TokenTrue}
}

func (p *Parser) parsePrefixExpression() Expression {
	expression := &PrefixExpression{
		Operator: p.curToken.Literal,
	}
	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)
	return expression
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	expression := &InfixExpression{
		Operator: p.curToken.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)
	return expression
}

func (p *Parser) parseAssignExpression(left Expression) Expression {
	name, ok := left.(*Identifier)
	if !ok {
		p.errors = append(p.errors, "expected identifier on left side of assignment")
		return nil
	}
	p.nextToken()
	val := p.parseExpression(LOWEST)
	return &AssignExpression{Name: name, Value: val}
}

func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(TokenRParen) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() Expression {
	exp := &IfExpression{}
	p.nextToken()
	exp.Condition = p.parseExpression(LOWEST)

	if p.peekToken.Type == TokenIs {
		p.nextToken()
		p.nextToken()
		exp.Consequence = p.parseExpression(LOWEST)
		exp.IsThen = false
	} else if p.peekToken.Type == TokenThen {
		p.nextToken()
		p.nextToken()
		exp.Consequence = p.parseExpression(LOWEST)
		exp.IsThen = true
	} else {
		exp.IsSimple = true
		return exp
	}

	if p.peekToken.Type == TokenElse {
		p.nextToken()
		if p.peekToken.Type == TokenIf {
			p.nextToken()
			exp.Alternative = p.parseIfExpression()
		} else if p.peekToken.Type == TokenIs {
			p.nextToken()
			p.nextToken()
			exp.Alternative = p.parseExpression(LOWEST)
		} else {
			p.nextToken()
			exp.Alternative = p.parseExpression(LOWEST)
		}
	}
	return exp
}

func (p *Parser) parseCallExpression(function Expression) Expression {
	exp := &CallExpression{Function: function}
	exp.Arguments = p.parseCallArguments()
	return exp
}

func (p *Parser) parseCallArguments() []Expression {
	args := []Expression{}
	if p.peekToken.Type == TokenRParen {
		p.nextToken()
		return args
	}
	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))
	for p.peekToken.Type == TokenComma {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}
	if !p.expectPeek(TokenRParen) {
		return nil
	}
	return args
}

func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	}
	p.errors = append(p.errors, fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type))
	return false
}

func (p *Parser) peekPrecedence() int {
	if pre, ok := precedences[p.peekToken.Type]; ok {
		return pre
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if pre, ok := precedences[p.curToken.Type]; ok {
		return pre
	}
	return LOWEST
}

func (p *Parser) registerPrefix(tokenType TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}
