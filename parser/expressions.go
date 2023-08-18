package parser

import (
	"fmt"
	"strconv"

	"github.com/gearsdatapacks/libra/lexer/token"
	"github.com/gearsdatapacks/libra/parser/ast"
)

func (p *parser) parseExpression() ast.Expression {
	return p.parseAssignmentExpression()
}

// Orders of precedence

// Assignment
// Logical operators
// Comparison
// Addition/Subtraction
// Multiplication/Division
// Member access
// Function call
// Unary operation
// Literal

func (p *parser) parseAssignmentExpression() ast.Expression {
	assignee := p.parseLogicalExpression()

	if !p.canContinue() || p.next().Type != token.ASSIGNMENT_OPERATOR {
		return assignee
	}

	p.consume()

	value := p.parseAssignmentExpression()

	return &ast.AssignmentExpression{
		Assignee: assignee,
		Value:    value,
		BaseNode: &ast.BaseNode{Token: assignee.GetToken()},
	}
}

func (p *parser) parseLogicalExpression() ast.Expression {
	left := p.parseComparisonExpression()

	for p.canContinue() && p.next().Type == token.LOGICAL_OPERATOR {
		operator := p.consume().Value
		right := p.parseComparisonExpression()
		left = &ast.BinaryOperation{
			Left:     left,
			Operator: operator,
			Right:    right,
			BaseNode: &ast.BaseNode{Token: left.GetToken()},
		}
	}

	return left
}

func (p *parser) parseComparisonExpression() ast.Expression {
	left := p.parseAdditiveExpression()

	for p.canContinue() && p.next().Type == token.COMPARISON_OPERATOR {
		operator := p.consume().Value
		right := p.parseAdditiveExpression()
		left = &ast.BinaryOperation{
			Left:     left,
			Operator: operator,
			Right:    right,
			BaseNode: &ast.BaseNode{Token: left.GetToken()},
		}
	}

	return left
}

func (p *parser) parseAdditiveExpression() ast.Expression {
	left := p.parseMultiplicativeExpression()

	for p.canContinue() && p.next().Type == token.ADDITIVE_OPERATOR {
		operator := p.consume().Value
		right := p.parseMultiplicativeExpression()
		left = &ast.BinaryOperation{
			Left:     left,
			Operator: operator,
			Right:    right,
			BaseNode: &ast.BaseNode{Token: left.GetToken()},
		}
	}

	return left
}

func (p *parser) parseMultiplicativeExpression() ast.Expression {
	left := p.parseLiteral()

	for p.canContinue() && p.next().Type == token.MULTIPLICATIVE_OPERATOR {
		operator := p.consume().Value
		right := p.parseLiteral()
		left = &ast.BinaryOperation{
			Left:     left,
			Operator: operator,
			Right:    right,
			BaseNode: &ast.BaseNode{Token: left.GetToken()},
		}
	}

	return left
}

func (p *parser) parseLiteral() ast.Expression {
	switch p.next().Type {
	case token.INTEGER:
		tok := p.consume()
		value, _ := strconv.ParseInt(tok.Value, 10, 32)
		return &ast.IntegerLiteral{
			Value:    int(value),
			BaseNode: &ast.BaseNode{Token: tok},
		}

	case token.FLOAT:
		tok := p.consume()
		value, _ := strconv.ParseFloat(tok.Value, 64)
		return &ast.FloatLiteral{
			Value:    value,
			BaseNode: &ast.BaseNode{Token: tok},
		}

	case token.IDENTIFIER:
		if p.isKeyword("true") {
			tok := p.consume()
			return &ast.BooleanLiteral{
				Value:    true,
				BaseNode: &ast.BaseNode{Token: tok},
			}
		}

		if p.isKeyword("false") {
			tok := p.consume()
			return &ast.BooleanLiteral{
				Value:    false,
				BaseNode: &ast.BaseNode{Token: tok},
			}
		}

		if p.isKeyword("null") {
			tok := p.consume()
			return &ast.NullLiteral{
				BaseNode: &ast.BaseNode{Token: tok},
			}
		}

		tok := p.consume()
		return &ast.Identifier{
			Symbol:   tok.Value,
			BaseNode: &ast.BaseNode{Token: tok},
		}

	case token.LEFT_PAREN:
		p.consume()
		p.bracketLevel++
		expression := p.parseExpression()
		p.expect(token.RIGHT_PAREN, "Expected closing parentheses after bracketed expression, got %q")
		p.bracketLevel--
		return expression
	default:
		p.error(fmt.Sprintf("Expected expression, got %q", p.next().Value), p.next())
		return &ast.IntegerLiteral{}
	}
}
