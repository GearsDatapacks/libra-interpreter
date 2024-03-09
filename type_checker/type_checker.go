package typechecker

import (
	"github.com/gearsdatapacks/libra/diagnostics"
	"github.com/gearsdatapacks/libra/parser/ast"
	"github.com/gearsdatapacks/libra/type_checker/ir"
)

type typeChecker struct {
	Diagnostics diagnostics.Manager
}

func New(diagnostics diagnostics.Manager) *typeChecker {
	return &typeChecker{
		Diagnostics: diagnostics,
	}
}

func (t *typeChecker) TypeCheck(program *ast.Program) *ir.Program {
	stmts := []ir.Statement{}

	for _, stmt := range program.Statements {
		stmts = append(stmts, t.typeCheckStatement(stmt))
	}

	return &ir.Program{
		Statements: stmts,
	}
}
