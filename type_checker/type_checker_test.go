package typechecker_test

import (
	"fmt"
	"testing"

	"github.com/gearsdatapacks/libra/lexer"
	"github.com/gearsdatapacks/libra/parser"
	utils "github.com/gearsdatapacks/libra/test_utils"
	typechecker "github.com/gearsdatapacks/libra/type_checker"
	"github.com/gearsdatapacks/libra/type_checker/ir"
	"github.com/gearsdatapacks/libra/text"
)

func TestIntegerLiteral(t *testing.T) {
	input := "1_23_456"
	val := 123456

	program := getProgram(t, input)

	integer := getExpr[*ir.IntegerLiteral](t, program)

	utils.AssertEq(t, integer.Value, int64(val))
}

func getProgram(t *testing.T, input string) *ir.Program {
	t.Helper()

	l := lexer.New(text.NewFile("test.lb", input))
	tokens := l.Tokenise()

	p := parser.New(tokens, l.Diagnostics)
	program := p.Parse()
	tc := typechecker.New(p.Diagnostics)
	irProgram := tc.TypeCheck(program)
	utils.AssertEq(t, len(tc.Diagnostics), 0,
		fmt.Sprintf("Expected no diagnostics (got %d)", len(tc.Diagnostics)))

	return irProgram
}

func getExpr[T ir.Expression](t *testing.T, program *ir.Program) T {
	t.Helper()

	stmt := utils.AssertSingle(t, program.Statements)
	exprStmt, ok := stmt.(*ir.ExpressionStatement)
	utils.Assert(t, ok, fmt.Sprintf(
		"Statement is not an expression statement (is %T)", stmt))

	expr, ok := exprStmt.Expression.(T)
	utils.Assert(t, ok, fmt.Sprintf("Expression is not %T (is %T)",
		struct{ t T }{}.t, exprStmt.Expression))
	return expr
}