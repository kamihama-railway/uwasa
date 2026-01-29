// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"sync"
)

var LexerPool = sync.Pool{
	New: func() any {
		return &Lexer{}
	},
}

type Lexer struct {
	Input        string
	Position     int
	ReadPosition int
	Ch           byte
}

func NewLexer(input string) *Lexer {
	l := LexerPool.Get().(*Lexer)
	l.Reset(input)
	return l
}

func (l *Lexer) Reset(input string) {
	l.Input = input
	l.Position = 0
	l.ReadPosition = 0
	l.Ch = 0
	l.readChar()
}

func (l *Lexer) readChar() {
	if l.ReadPosition >= len(l.Input) {
		l.Ch = 0
	} else {
		l.Ch = l.Input[l.ReadPosition]
	}
	l.Position = l.ReadPosition
	l.ReadPosition++
}

func (l *Lexer) peekChar() byte {
	if l.ReadPosition >= len(l.Input) {
		return 0
	}
	return l.Input[l.ReadPosition]
}

func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	switch l.Ch {
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
	case '*':
		tok = Token{Type: TokenAsterisk, Literal: "*"}
	case '/':
		tok = Token{Type: TokenSlash, Literal: "/"}
	case '%':
		tok = Token{Type: TokenPercent, Literal: "%"}
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
			tok = Token{Type: TokenIllegal, Literal: string(l.Ch)}
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = Token{Type: TokenOr, Literal: "||"}
		} else {
			tok = Token{Type: TokenIllegal, Literal: string(l.Ch)}
		}
	case '(':
		tok = Token{Type: TokenLParen, Literal: "("}
	case ')':
		tok = Token{Type: TokenRParen, Literal: ")"}
	case ',':
		tok = Token{Type: TokenComma, Literal: ","}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TokenNotEq, Literal: "!="}
		} else {
			tok = Token{Type: TokenBang, Literal: "!"}
		}
	case '"':
		tok.Type = TokenString
		tok.Literal = l.readString()
	case 0:
		tok.Literal = ""
		tok.Type = TokenEOF
	default:
		if isLetter(l.Ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = lookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.Ch) {
			tok.Literal = l.readNumber()
			tok.Type = TokenNumber
			return tok
		} else {
			tok = Token{Type: TokenIllegal, Literal: string(l.Ch)}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.Ch == ' ' || l.Ch == '\t' || l.Ch == '\n' || l.Ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.Position
	for isLetter(l.Ch) || isDigit(l.Ch) {
		l.readChar()
	}
	return l.Input[position:l.Position]
}

func (l *Lexer) readNumber() string {
	position := l.Position
	for isDigit(l.Ch) || l.Ch == '.' {
		l.readChar()
	}
	return l.Input[position:l.Position]
}

func (l *Lexer) readString() string {
	l.readChar() // skip "
	position := l.Position
	for l.Ch != '"' && l.Ch != 0 {
		l.readChar()
	}
	str := l.Input[position:l.Position]
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
	TokenAsterisk  // *
	TokenSlash     // /
	TokenPercent   // %
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
	TokenComma     // ,
	TokenBang      // !
	TokenNotEq     // !=
)

type Token struct {
	Type    TokenType
	Literal string
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
	case TokenAsterisk: return "*"
	case TokenSlash: return "/"
	case TokenPercent: return "%"
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
	case TokenComma: return ","
	case TokenBang: return "!"
	case TokenNotEq: return "!="
	default: return "UNKNOWN"
	}
}
