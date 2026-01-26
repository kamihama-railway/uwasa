// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"sync"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIf
	TokenIs
	TokenElse
	TokenThen
	TokenAssign    // =
	TokenPlus      // +
	TokenMinus     // -
	TokenGt        // >
	TokenLt        // <
	TokenGe        // >=
	TokenLe        // <=
	TokenEq        // ==
	TokenAnd       // &&
	TokenOr        // ||
	TokenIllegal   // illegal
	TokenIdent     // identifier
	TokenNumber    // number literal
	TokenString    // string literal
	TokenTrue      // true
	TokenFalse     // false
	TokenLParen    // (
	TokenRParen    // )
)

type Token struct {
	Type    TokenType
	Literal string
}

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
}

var lexerPool = sync.Pool{
	New: func() any {
		return &Lexer{}
	},
}

func NewLexer(input string) *Lexer {
	l := lexerPool.Get().(*Lexer)
	l.Reset(input)
	return l
}

func (l *Lexer) Reset(input string) {
	l.input = input
	l.position = 0
	l.readPosition = 0
	l.ch = 0
	l.readChar()
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TokenEq, Literal: "=="}
		} else {
			tok = Token{Type: TokenAssign, Literal: "="}
		}
	case '+':
		tok = Token{Type: TokenPlus, Literal: "+"}
	case '-':
		tok = Token{Type: TokenMinus, Literal: "-"}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TokenGe, Literal: ">="}
		} else {
			tok = Token{Type: TokenGt, Literal: ">"}
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TokenLe, Literal: "<="}
		} else {
			tok = Token{Type: TokenLt, Literal: "<"}
		}
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok = Token{Type: TokenAnd, Literal: "&&"}
		} else {
			tok = Token{Type: TokenIllegal, Literal: string(l.ch)}
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = Token{Type: TokenOr, Literal: "||"}
		} else {
			tok = Token{Type: TokenIllegal, Literal: string(l.ch)}
		}
	case '(':
		tok = Token{Type: TokenLParen, Literal: "("}
	case ')':
		tok = Token{Type: TokenRParen, Literal: ")"}
	case '"':
		tok.Type = TokenString
		tok.Literal = l.readString()
	case 0:
		tok.Literal = ""
		tok.Type = TokenEOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = lookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Literal = l.readNumber()
			tok.Type = TokenNumber
			return tok
		} else {
			tok = Token{Type: TokenIllegal, Literal: string(l.ch)}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) || l.ch == '.' {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readString() string {
	l.readChar() // skip "
	position := l.position
	for l.ch != '"' && l.ch != 0 {
		l.readChar()
	}
	str := l.input[position:l.position]
	// l.readChar() // would be done by caller or next read?
	// Actually we should skip the closing quote
	return str
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

var keywords = map[string]TokenType{
	"if":    TokenIf,
	"is":    TokenIs,
	"else":  TokenElse,
	"then":  TokenThen,
	"true":  TokenTrue,
	"false": TokenFalse,
}

func lookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TokenIdent
}

func (t TokenType) String() string {
	switch t {
	case TokenEOF: return "EOF"
	case TokenIf: return "if"
	case TokenIs: return "is"
	case TokenElse: return "else"
	case TokenThen: return "then"
	case TokenAssign: return "="
	case TokenPlus: return "+"
	case TokenMinus: return "-"
	case TokenGt: return ">"
	case TokenLt: return "<"
	case TokenGe: return ">="
	case TokenLe: return "<="
	case TokenEq: return "=="
	case TokenAnd: return "&&"
	case TokenOr: return "||"
	case TokenIllegal: return "ILLEGAL"
	case TokenIdent: return "IDENT"
	case TokenNumber: return "NUMBER"
	case TokenString: return "STRING"
	case TokenTrue: return "true"
	case TokenFalse: return "false"
	case TokenLParen: return "("
	case TokenRParen: return ")"
	default: return "UNKNOWN"
	}
}
