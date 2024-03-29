package parser

import (
	"fmt"

	"github.com/gearsdatapacks/libra/lexer/token"
	"github.com/gearsdatapacks/libra/parser/ast"
)

func (p *parser) parseStatement(inline ...bool) (ast.Statement, error) {
	var statement ast.Statement
	var err error = nil

	if p.needsNewline() && !p.eof() && !p.next().LeadingNewline {
		return nil, p.error(fmt.Sprintf("Expected new line after statement, got %q", p.next().Value), p.next())
	}

	if p.isKeyword("var") || p.isKeyword("const") {
		statement, err = p.parseVariableDeclaration()
	} else if p.isKeyword("fn") {
		statement, err = p.parseFunctionDeclaration()
	} else if p.isKeyword("return") {
		statement, err = p.parseReturnStatement()
	} else if p.isKeyword("if") {
		statement, err = p.parseIfStatement()
	} else if p.isKeyword("else") {
		return nil, p.error("Cannot use else statement without preceding if", p.next())
	} else if p.isKeyword("while") {
		statement, err = p.parseWhileLoop()
	} else if p.isKeyword("for") {
		statement, err = p.parseForLoop()
	} else if p.isKeyword("struct") {
		statement, err = p.parseStructDeclaration()
	} else if p.isKeyword("interface") {
		statement, err = p.parseInterfaceDeclaration()
	} else if p.isKeyword("type") {
		statement, err = p.parseTypeDeclaration()
	} else if p.isKeyword("import") {
		statement, err = p.parseImportStatement()
	} else if p.isKeyword("pub") {
		statement, err = p.parseExportStatement()
	} else if p.isKeyword("enum") || p.isKeyword("union") {
		statement, err = p.parseEnumDeclaration()
	} else {
		statement, err = p.parseExpressionStatement()
	}

	if err != nil {
		return nil, err
	}

	if len(inline) != 0 && inline[0] {
		return statement, nil
	}

	p.requireNewline = true
	return statement, nil
}

func (p *parser) parseExpressionStatement() (ast.Statement, error) {
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &ast.ExpressionStatement{
		Expression: expr,
		BaseNode:   ast.BaseNode{Token: expr.GetToken()},
	}, nil
}

func (p *parser) parseVariableDeclaration() (ast.Statement, error) {
	tok := p.consume()
	isConstant := tok.Value == "const"
	name, err := p.expect(
		token.IDENTIFIER,
		"Invalid variable name %q",
	)
	if err != nil {
		return nil, err
	}

	var dataType ast.TypeExpression = &ast.InferType{}

	if p.canContinue() && p.next().Type == token.COLON {
		p.consume()
		dataType, err = p.parseType()
		if err != nil {
			return nil, err
		}
	}

	if !p.canContinue() || p.next().Type != token.EQUALS {
		if isConstant {
			return nil, p.error(fmt.Sprintf("Cannot leave constant %q uninitialised", name.Value), p.next())
		}

		if dataType.Type() == "Infer" {
			return nil, p.error(fmt.Sprintf("Cannot declare uninitialised variable %q without type annotation", name.Value), p.next())
		}

		return &ast.VariableDeclaration{
			Constant: isConstant,
			Name:     name.Value,
			BaseNode: ast.BaseNode{Token: tok},
			Value:    nil,
			DataType: dataType,
		}, nil
	}

	_, err = p.expect(
		token.EQUALS,
		"Missing initialiser in variable declaration",
	)
	if err != nil {
		return nil, err
	}

	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	p.usedSymbols = append(p.usedSymbols, name.Value)

	return &ast.VariableDeclaration{
		Constant: isConstant,
		Name:     name.Value,
		BaseNode: ast.BaseNode{Token: tok},
		Value:    value,
		DataType: dataType,
	}, nil
}

