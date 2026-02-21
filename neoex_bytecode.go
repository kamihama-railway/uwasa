// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import "fmt"

type NeoOpCode byte

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
	NeoOpAddConstGlobal
	NeoOpFusedCompareGlobalConstJumpIfFalse
	NeoOpFusedGreaterGlobalConstJumpIfFalse
	NeoOpFusedLessGlobalConstJumpIfFalse
	NeoOpEqualGlobalConst
	NeoOpGreaterGlobalConst
	NeoOpLessGlobalConst
	NeoOpEqualC
	NeoOpGreaterC
	NeoOpLessC
	NeoOpAddGlobalGlobal
	NeoOpSubGlobalGlobal
	NeoOpMulGlobalGlobal
	NeoOpAddGC // Global + Const
	NeoOpSubGC
	NeoOpMulGC
	NeoOpDivGC
	NeoOpSubCG
	NeoOpMulCG
	NeoOpDivCG
	NeoOpGetGlobalJumpIfFalse
	NeoOpGetGlobalJumpIfTrue
	NeoOpConcat
	NeoOpConcat2
	NeoOpConcatGC
	NeoOpConcatCG
	NeoOpAddInt
	NeoOpAddFloat
	NeoOpSubInt
	NeoOpSubFloat
	NeoOpMulInt
	NeoOpMulFloat
	NeoOpAddC
	NeoOpSubC
	NeoOpMulC
	NeoOpDivC
	NeoOpMapGet
	NeoOpMapSet
	NeoOpMapHas
	NeoOpMapDel
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
	case NeoOpAddConstGlobal: return "ADDCG"
	case NeoOpFusedCompareGlobalConstJumpIfFalse: return "FCG EQJIF"
	case NeoOpFusedGreaterGlobalConstJumpIfFalse: return "FCG GTJIF"
	case NeoOpFusedLessGlobalConstJumpIfFalse: return "FCG LTJIF"
	case NeoOpEqualGlobalConst: return "EQGC"
	case NeoOpGreaterGlobalConst: return "GTGC"
	case NeoOpLessGlobalConst: return "LTGC"
	case NeoOpEqualC: return "EQC"
	case NeoOpGreaterC: return "GTC"
	case NeoOpLessC: return "LTC"
	case NeoOpAddGlobalGlobal: return "ADDGG"
	case NeoOpSubGlobalGlobal: return "SUBGG"
	case NeoOpMulGlobalGlobal: return "MULGG"
	case NeoOpAddGC: return "ADDGC"
	case NeoOpSubGC: return "SUBGC"
	case NeoOpMulGC: return "MULGC"
	case NeoOpDivGC: return "DIVGC"
	case NeoOpSubCG: return "SUBCG"
	case NeoOpMulCG: return "MULCG"
	case NeoOpDivCG: return "DIVCG"
	case NeoOpGetGlobalJumpIfFalse: return "GG JIF"
	case NeoOpGetGlobalJumpIfTrue: return "GG JIT"
	case NeoOpConcat: return "CONCAT"
	case NeoOpConcat2: return "CONCAT2"
	case NeoOpConcatGC: return "CONCATGC"
	case NeoOpConcatCG: return "CONCATCG"
	case NeoOpAddInt: return "ADD_I"
	case NeoOpAddFloat: return "ADD_F"
	case NeoOpSubInt: return "SUB_I"
	case NeoOpSubFloat: return "SUB_F"
	case NeoOpMulInt: return "MUL_I"
	case NeoOpMulFloat: return "MUL_F"
	case NeoOpAddC: return "ADDC"
	case NeoOpSubC: return "SUBC"
	case NeoOpMulC: return "MULC"
	case NeoOpDivC: return "DIVC"
	case NeoOpMapGet: return "MGET"
	case NeoOpMapSet: return "MSET"
	case NeoOpMapHas: return "MHAS"
	case NeoOpMapDel: return "MDEL"
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
