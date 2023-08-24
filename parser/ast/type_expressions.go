package ast

import (
	"strings"

	"github.com/gearsdatapacks/libra/lexer/token"
)

type BaseType struct {}

func (bt *BaseType) typeNode() {}

type TypeName struct {
	*BaseNode
	*BaseType
	Name string
}

func (tn *TypeName) Type() NodeType {
	return "TypeName"
}

func (tn *TypeName) String() string {
	return tn.Name
}

type Union struct {
	*BaseNode
	*BaseType
	ValidTypes []TypeExpression
}

func (u *Union) Type() NodeType {
	return "Union"
}

func (u *Union) String() string {
	stringTypes := []string{}

	for _, typeExpr := range u.ValidTypes {
		stringTypes = append(stringTypes, typeExpr.String())
	}

	return strings.Join(stringTypes, " | ")
}

type InferType struct {
	*BaseNode
	*BaseType
}

func (i *InferType) Type() NodeType { return "Infer" }
func (i *InferType) String() string { return "" }

type VoidType struct {
	*BaseNode
	*BaseType
}

func (v *VoidType) Type() NodeType { return "Void" }
func (v *VoidType) String() string { return "void" }
func (v *VoidType) GetToken() token.Token { return token.Token{Value: "void"} }