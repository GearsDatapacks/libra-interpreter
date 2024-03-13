package types

import (
	"fmt"

	"github.com/gearsdatapacks/libra/parser/ast"
)

type Type interface {
	String() string
	valid(Type) bool
}

func Assignable(to, from Type) bool {
	if to == Invalid || from == Invalid {
		return true
	}

	return to.valid(from)
}

type PrimaryType int

const (
	Invalid PrimaryType = iota
	Int
	Float
	Bool
	String
)

var typeNames = map[PrimaryType]string{
	Invalid: "<?>",
	Int:     "i32",
	Float:   "f32",
	Bool:    "bool",
	String:  "string",
}

func (pt PrimaryType) String() string {
	return typeNames[pt]
}

func (pt PrimaryType) valid(other Type) bool {
	primary, isPrimary := other.(PrimaryType)
	return isPrimary && primary == pt
}

func FromAst(node ast.TypeExpression) Type {
	switch ty := node.(type) {
	case *ast.TypeName:
		return lookupType(ty.Name.Value)
	default:
		panic(fmt.Sprintf("TODO: Types from %T", ty))
	}
}

func lookupType(name string) Type {
	switch name {
	case "i32":
		return Int
	case "f32":
		return Float
	case "bool":
		return Bool
	case "string":
		return String
	default:
		return nil
	}
}
