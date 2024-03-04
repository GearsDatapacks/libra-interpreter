package parser

import (
	"github.com/gearsdatapacks/libra/lexer/token"
	"github.com/gearsdatapacks/libra/parser/ast"
)

func (p *parser) parseType() ast.TypeExpression {
	ty := p.parsePostfixType()

	if p.next().Kind == token.PIPE {
		types := []ast.TypeExpression{ty}

		for p.canContinue() && p.next().Kind == token.PIPE {
			p.consume()
			types = append(types, p.parsePostfixType())
		}

		ty = &ast.Union{
			Types: types,
		}
	}

	return ty
}

func (p *parser) parsePostfixType() ast.TypeExpression {
	left := p.parsePrefixType()

	done := false
	for !done {
		switch p.next().Kind {
		case token.LEFT_SQUARE:
			left = p.parseArrayType(left)
		case token.QUESTION:
			left = &ast.OptionType{
				Type:     left,
				Question: p.consume(),
			}
		case token.BANG:
			left = &ast.ErrorType{
				Type: left,
				Bang: p.consume(),
			}
		default:
			done = true
		}
	}

	return left
}

func (p *parser) parsePrefixType() ast.TypeExpression {
	switch p.next().Kind {
	case token.STAR:
		return p.parsePointerType()
	default:
		return p.parsePrimaryType()
	}
}

func (p *parser) parsePrimaryType() ast.TypeExpression {
	switch p.next().Kind {
	case token.IDENTIFIER:
		return p.parseTypeName()
	case token.BANG:
		return &ast.ErrorType{
			Type: nil,
			Bang: p.consume(),
		}
	default:
		p.Diagnostics.ReportExpectedType(p.next().Span, p.next().Kind)
		return &ast.ErrorNode{}
	}
}

func (p *parser) parseArrayType(ty ast.TypeExpression) ast.TypeExpression {
	leftSquare := p.consume()
	var count ast.Expression
	if p.next().Kind != token.RIGHT_SQUARE {
		count = p.parseExpression()
	}
	rightSquare := p.expect(token.RIGHT_SQUARE)

	return &ast.ArrayType{
		Type:        ty,
		LeftSquare:  leftSquare,
		Count:       count,
		RightSquare: rightSquare,
	}
}

func (p *parser) parsePointerType() ast.TypeExpression {
	star := p.consume()
	var mut *token.Token
	if p.isKeyword("mut") {
		tok := p.consume()
		mut = &tok
	}
	ty := p.parsePrefixType()

	return &ast.PointerType{
		Star: star,
		Mut:  mut,
		Type: ty,
	}
}

func (p *parser) parseTypeName() ast.TypeExpression {
	name := p.expect(token.IDENTIFIER)

	return &ast.TypeName{
		Name: name,
	}
}
