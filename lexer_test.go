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
