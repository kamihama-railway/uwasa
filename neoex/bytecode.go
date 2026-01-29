package neoex

import (
	"github.com/kamihama-railway/uwasa/types"
)

type OpCode byte

const (
	OpLoadConst OpCode = iota
	OpGetGlobal
	OpSetGlobal
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
	OpCall
	OpConcat
	OpReturn

	// Fused Instructions
	OpEqualGlobalConst
	OpAddGlobalConst
	OpGetGlobalJumpIfFalse
	OpFusedCompareGlobalConstJumpIfFalse
	OpAddGlobalGlobal
	OpSubGlobalGlobal
)

type Instruction struct {
	Op   OpCode
	Dest uint8
	Src1 uint8
	Src2 uint8
	Arg  int32
}

type Bytecode struct {
	Instructions []Instruction
	Constants    []types.Value
}
