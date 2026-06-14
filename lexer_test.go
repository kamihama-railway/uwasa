package uwasa

import (
	"testing"
)

func TestLexer(t *testing.T) {
	input := `if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenIf, "if"},
		{TokenIdent, "a"},
		{TokenEq, "=="},
		{TokenNumber, "0"},
		{TokenIs, "is"},
		{TokenString, "yes"},
		{TokenElse, "else"},
		{TokenIf, "if"},
		{TokenIdent, "a"},
		{TokenEq, "=="},
		{TokenNumber, "1"},
		{TokenIs, "is"},
		{TokenString, "ok"},
		{TokenElse, "else"},
		{TokenIs, "is"},
		{TokenString, "bad"},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
	lexerPool.Put(l)
}

func TestLexerKeywords(t *testing.T) {
	input := `true false if is else then`
	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenTrue, "true"},
		{TokenFalse, "false"},
		{TokenIf, "if"},
		{TokenIs, "is"},
		{TokenElse, "else"},
		{TokenThen, "then"},
		{TokenEOF, ""},
	}
	l := NewLexer(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
	lexerPool.Put(l)
}

func TestLexerNumbersAndIdents(t *testing.T) {
	input := `123 123.456 _var_name var123`
	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenNumber, "123"},
		{TokenNumber, "123.456"},
		{TokenIdent, "_var_name"},
		{TokenIdent, "var123"},
		{TokenEOF, ""},
	}
	l := NewLexer(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
	lexerPool.Put(l)
}

func TestLexer2(t *testing.T) {
	input := `if a == 0 && b >= 1 then b = b + 10`
	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenIf, "if"},
		{TokenIdent, "a"},
		{TokenEq, "=="},
		{TokenNumber, "0"},
		{TokenAnd, "&&"},
		{TokenIdent, "b"},
		{TokenGe, ">="},
		{TokenNumber, "1"},
		{TokenThen, "then"},
		{TokenIdent, "b"},
		{TokenAssign, "="},
		{TokenIdent, "b"},
		{TokenPlus, "+"},
		{TokenNumber, "10"},
		{TokenEOF, ""},
	}
	l := NewLexer(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexerIllegal(t *testing.T) {
	input := `a & b`
	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenIdent, "a"},
		{TokenIllegal, "&"},
		{TokenIdent, "b"},
		{TokenEOF, ""},
	}
	l := NewLexer(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexerChineseIdent(t *testing.T) {
	input := `分数 = 95`
	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenIdent, "分数"},
		{TokenAssign, "="},
		{TokenNumber, "95"},
		{TokenEOF, ""},
	}
	l := NewLexer(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
	lexerPool.Put(l)
}

// TestLexerStringPosition verifies that readString() correctly consumes the
// closing quote by testing strings adjacent to other tokens. If the closing "
// were not consumed (e.g., if case '"' had a return before l.readChar()),
// these sequences would fail:
//   - "hello"+"world" → "hello" would consume "+world" as part of the string
//   - "a" "b" → the space before "b" would be misread
func TestLexerStringPosition(t *testing.T) {
	tests := []struct {
		name  string
		input string
		items []struct {
			typ TokenType
			lit string
		}
	}{
		{
			name:  "string + string (no space)",
			input: `"hello"+"world"`,
			items: []struct {
				typ TokenType
				lit string
			}{
				{TokenString, "hello"},
				{TokenPlus, "+"},
				{TokenString, "world"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "string + string (with spaces)",
			input: `"hello" + "world"`,
			items: []struct {
				typ TokenType
				lit string
			}{
				{TokenString, "hello"},
				{TokenPlus, "+"},
				{TokenString, "world"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "empty string",
			input: `""`,
			items: []struct {
				typ TokenType
				lit string
			}{
				{TokenString, ""},
				{TokenEOF, ""},
			},
		},
		{
			name:  "string followed by identifier",
			input: `"hello" foo`,
			items: []struct {
				typ TokenType
				lit string
			}{
				{TokenString, "hello"},
				{TokenIdent, "foo"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "string with comparison",
			input: `"hello" == "world"`,
			items: []struct {
				typ TokenType
				lit string
			}{
				{TokenString, "hello"},
				{TokenEq, "=="},
				{TokenString, "world"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "string at EOF",
			input: `"hello"`,
			items: []struct {
				typ TokenType
				lit string
			}{
				{TokenString, "hello"},
				{TokenEOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.input)
			for i, item := range tt.items {
				tok := l.NextToken()
				if tok.Type != item.typ {
					t.Fatalf("token %d: type = %q, want %q", i, tok.Type, item.typ)
				}
				if tok.Literal != item.lit {
					t.Fatalf("token %d: literal = %q, want %q", i, tok.Literal, item.lit)
				}
			}
			lexerPool.Put(l)
		})
	}
}

// TestLexerStringInternalState demonstrates the root cause:
// readString() does NOT consume the closing quote.
// After readString() returns, l.ch is still '"', relying on NextToken()'s
// post-switch l.readChar() to advance past it.
//
// This test then simulates what happens if that l.readChar() is missed
// (e.g., if case '"' is modified to return early) — the closing quote
// bleeds into the next token sequence.
func TestLexerStringInternalState(t *testing.T) {
	t.Run("readString_leaves_closing_quote", func(t *testing.T) {
		input := `"abc"`
		l := NewLexer(input)

		l.skipWhitespace()
		str := l.readString()
		if str != "abc" {
			t.Fatalf("readString() = %q, want %q", str, "abc")
		}

		// readString() does NOT consume the closing "
		if l.ch != '"' {
			t.Fatalf("readString() consumed the closing quote (l.ch = %q, want '\"'). "+
				"If this fails, the vulnerability no longer exists.", l.ch)
		}

		// l.position still points to the closing "
		pos := l.position
		expectedClose := len(`"abc"`) - 1
		if pos != expectedClose {
			t.Fatalf("after readString(): l.position = %d, want %d (closing quote offset)", pos, expectedClose)
		}

		lexerPool.Put(l)
	})

	t.Run("simulated_bug_adjacent_tokens", func(t *testing.T) {
		// Without the post-switch l.readChar(), the lexer would fail on
		// strings followed by other tokens. We simulate this by checking
		// the parser's behavior on "hello"+"world" — closing quote position
		// must be correct for the next token to start at "+".
		input := `"hello"+"world"`
		l := NewLexer(input)

		tok1 := l.NextToken()
		if tok1.Type != TokenString || tok1.Literal != "hello" {
			t.Fatalf("first token: got %s %q, want TokenString %q", tok1.Type, tok1.Literal, "hello")
		}

		// After NextToken() returns, l.ch should be '+' (the next token)
		if l.ch != '+' {
			t.Fatalf("after first NextToken(): l.ch = %q, want '+'. "+
				"This means the closing quote was consumed correctly by the post-switch l.readChar().",
				l.ch)
		}

		tok2 := l.NextToken()
		if tok2.Type != TokenPlus || tok2.Literal != "+" {
			t.Fatalf("second token: got %s %q, want TokenPlus '+'", tok2.Type, tok2.Literal)
		}

		lexerPool.Put(l)
	})
}

func TestLexerChineseExpression(t *testing.T) {
	input := `if 分数 >= 90 then 奖金 = 100`
	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenIf, "if"},
		{TokenIdent, "分数"},
		{TokenGe, ">="},
		{TokenNumber, "90"},
		{TokenThen, "then"},
		{TokenIdent, "奖金"},
		{TokenAssign, "="},
		{TokenNumber, "100"},
		{TokenEOF, ""},
	}
	l := NewLexer(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
	lexerPool.Put(l)
}

func TestLexerChineseMixedIdent(t *testing.T) {
	input := `_总分_avg1 = (语文 + 数学) / 2`
	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenIdent, "_总分_avg1"},
		{TokenAssign, "="},
		{TokenLParen, "("},
		{TokenIdent, "语文"},
		{TokenPlus, "+"},
		{TokenIdent, "数学"},
		{TokenRParen, ")"},
		{TokenSlash, "/"},
		{TokenNumber, "2"},
		{TokenEOF, ""},
	}
	l := NewLexer(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
	lexerPool.Put(l)
}
