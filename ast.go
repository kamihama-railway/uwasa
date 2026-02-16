// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import "strings"

import "fmt"

type Node interface {
	String() string
}

type Expression interface {
	Node
	expressionNode()
}

type Identifier struct {
	Value string
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) String() string  { return i.Value }

type NumberLiteral struct {
	Int64Value   int64
	Float64Value float64
	IsInt        bool
}

func (n *NumberLiteral) expressionNode() {}
func (n *NumberLiteral) String() string {
	if n.IsInt {
		return fmt.Sprintf("%d", n.Int64Value)
	}
	return fmt.Sprintf("%g", n.Float64Value)
}

type StringLiteral struct {
	Value string
}

func (s *StringLiteral) expressionNode() {}
func (s *StringLiteral) String() string  { return s.Value }

type BooleanLiteral struct {
	Value bool
}

func (b *BooleanLiteral) expressionNode() {}
func (b *BooleanLiteral) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

type PrefixExpression struct {
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode() {}
func (pe *PrefixExpression) String() string {
	return "(" + pe.Operator + pe.Right.String() + ")"
}

type InfixExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode() {}
func (ie *InfixExpression) String() string {
	return "(" + ie.Left.String() + " " + ie.Operator + " " + ie.Right.String() + ")"
}

type IfExpression struct {
	Condition   Expression
	Consequence Expression // for 'is' or 'then'
	Alternative Expression // for 'else'
	IsThen      bool       // true if 'then', false if 'is'
	IsSimple    bool       // true if only 'if <cond>'
}

func (ie *IfExpression) expressionNode() {}
func (ie *IfExpression) String() string {
	out := "if " + ie.Condition.String()
	if ie.IsSimple {
		return out
	}
	if ie.IsThen {
		out += " then " + ie.Consequence.String()
	} else {
		out += " is " + ie.Consequence.String()
	}
	if ie.Alternative != nil {
		out += " else "
		if !ie.IsThen {
			if _, ok := ie.Alternative.(*IfExpression); !ok {
				out += "is "
			}
		}
		out += ie.Alternative.String()
	}
	return out
}

type AssignExpression struct {
	Name  *Identifier
	Value Expression
}

func (ae *AssignExpression) expressionNode() {}
func (ae *AssignExpression) String() string {
	return "(" + ae.Name.String() + " = " + ae.Value.String() + ")"
}

type CallExpression struct {
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode() {}
func (ce *CallExpression) String() string {
	var out strings.Builder
	out.WriteString(ce.Function.String() + "(")
	for i, arg := range ce.Arguments {
		out.WriteString(arg.String())
		if i < len(ce.Arguments)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString(")")
	return out.String()
}