func (p *parser) parseFunctionDeclaration() (ast.Statement, error) {
	tok := p.consume()

	var methodOf ast.TypeExpression = nil
	if p.next().Type == token.LEFT_PAREN {
		p.consume()
		var err error
		methodOf, err = p.parseType()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(token.RIGHT_PAREN, "Unexpected %q, expected ')'")
		if err != nil {
			return nil, err
		}
	}

	name, err := p.expect(token.IDENTIFIER, "Invalid function name %q")
	if err != nil {
		return nil, err
	}

	p.usedSymbols = append(p.usedSymbols, name.Value)

	parameters, err := p.parseParameterList()
	if err != nil {
		return nil, err
	}

	outerSymbols := make([]string, len(p.usedSymbols))
	copy(outerSymbols, p.usedSymbols)

	for _, param := range parameters {
		p.usedSymbols = append(p.usedSymbols, param.Name)
	}

	var returnType ast.TypeExpression = &ast.VoidType{}

	if p.next().Type == token.COLON {
		p.consume()
		returnType, err = p.parseType()
		if err != nil {
			return nil, err
		}
	}

	if p.next().Type != token.LEFT_BRACE {
		return nil, p.error("Expected type annotation or function body", p.next())
	}

	code, err := p.parseCodeBlock()
	if err != nil {
		return nil, err
	}

	p.usedSymbols = outerSymbols

	return &ast.FunctionDeclaration{
		Name:       name.Value,
		Parameters: parameters,
		Body:       code,
		ReturnType: returnType,
		BaseNode:   ast.BaseNode{Token: tok},
		MethodOf:   methodOf,
	}, nil
}

func (p *parser) parseReturnStatement() (ast.Statement, error) {
	token := p.consume()

	var value ast.Expression = &ast.VoidValue{}

	if p.canContinue() {
		var err error = nil
		value, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}

	return &ast.ReturnStatement{
		Value:    value,
		BaseNode: ast.BaseNode{Token: token},
	}, nil
}

func (p *parser) parseIfStatement() (*ast.IfStatement, error) {
	tok := p.consume()
	noBraces := p.noBraces
	p.noBraces = true

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	p.noBraces = noBraces

	body, err := p.parseCodeBlock()
	if err != nil {
		return nil, err
	}
	var elseStatement ast.IfElseStatement = nil

	if p.isKeyword("else") {
		elseToken := p.consume()
		if p.next().Type == token.LEFT_BRACE {
			code, err := p.parseCodeBlock()
			if err != nil {
				return nil, err
			}
			elseStatement = &ast.ElseStatement{
				Body:     code,
				BaseNode: ast.BaseNode{Token: elseToken},
			}
		} else {
			elseStatement, err = p.parseIfStatement()
		}

		if err != nil {
			return nil, err
		}
	}

	return &ast.IfStatement{
		Condition: condition,
		Body:      body,
		BaseNode:  ast.BaseNode{Token: tok},
		Else:      elseStatement,
	}, nil
}

func (p *parser) parseWhileLoop() (ast.Statement, error) {
	tok := p.consume()

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	body, err := p.parseCodeBlock()
	if err != nil {
		return nil, err
	}

	return &ast.WhileLoop{
		Condition: condition,
		Body:      body,
		BaseNode:  ast.BaseNode{Token: tok},
	}, nil
}

func (p *parser) parseForLoop() (ast.Statement, error) {
	tok := p.consume()

	outerSymbols := make([]string, len(p.usedSymbols))
	copy(outerSymbols, p.usedSymbols)

	initial, err := p.parseStatement(true)
	if err != nil {
		return nil, err
	}

	// _, err = p.expect(token.SEMICOLON, "Unexpected %q, expecting ';'")
	// if err != nil {
	// 	return nil, err
	// }

	if !p.eof() && !p.next().LeadingNewline {
		return nil, p.error(fmt.Sprintf("Expected new line after statement, got %q", p.next().Value), p.next())
	}

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// _, err = p.expect(token.SEMICOLON, "Unexpected %q, expecting ';'")
	// if err != nil {
	// 	return nil, err
	// }

	update, err := p.parseStatement()
	if err != nil {
		return nil, err
	}

	body, err := p.parseCodeBlock()
	if err != nil {
		return nil, err
	}

	p.usedSymbols = outerSymbols

	return &ast.ForLoop{
		Initial:   initial,
		Condition: condition,
		Update:    update,
		Body:      body,
		BaseNode:  ast.BaseNode{Token: tok},
	}, nil
}

