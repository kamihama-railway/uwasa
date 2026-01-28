// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type OpCode byte

const (
	OpConstant OpCode = iota
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpMinus
	OpEqual
	OpGreater
	OpLess
	OpGreaterEqual
	OpLessEqual
	OpPop
	OpGetGlobal
	OpSetGlobal
	OpJump
	OpJumpIfFalse
	OpJumpIfTrue
	OpCall
	OpCallResolved
	OpConcat
	OpToBool
	OpEqualConst
	OpNotEqualConst
	OpLessConst
	OpGreaterConst
	OpLessEqualConst
	OpGreaterEqualConst
	OpJumpIfFalsePop
	OpFusedCompareGlobalConstJumpIfFalse
	OpFusedGreaterGlobalConstJumpIfFalse
	OpFusedLessGlobalConstJumpIfFalse
	OpFusedGreaterEqualGlobalConstJumpIfFalse
	OpFusedLessEqualGlobalConstJumpIfFalse
	OpFusedGreaterConstJumpIfFalsePop
	OpFusedLessConstJumpIfFalsePop
	OpFusedGreaterEqualConstJumpIfFalsePop
	OpFusedLessEqualConstJumpIfFalsePop
	OpAddGlobal
	OpSubGlobal
	OpMulGlobal
	OpDivGlobal
	OpAddConst
	OpSubConst
	OpMulConst
	OpDivConst
)

type Definition struct {
	Name          string
	OperandWidths []int
}

var definitions = map[OpCode]*Definition{
	OpConstant:     {"OpConstant", []int{2}}, // 2-byte index to constant pool
	OpAdd:          {"OpAdd", []int{}},
	OpSub:          {"OpSub", []int{}},
	OpMul:          {"OpMul", []int{}},
	OpDiv:          {"OpDiv", []int{}},
	OpMod:          {"OpMod", []int{}},
	OpMinus:        {"OpMinus", []int{}},
	OpEqual:        {"OpEqual", []int{}},
	OpGreater:      {"OpGreater", []int{}},
	OpLess:         {"OpLess", []int{}},
	OpGreaterEqual: {"OpGreaterEqual", []int{}},
	OpLessEqual:    {"OpLessEqual", []int{}},
	OpPop:          {"OpPop", []int{}},
	OpGetGlobal:    {"OpGetGlobal", []int{2}}, // 2-byte index to name constant
	OpSetGlobal:    {"OpSetGlobal", []int{2}}, // 2-byte index to name constant
	OpJump:         {"OpJump", []int{2}},
	OpJumpIfFalse:  {"OpJumpIfFalse", []int{2}},
	OpJumpIfTrue:   {"OpJumpIfTrue", []int{2}},
	OpCall:              {"OpCall", []int{1}}, // 1-byte number of arguments
	OpCallResolved:      {"OpCallResolved", []int{1, 2}}, // 1-byte numArgs, 2-byte index to builtins
	OpConcat:            {"OpConcat", []int{1}},          // 1-byte number of arguments
	OpToBool:            {"OpToBool", []int{}},
	OpEqualConst:        {"OpEqualConst", []int{2}},
	OpNotEqualConst:     {"OpNotEqualConst", []int{2}},
	OpLessConst:         {"OpLessConst", []int{2}},
	OpGreaterConst:      {"OpGreaterConst", []int{2}},
	OpLessEqualConst:    {"OpLessEqualConst", []int{2}},
	OpGreaterEqualConst: {"OpGreaterEqualConst", []int{2}},
	OpJumpIfFalsePop:    {"OpJumpIfFalsePop", []int{2}},
	OpFusedCompareGlobalConstJumpIfFalse:      {"OpFusedCompareGlobalConstJumpIfFalse", []int{2, 2, 2}},
	OpFusedGreaterGlobalConstJumpIfFalse:      {"OpFusedGreaterGlobalConstJumpIfFalse", []int{2, 2, 2}},
	OpFusedLessGlobalConstJumpIfFalse:         {"OpFusedLessGlobalConstJumpIfFalse", []int{2, 2, 2}},
	OpFusedGreaterEqualGlobalConstJumpIfFalse: {"OpFusedGreaterEqualGlobalConstJumpIfFalse", []int{2, 2, 2}},
	OpFusedLessEqualGlobalConstJumpIfFalse:    {"OpFusedLessEqualGlobalConstJumpIfFalse", []int{2, 2, 2}},
	OpFusedGreaterConstJumpIfFalsePop:         {"OpFusedGreaterConstJumpIfFalsePop", []int{2, 2}},
	OpFusedLessConstJumpIfFalsePop:            {"OpFusedLessConstJumpIfFalsePop", []int{2, 2}},
	OpFusedGreaterEqualConstJumpIfFalsePop:    {"OpFusedGreaterEqualConstJumpIfFalsePop", []int{2, 2}},
	OpFusedLessEqualConstJumpIfFalsePop:       {"OpFusedLessEqualConstJumpIfFalsePop", []int{2, 2}},
	OpAddGlobal: {"OpAddGlobal", []int{2}},
	OpSubGlobal: {"OpSubGlobal", []int{2}},
	OpMulGlobal: {"OpMulGlobal", []int{2}},
	OpDivGlobal: {"OpDivGlobal", []int{2}},
	OpAddConst:  {"OpAddConst", []int{2}},
	OpSubConst:  {"OpSubConst", []int{2}},
	OpMulConst:  {"OpMulConst", []int{2}},
	OpDivConst:  {"OpDivConst", []int{2}},
}

func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[OpCode(op)]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}
	return def, nil
}

func Make(op OpCode, operands ...int) []byte {
	def, ok := definitions[op]
	if !ok {
		return []byte{}
	}

	instructionLen := 1
	for _, w := range def.OperandWidths {
		instructionLen += w
	}

	instruction := make([]byte, instructionLen)
	instruction[0] = byte(op)

	offset := 1
	for i, o := range operands {
		width := def.OperandWidths[i]
		switch width {
		case 2:
			binary.BigEndian.PutUint16(instruction[offset:], uint16(o))
		case 1:
			instruction[offset] = byte(o)
		}
		offset += width
	}

	return instruction
}

func ReadOperands(def *Definition, ins []byte) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))
	offset := 0

	for i, width := range def.OperandWidths {
		switch width {
		case 2:
			operands[i] = int(binary.BigEndian.Uint16(ins[offset:]))
		case 1:
			operands[i] = int(ins[offset])
		}
		offset += width
	}

	return operands, offset
}

type Instructions []byte

func (ins Instructions) String() string {
	var out bytes.Buffer

	i := 0
	for i < len(ins) {
		def, err := Lookup(ins[i])
		if err != nil {
			fmt.Fprintf(&out, "ERROR: %s\n", err)
			continue
		}

		operands, read := ReadOperands(def, ins[i+1:])

		fmt.Fprintf(&out, "%04d %s\n", i, ins.fmtInstruction(def, operands))

		i += 1 + read
	}

	return out.String()
}

func (ins Instructions) fmtInstruction(def *Definition, operands []int) string {
	operandCount := len(def.OperandWidths)

	if len(operands) != operandCount {
		return fmt.Sprintf("ERROR: operand len %d does not match defined %d\n",
			len(operands), operandCount)
	}

	switch operandCount {
	case 0:
		return def.Name
	case 1:
		return fmt.Sprintf("%s %d", def.Name, operands[0])
	}

	return fmt.Sprintf("ERROR: unhandled operandCount for %s\n", def.Name)
}
