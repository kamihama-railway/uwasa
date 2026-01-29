// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
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
	OpConcat
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
	case OpConcat: return "CONCAT"
	default: return fmt.Sprintf("UNKNOWN(%d)", o)
	}
}

type VmInstruction struct {
	Op  OpCode
	Arg int32
}

type RenderedBytecode struct {
	Instructions []VmInstruction
	Constants    []Value
}
