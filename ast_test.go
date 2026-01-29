package uwasa

import (
	"testing"
)

func TestASTString(t *testing.T) {
	tests := []struct {
		name     string
		node     Node
		expected string
	}{
		{
			"Identifier",
			&Identifier{Value: "myVar"},
			"myVar",
		},
		{
			"NumberLiteral",
			&NumberLiteral{Int64Value: 5, IsInt: true},
			"5",
		},
		{
			"BooleanLiteral",
			&BooleanLiteral{Value: true},
			"true",
		},
		{
			"PrefixExpression",
			&PrefixExpression{Operator: "-", Right: &NumberLiteral{Int64Value: 5, IsInt: true}},
			"(-5)",
		},
		{
			"InfixExpression",
			&InfixExpression{Left: &NumberLiteral{Int64Value: 5, IsInt: true}, Operator: "+", Right: &NumberLiteral{Int64Value: 10, IsInt: true}},
			"(5 + 10)",
		},
		{
			"IfExpression",
			&IfExpression{
				Condition:   &BooleanLiteral{Value: true},
				Consequence: &Identifier{Value: "x"},
				Alternative: &Identifier{Value: "y"},
			},
			"if true is x else is y",
		},
		{
			"AssignExpression",
			&AssignExpression{Name: &Identifier{Value: "x"}, Value: &NumberLiteral{Int64Value: 10, IsInt: true}},
			"(x = 10)",
		},
		{
			"CallExpression",
			&CallExpression{
				Function:  &Identifier{Value: "add"},
				Arguments: []Expression{&NumberLiteral{Int64Value: 1, IsInt: true}, &NumberLiteral{Int64Value: 2, IsInt: true}},
			},
			"add(1, 2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.node.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.node.String())
			}
		})
	}
}
