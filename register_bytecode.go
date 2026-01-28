// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import "fmt"

type ROpCode byte

const (
	ROpLoadConst ROpCode = iota
	ROpGetGlobal
	ROpSetGlobal
	ROpMove
	ROpAdd
	ROpSub
	ROpMul
	ROpDiv
	ROpMod
	ROpEqual
	ROpGreater
	ROpLess
	ROpGreaterEqual
	ROpLessEqual
	ROpAnd
	ROpOr
	ROpNot
	ROpJump
	ROpJumpIfFalse
	ROpJumpIfTrue
	ROpCall
	ROpConcat
	ROpReturn
)

func (o ROpCode) String() string {
	switch o {
	case ROpLoadConst: return "LOADC"
	case ROpGetGlobal: return "GETG"
	case ROpSetGlobal: return "SETG"
	case ROpMove: return "MOVE"
	case ROpAdd: return "ADD"
	case ROpSub: return "SUB"
	case ROpMul: return "MUL"
	case ROpDiv: return "DIV"
	case ROpMod: return "MOD"
	case ROpEqual: return "EQ"
	case ROpGreater: return "GT"
	case ROpLess: return "LT"
	case ROpGreaterEqual: return "GE"
	case ROpLessEqual: return "LE"
	case ROpAnd: return "AND"
	case ROpOr: return "OR"
	case ROpNot: return "NOT"
	case ROpJump: return "JUMP"
	case ROpJumpIfFalse: return "JIF"
	case ROpJumpIfTrue: return "JIT"
	case ROpCall: return "CALL"
	case ROpConcat: return "CONCAT"
	case ROpReturn: return "RET"
	default: return fmt.Sprintf("RUNKNOWN(%d)", o)
	}
}

type regInstruction struct {
	Op   ROpCode
	Dest uint8
	Src1 uint8
	Src2 uint8
	Arg  int32
}

type RegisterBytecode struct {
	Instructions []regInstruction
	Constants    []Value
	MaxRegisters uint8
}
