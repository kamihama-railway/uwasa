// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import "fmt"

type NeoOpCode uint16

const (
	NeoOpPush NeoOpCode = iota
	NeoOpPop
	NeoOpAdd
	NeoOpSub
	NeoOpMul
	NeoOpDiv
	NeoOpMod
	NeoOpEqual
	NeoOpGreater
	NeoOpLess
	NeoOpGreaterEqual
	NeoOpLessEqual
	NeoOpAnd
	NeoOpOr
	NeoOpNot
	NeoOpJump
	NeoOpJumpIfFalse
	NeoOpJumpIfTrue
	NeoOpGetGlobal
	NeoOpSetGlobal
	NeoOpCall
	NeoOpEqualConst
	NeoOpAddGlobal
	NeoOpFusedCompareGlobalConstJumpIfFalse
	NeoOpEqualGlobalConst
	NeoOpGreaterGlobalConst
	NeoOpLessGlobalConst
	NeoOpAddGlobalGlobal
	NeoOpGetGlobalJumpIfFalse
	NeoOpGetGlobalJumpIfTrue
	NeoOpConcat
	NeoOpReturn // New for NeoEx to signal end of execution if needed
)

func (o NeoOpCode) String() string {
	switch o {
	case NeoOpPush: return "PUSH"
	case NeoOpPop: return "POP"
	case NeoOpAdd: return "ADD"
	case NeoOpSub: return "SUB"
	case NeoOpMul: return "MUL"
	case NeoOpDiv: return "DIV"
	case NeoOpMod: return "MOD"
	case NeoOpEqual: return "EQUAL"
	case NeoOpGreater: return "GREATER"
	case NeoOpLess: return "LESS"
	case NeoOpGreaterEqual: return "GE"
	case NeoOpLessEqual: return "LE"
	case NeoOpAnd: return "AND"
	case NeoOpOr: return "OR"
	case NeoOpNot: return "NOT"
	case NeoOpJump: return "JUMP"
	case NeoOpJumpIfFalse: return "JIF"
	case NeoOpJumpIfTrue: return "JIT"
	case NeoOpGetGlobal: return "GETG"
	case NeoOpSetGlobal: return "SETG"
	case NeoOpCall: return "CALL"
	case NeoOpEqualConst: return "EQC"
	case NeoOpAddGlobal: return "ADDG"
	case NeoOpFusedCompareGlobalConstJumpIfFalse: return "FCG CJIF"
	case NeoOpEqualGlobalConst: return "EQGC"
	case NeoOpGreaterGlobalConst: return "GTGC"
	case NeoOpLessGlobalConst: return "LTGC"
	case NeoOpAddGlobalGlobal: return "ADDGG"
	case NeoOpGetGlobalJumpIfFalse: return "GG JIF"
	case NeoOpGetGlobalJumpIfTrue: return "GG JIT"
	case NeoOpConcat: return "CONCAT"
	case NeoOpReturn: return "RET"
	default: return fmt.Sprintf("NEO_UNKNOWN(%d)", o)
	}
}

type neoInstruction struct {
	Op  NeoOpCode
	Arg int32
}

type NeoBytecode struct {
	Instructions []neoInstruction
	Constants    []Value
}