func (p *parser) parseStructDeclaration() (ast.Statement, error) {
	tok := p.consume()

	name, err := p.expect(token.IDENTIFIER, "Invalid struct name %q")
	if err != nil {
		return nil, err
	}

	if p.next().LeadingNewline {
		return &ast.UnitStructDeclaration{
			BaseNode: ast.BaseNode{Token: tok},
			Name:     name.Value,
		}, nil
	}

	if p.next().Type == token.LEFT_PAREN {
		return p.parseTupleStructDeclaration(tok, name.Value)
	}

	_, err = p.expect(token.LEFT_BRACE, "Expected struct body")
	if err != nil {
		return nil, err
	}

	members := map[string]ast.StructField{}

	for !p.eof() && p.next().Type != token.RIGHT_BRACE {
		name, field, err := p.parseStructField()
		if err != nil {
			return nil, err
		}
		members[name] = field

		if p.next().Type != token.RIGHT_BRACE {
			_, err = p.expect(token.COMMA, "Expected comma or end of struct body")
			if err != nil {
				return nil, err
			}
		}
	}

	_, err = p.expect(token.RIGHT_BRACE, "Unexpected EOF, expected '}'")
	if err != nil {
		return nil, err
	}

	return &ast.StructDeclaration{
		BaseNode: ast.BaseNode{Token: tok},
		Name:     name.Value,
		Members:  members,
	}, nil
}

func (p *parser) parseStructField() (string, ast.StructField, error) {
	exported := false
	if p.isKeyword("pub") {
		p.consume()
		exported = true
	}

	memberName, err := p.expect(token.IDENTIFIER, "Expected closing brace or struct member")
	if err != nil {
		return "", ast.StructField{}, err
	}
	_, err = p.expect(token.COLON, "Expected type annotation")
	if err != nil {
		return "", ast.StructField{}, err
	}
	memberType, err := p.parseType()
	if err != nil {
		return "", ast.StructField{}, err
	}
	return memberName.Value, ast.StructField{Exported: exported, Type: memberType}, nil
}

func (p *parser) parseTupleStructDeclaration(tok token.Token, name string) (ast.Statement, error) {
	p.consume()

	members := []ast.TypeExpression{}

	for p.next().Type != token.RIGHT_PAREN {
		expr, err := p.parseType()
		if err != nil {
			return nil, err
		}
		members = append(members, expr)
		if p.next().Type != token.RIGHT_PAREN {
			_, err := p.expect(token.COMMA, "Expected comma or end of tuple struct body")
			if err != nil {
				return nil, err
			}
		}
	}

	p.consume()
	return &ast.TupleStructDeclaration{
		BaseNode: ast.BaseNode{Token: tok},
		Name:     name,
		Members:  members,
	}, nil
}

