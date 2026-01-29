package uwasa
import "github.com/kamihama-railway/uwasa/lexer"

type Lexer = lexer.Lexer
type Token = lexer.Token
type TokenType = lexer.TokenType

const (
	TokenEOF      = lexer.TokenEOF
	TokenIf       = lexer.TokenIf
	TokenIs       = lexer.TokenIs
	TokenElse     = lexer.TokenElse
	TokenThen     = lexer.TokenThen
	TokenAssign   = lexer.TokenAssign
	TokenPlus     = lexer.TokenPlus
	TokenMinus    = lexer.TokenMinus
	TokenAsterisk = lexer.TokenAsterisk
	TokenSlash    = lexer.TokenSlash
	TokenPercent  = lexer.TokenPercent
	TokenGt       = lexer.TokenGt
	TokenLt       = lexer.TokenLt
	TokenGe       = lexer.TokenGe
	TokenLe       = lexer.TokenLe
	TokenEq       = lexer.TokenEq
	TokenAnd      = lexer.TokenAnd
	TokenOr       = lexer.TokenOr
	TokenIllegal  = lexer.TokenIllegal
	TokenIdent    = lexer.TokenIdent
	TokenNumber   = lexer.TokenNumber
	TokenString   = lexer.TokenString
	TokenTrue     = lexer.TokenTrue
	TokenFalse    = lexer.TokenFalse
	TokenLParen   = lexer.TokenLParen
	TokenRParen   = lexer.TokenRParen
	TokenComma    = lexer.TokenComma
	TokenBang     = lexer.TokenBang
)

var NewLexer = lexer.NewLexer
var lexerPool = lexer.LexerPool
