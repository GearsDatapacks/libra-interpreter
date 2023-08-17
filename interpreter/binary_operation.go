package interpreter

import (
	"fmt"

	"github.com/gearsdatapacks/libra/errors"
	"github.com/gearsdatapacks/libra/interpreter/environment"
	"github.com/gearsdatapacks/libra/interpreter/values"
	"github.com/gearsdatapacks/libra/parser/ast"
)

var operators = map[[3]string]opFn{}

type opFn func(values.RuntimeValue, values.RuntimeValue) values.RuntimeValue

func RegisterOperator(op string, left string, right string, operation opFn) {
	operators[[3]string{op, left, right}] = operation
}

func evaluateBinaryOperation(binOp ast.BinaryOperation, env *environment.Environment) values.RuntimeValue {
	left := evaluateExpression(binOp.Left, env)
	right := evaluateExpression(binOp.Right, env)

	operation, ok := operators[[3]string{binOp.Operator, string(left.Type()), string(right.Type())}]

	if !ok {
		errors.RuntimeError(fmt.Sprintf("Operator %q does not exist or does not support operands of type %q and %q", binOp.Operator, left.Type(), right.Type()), &binOp)
	}

	return operation(left, right)
}

func evaluateAssignmentExpression(assignment ast.AssignmentExpression, env *environment.Environment) values.RuntimeValue {
	if assignment.Assignee.Type() != "Identifier" {
		errors.RuntimeError(fmt.Sprintf("Cannot assign value to type %q", assignment.Assignee.Type()), &assignment)
	}

	varName := assignment.Assignee.(*ast.Identifier).Symbol
	value := evaluateExpression(assignment.Value, env)

	return env.AssignVariable(varName, value)
}