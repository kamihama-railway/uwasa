// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import "fmt"

type OpCode byte

const (
	OpPush OpCode = iota
	OpPop
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpEqual
	OpGreater
	OpLess
	OpGreaterEqual
	OpLessEqual
	OpAnd
	OpOr
	OpNot
	OpJump
	OpJumpIfFalse
	OpJumpIfTrue
	OpGetGlobal
	OpSetGlobal
	OpCall
)

func (o OpCode) String() string {
	switch o {
	case OpPush: return "PUSH"
	case OpPop: return "POP"
	case OpAdd: return "ADD"
	case OpSub: return "SUB"
	case OpMul: return "MUL"
	case OpDiv: return "DIV"
	case OpMod: return "MOD"
	case OpEqual: return "EQUAL"
	case OpGreater: return "GREATER"
	case OpLess: return "LESS"
	case OpGreaterEqual: return "GE"
	case OpLessEqual: return "LE"
	case OpAnd: return "AND"
	case OpOr: return "OR"
	case OpNot: return "NOT"
	case OpJump: return "JUMP"
	case OpJumpIfFalse: return "JIF"
	case OpJumpIfTrue: return "JIT"
	case OpGetGlobal: return "GETG"
	case OpSetGlobal: return "SETG"
	case OpCall: return "CALL"
	default: return fmt.Sprintf("UNKNOWN(%d)", o)
	}
}

type ValueType byte

const (
	ValNil ValueType = iota
	ValInt
	ValFloat
	ValBool
	ValString
)

type Value struct {
	Type   ValueType
	Int    int64
	Float  float64
	Bool   bool
	String string
}

func (v Value) ToInterface() any {
	switch v.Type {
	case ValInt: return v.Int
	case ValFloat: return v.Float
	case ValBool: return v.Bool
	case ValString: return v.String
	default: return nil
	}
}

func FromInterface(v any) Value {
	switch val := v.(type) {
	case int64: return Value{Type: ValInt, Int: val}
	case int: return Value{Type: ValInt, Int: int64(val)}
	case float64: return Value{Type: ValFloat, Float: val}
	case bool: return Value{Type: ValBool, Bool: val}
	case string: return Value{Type: ValString, String: val}
	default: return Value{Type: ValNil}
	}
}

type vmInstruction struct {
	Op  OpCode
	Arg int32
}

type RenderedBytecode struct {
	Instructions []vmInstruction
	Constants    []Value
}
