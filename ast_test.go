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
			&NumberLiteral{Value: 123.45},
			"123.45",
		},
		{
			"StringLiteral",
			&StringLiteral{Value: "hello"},
			"hello",
		},
		{
			"PrefixExpression",
			&PrefixExpression{
				Operator: "-",
				Right:    &NumberLiteral{Value: 5},
			},
			"(-5)",
		},
		{
			"InfixExpression",
			&InfixExpression{
				Left:     &Identifier{Value: "a"},
				Operator: "+",
				Right:    &Identifier{Value: "b"},
			},
			"(a + b)",
		},
		{
			"AssignExpression",
			&AssignExpression{
				Name:  &Identifier{Value: "x"},
				Value: &NumberLiteral{Value: 10},
			},
			"(x = 10)",
		},
		{
			"Simple IfExpression",
			&IfExpression{
				Condition: &InfixExpression{
					Left:     &Identifier{Value: "a"},
					Operator: "==",
					Right:    &NumberLiteral{Value: 0},
				},
				IsSimple: true,
			},
			"if (a == 0)",
		},
		{
			"If-Is-Else Expression",
			&IfExpression{
				Condition: &InfixExpression{
					Left:     &Identifier{Value: "a"},
					Operator: "==",
					Right:    &NumberLiteral{Value: 0},
				},
				Consequence: &StringLiteral{Value: "yes"},
				Alternative: &IfExpression{
					Condition: &InfixExpression{
						Left:     &Identifier{Value: "a"},
						Operator: "==",
						Right:    &NumberLiteral{Value: 1},
					},
					Consequence: &StringLiteral{Value: "ok"},
					Alternative: &StringLiteral{Value: "bad"},
				},
				IsThen: false,
			},
			"if (a == 0) is yes else if (a == 1) is ok else is bad",
		},
		{
			"If-Then Expression",
			&IfExpression{
				Condition: &InfixExpression{
					Left:     &Identifier{Value: "a"},
					Operator: ">",
					Right:    &NumberLiteral{Value: 10},
				},
				Consequence: &AssignExpression{
					Name:  &Identifier{Value: "b"},
					Value: &NumberLiteral{Value: 1},
				},
				IsThen: true,
			},
			"if (a > 10) then (b = 1)",
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
