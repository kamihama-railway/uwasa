// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"github.com/kamihama-railway/uwasa/ast"
	"github.com/kamihama-railway/uwasa/types"
)

type Node = ast.Node
type Expression = ast.Expression
type Identifier = ast.Identifier
type NumberLiteral = ast.NumberLiteral
type StringLiteral = ast.StringLiteral
type BooleanLiteral = ast.BooleanLiteral
type PrefixExpression = ast.PrefixExpression
type InfixExpression = ast.InfixExpression
type IfExpression = ast.IfExpression
type AssignExpression = ast.AssignExpression
type CallExpression = ast.CallExpression

type Value = types.Value
type ValueType = types.ValueType
type Context = types.Context
type MapContext = types.MapContext
type BuiltinFunc = types.BuiltinFunc

const (
	ValNil    = types.ValNil
	ValInt    = types.ValInt
	ValFloat  = types.ValFloat
	ValBool   = types.ValBool
	ValString = types.ValString
)

func FromInterface(v any) Value {
	return types.FromInterface(v)
}

func NewMapContext(vars map[string]any) *MapContext {
	return types.NewMapContext(vars)
}

var (
	lexerPool      = &LexerPool
	parserPool     = &ParserPool
	MapContextPool = &types.MapContextPool
	Builtins       = types.Builtins
	BufferPool     = &types.BufferPool
)

func ValToFloat64(v Value) (float64, bool) {
	return types.ValToFloat64(v)
}

func IsValTruthy(v Value) bool {
	return types.IsValTruthy(v)
}

func BoolToUint64(b bool) uint64 {
	return types.BoolToUint64(b)
}
