// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"math"
)

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
	// Fused Instructions
	OpEqualConst
	OpAddGlobal
	OpFusedCompareGlobalConstJumpIfFalse
	OpEqualGlobalConst
	OpGreaterGlobalConst
	OpLessGlobalConst
	OpAddGlobalGlobal
	OpGetGlobalJumpIfFalse
	OpGetGlobalJumpIfTrue
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
	case OpEqualConst: return "EQC"
	case OpAddGlobal: return "ADDG"
	case OpFusedCompareGlobalConstJumpIfFalse: return "FCG CJIF"
	case OpEqualGlobalConst: return "EQGC"
	case OpGreaterGlobalConst: return "GTGC"
	case OpLessGlobalConst: return "LTGC"
	case OpAddGlobalGlobal: return "ADDGG"
	case OpGetGlobalJumpIfFalse: return "GG JIF"
	case OpGetGlobalJumpIfTrue: return "GG JIT"
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
	Type ValueType
	Num  uint64
	Str  string
}

func (v Value) ToInterface() any {
	switch v.Type {
	case ValInt:
		return int64(v.Num)
	case ValFloat:
		return math.Float64frombits(v.Num)
	case ValBool:
		return v.Num != 0
	case ValString:
		return v.Str
	default:
		return nil
	}
}

func FromInterface(v any) Value {
	switch val := v.(type) {
	case int64:
		return Value{Type: ValInt, Num: uint64(val)}
	case int:
		return Value{Type: ValInt, Num: uint64(val)}
	case float64:
		return Value{Type: ValFloat, Num: math.Float64bits(val)}
	case bool:
		if val {
			return Value{Type: ValBool, Num: 1}
		}
		return Value{Type: ValBool, Num: 0}
	case string:
		return Value{Type: ValString, Str: val}
	default:
		return Value{Type: ValNil}
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