func (p *parser) parseInterfaceDeclaration() (ast.Statement, error) {
	tok := p.consume()

	name, err := p.expect(token.IDENTIFIER, "Invalid interface name %q")
	if err != nil {
		return nil, err
	}

	_, err = p.expect(token.LEFT_BRACE, "Expected interface body")
	if err != nil {
		return nil, err
	}

	members := []ast.InterfaceMember{}

	for !p.eof() && p.next().Type != token.RIGHT_BRACE {
		memberName, err := p.expect(token.IDENTIFIER, "Expected closing brace or interface member")
		if err != nil {
			return nil, err
		}

		currentMember := ast.InterfaceMember{Name: memberName.Value}

		if p.next().Type == token.LEFT_PAREN {
			p.consume()
			currentMember.IsFunction = true
			currentMember.Parameters = []ast.TypeExpression{}

			for p.next().Type != token.RIGHT_PAREN {
				nextType, err := p.parseType()
				if err != nil {
					return nil, err
				}
				currentMember.Parameters = append(currentMember.Parameters, nextType)

				if p.next().Type != token.RIGHT_PAREN {
					return nil, p.error("Expected comma or end of parameter list", p.next())
				}
			}

			p.consume()
		}

		_, err = p.expect(token.COLON, "Expected type annotation")
		if err != nil {
			return nil, err
		}
		resultType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		currentMember.ResultType = resultType

		members = append(members, currentMember)

		if p.next().Type != token.RIGHT_BRACE {
			_, err = p.expect(token.COMMA, "Expected comma or end of interface body")
			if err != nil {
				return nil, err
			}
		}
	}

	_, err = p.expect(token.RIGHT_BRACE, "Unexpected EOF, expected '}'")
	if err != nil {
		return nil, err
	}

	return &ast.InterfaceDeclaration{
		BaseNode: ast.BaseNode{Token: tok},
		Name:     name.Value,
		Members:  members,
	}, nil
}

func (p *parser) parseTypeDeclaration() (ast.Statement, error) {
	tok := p.consume()
	name, err := p.expect(token.IDENTIFIER, "Expected type name, got %q")
	if err != nil {
		return nil, err
	}
	_, err = p.expect(token.EQUALS, "Expected initialiser for type declaration")
	if err != nil {
		return nil, err
	}

	dataType, err := p.parseType()
	if err != nil {
		return nil, err
	}

	return &ast.TypeDeclaration{
		BaseNode: ast.BaseNode{Token: tok},
		Name:     name.Value,
		DataType: dataType,
	}, nil
}

func (p *parser) parseImportStatement() (ast.Statement, error) {
	tok := p.consume()
	importAll := false

	if p.next().Type == token.STAR {
		importAll = true
		p.consume()
		_, err := p.expectKeyword("from", "Expected from keyword after `import *`")
		if err != nil {
			return nil, err
		}
	}

	var importedSymbols []string
	if p.next().Type == token.LEFT_BRACE {
		if importAll {
			return nil, p.error("Cannot list imported symbols and import all symbols", p.next())
		}
		p.consume()
		importedSymbols = []string{}
		for p.next().Type != token.RIGHT_BRACE {
			symbol, err := p.expect(token.IDENTIFIER, "Invalid imported symbol name %q")
			if err != nil {
				return nil, err
			}
			importedSymbols = append(importedSymbols, symbol.Value)
			if p.next().Type != token.RIGHT_BRACE {
				_, err := p.expect(token.COMMA, "Expected comma or end of imported symbol list")
				if err != nil {
					return nil, err
				}
			}
		}
		p.consume()
		_, err := p.expectKeyword("from", "Expected from keyword after listing imported symbols")
		if err != nil {
			return nil, err
		}
	}

	mod, err := p.expect(token.STRING, "Invalid import module %q")
	if err != nil {
		return nil, err
	}

	alias := ""
	if p.isKeyword("as") {
		if importAll {
			return nil, p.error("Cannot use alias import in conjunction with importing all symbols", p.next())
		}
		if importedSymbols != nil {
			return nil, p.error("Cannot use alias import in conjunction with listing imported symbols", p.next())
		}

		p.consume()
		name, err := p.expect(token.IDENTIFIER, "Expected import alias")
		if err != nil {
			return nil, err
		}
		alias = name.Value
	}

	return &ast.ImportStatement{
		BaseNode:        ast.BaseNode{Token: tok},
		Module:          mod.Value,
		ImportAll:       importAll,
		Alias:           alias,
		ImportedSymbols: importedSymbols,
	}, nil
}

func (p *parser) parseExportStatement() (ast.Statement, error) {
	p.consume()

	stmt, err := p.parseStatement()
	if err != nil {
		return nil, err
	}

	if stmt.IsExport() {
		return nil, p.error("Cannot double-export a statement", stmt.GetToken())
	}

	if _, isExportable := stmt.(ast.Exportable); !isExportable {
		return nil, p.error(fmt.Sprintf("Cannot export statement of type %q", stmt.Type()), stmt.GetToken())
	}

	stmt.MarkExport()
	return stmt, nil
}

func (p *parser) parseEnumDeclaration() (ast.Statement, error) {
	tok := p.consume()
	isUnion := tok.Value == "union"

	name, err := p.expect(token.IDENTIFIER, "Expected "+tok.Value+" name")
	if err != nil {
		return nil, err
	}

	_, err = p.expect(token.LEFT_BRACE, "Expected "+tok.Value+" body")
	if err != nil {
		return nil, err
	}

	members := map[string]ast.EnumMember{}

	for !p.eof() && p.next().Type != token.RIGHT_BRACE {
		member, err := p.parseEnumMember(tok.Value)
		if err != nil {
			return nil, err
		}
		members[member.Name] = member

		if p.next().Type != token.RIGHT_BRACE {
			_, err = p.expect(token.COMMA, "Expected comma or end of "+tok.Value+" body")
			if err != nil {
				return nil, err
			}
		}
	}

	_, err = p.expect(token.RIGHT_BRACE, "Expected end of "+tok.Value+" body, reached end of file")
	if err != nil {
		return nil, err
	}

	return &ast.EnumDeclaration{
		BaseNode: ast.BaseNode{Token: tok},
		IsUnion:  isUnion,
		Name:     name.Value,
		Members:  members,
	}, nil
}

func (p *parser) parseEnumMember(kind string) (ast.EnumMember, error) {
	exported := false
	if p.isKeyword("pub") {
		p.consume()
		exported = true
	}

	name, err := p.expect(token.IDENTIFIER, "Expected "+kind+" member")
	if err != nil {
		return ast.EnumMember{}, err
	}

	var types []ast.TypeExpression
	var structMembers map[string]ast.StructField
	if p.next().Type == token.LEFT_PAREN {
		p.consume()
		types = []ast.TypeExpression{}
		for !p.eof() && p.next().Type != token.RIGHT_PAREN {
			nextType, err := p.parseType()

			if err != nil {
				return ast.EnumMember{}, err
			}
			types = append(types, nextType)

			if p.next().Type != token.RIGHT_PAREN {
				_, err = p.expect(token.COMMA, "Expected comma or end of type list")
				if err != nil {
					return ast.EnumMember{}, err
				}
			}
		}
		_, err := p.expect(token.RIGHT_PAREN, "Unexpected eof")
		if err != nil {
			return ast.EnumMember{}, nil
		}
	} else if p.next().Type == token.LEFT_BRACE {
		p.consume()
		structMembers = map[string]ast.StructField{}

		for !p.eof() && p.next().Type != token.RIGHT_BRACE {
			name, member, err := p.parseStructField()
			if err != nil {
				return ast.EnumMember{}, err
			}
			structMembers[name] = member

			if p.next().Type != token.RIGHT_BRACE {
				_, err := p.expect(token.COMMA, "Expected comma or end of struct")
				if err != nil {
					return ast.EnumMember{}, err
				}
			}
		}

		_, err := p.expect(token.RIGHT_BRACE, "Unexpected eof")
		if err != nil {
			return ast.EnumMember{}, nil
		}
	}

	return ast.EnumMember{
		Name:          name.Value,
		Exported:      exported,
		Types:         types,
		StructMembers: structMembers,
	}, nil
}
