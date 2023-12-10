package typechecker

import (
	"fmt"
	"log"

	"github.com/gearsdatapacks/libra/errors"
	"github.com/gearsdatapacks/libra/modules"
	"github.com/gearsdatapacks/libra/parser/ast"
	"github.com/gearsdatapacks/libra/type_checker/registry"
	"github.com/gearsdatapacks/libra/type_checker/types"
)

func typeCheckExpression(expr ast.Expression, manager *modules.ModuleManager) types.ValidType {
	dataType := doTypeCheckExpression(expr, manager)
	if dataType.String() == "TypeError" {
		return dataType
	}

	if _, ok := dataType.(*types.Type); ok {
		return types.Error(fmt.Sprintf("Cannot use %q as a value, it is a type", expr.String()), expr)
	}

	return dataType
}

func doTypeCheckExpression(expr ast.Expression, manager *modules.ModuleManager) types.ValidType {
	switch expression := expr.(type) {
	case *ast.IntegerLiteral:
		return &types.IntLiteral{}
	case *ast.FloatLiteral:
		return &types.FloatLiteral{}
	case *ast.StringLiteral:
		return &types.StringLiteral{}
	case *ast.NullLiteral:
		return &types.NullLiteral{}
	case *ast.BooleanLiteral:
		return &types.BoolLiteral{}
	case *ast.VoidValue:
		return &types.Void{}

	case *ast.Identifier:
		dataType := manager.SymbolTable.GetSymbol(expression.Symbol)
		if err, isErr := dataType.(*types.TypeError); isErr {
			err.Line = expression.Token.Line
			err.Column = expression.Token.Column
		}
		return dataType

	case *ast.BinaryOperation:
		return typeCheckBinaryOperation(expression, manager)

	case *ast.UnaryOperation:
		return typeCheckUnaryOperation(expression, manager)

	case *ast.AssignmentExpression:
		return typeCheckAssignmentExpression(expression, manager)

	case *ast.FunctionCall:
		return typeCheckFunctionCall(expression, manager)

	case *ast.ListLiteral:
		return typeCheckList(expression, manager)

	case *ast.MapLiteral:
		return typeCheckMap(expression, manager)

	case *ast.IndexExpression:
		return typeCheckIndexExpression(expression, manager)

	case *ast.MemberExpression:
		return typeCheckMemberExpression(expression, manager)

	case *ast.StructExpression:
		return typeCheckStructExpression(expression, manager)

	case *ast.TupleExpression:
		return typeCheckTuple(expression, manager)

	case *ast.CastExpression:
		return typeCheckCastExpression(expression, manager)

	case *ast.TypeCheckExpression:
		return typeCheckTypeCheckExpression(expression, manager)

	default:
		log.Fatal(errors.DevError("(Type checker) Unexpected expression type: " + expr.String()))
		return nil
	}
}

func TypeCheckTypeExpression(expr ast.Expression, manager *modules.ModuleManager) types.ValidType {
	if name, ok := expr.(*ast.Identifier); ok {
		return manager.SymbolTable.GetType(name.Symbol)
	}

	dataType := doTypeCheckExpression(expr, manager)
	if dataType.String() == "TypeError" {
		return dataType
	}

	if ty, isType := dataType.(*types.Type); isType {
		return ty.DataType
	}
	return types.Error(fmt.Sprintf("Cannot use %q as type, it is a value", expr.String()), expr)
}

func typeCheckAssignmentExpression(assignment *ast.AssignmentExpression, manager *modules.ModuleManager) types.ValidType {
	var dataType types.ValidType
	if assignment.Assignee.Type() == "Identifier" {
		symbolName := assignment.Assignee.(*ast.Identifier).Symbol

		dataType = manager.SymbolTable.GetSymbol(symbolName)

	} else if assignment.Assignee.Type() == "IndexExpression" {
		index := assignment.Assignee.(*ast.IndexExpression)
		leftType := typeCheckExpression(index.Left, manager)
		if leftType.String() == "TypeError" {
			return leftType
		}
		indexType := typeCheckExpression(index.Index, manager)
		if indexType.String() == "TypeError" {
			return indexType
		}

		dataType = leftType.IndexBy(indexType)
	} else if assignment.Assignee.Type() == "MemberExpression" {
		member := assignment.Assignee.(*ast.MemberExpression)
		leftType := typeCheckExpression(member.Left, manager)
		if leftType.String() == "TypeError" {
			return leftType
		}

		dataType = types.Member(leftType, member.Member, member.IsNumberMember)
	} else {
		return types.Error("Can only assign values to variables", assignment)
	}

	if dataType.String() == "TypeError" {
		return dataType
	}

	if dataType.Constant() {
		return types.Error("Cannot assign data to constant value", assignment)
	}

	expressionType := typeCheckExpression(assignment.Value, manager)
	if expressionType.String() == "TypeError" {
		return expressionType
	}
	correctType := dataType.Valid(expressionType)

	if correctType {
		return dataType
	}

	return types.Error(fmt.Sprintf("Type %q is not assignable to type %q", expressionType, dataType), assignment)
}

func typeCheckFunctionCall(call *ast.FunctionCall, manager *modules.ModuleManager) types.ValidType {
	if ident, ok := call.Left.(*ast.Identifier); ok {
		name := ident.Symbol

		if structType, isStruct := manager.SymbolTable.GetType(name).(*types.TupleStruct); isStruct {
			return typeCheckTupleStructExpression(structType, call, manager)
		}

		if builtin, ok := registry.Builtins[name]; ok {
			if len(builtin.Parameters) != len(call.Args) {
				if len(call.Args) < len(builtin.Parameters) {

					return types.Error(fmt.Sprintf("Missing argument for function %q", name), call)
				}
				return types.Error(fmt.Sprintf("Extra argument passed to function %q", name), call)
			}

			for i, param := range builtin.Parameters {
				arg := typeCheckExpression(call.Args[i], manager)
				if arg.String() == "TypeError" {
					return arg
				}
				if !param.Valid(arg) {
					return types.Error(fmt.Sprintf("Invalid arguments passed to function %q: Type %q is not a valid argument for parameter of type %q", name, arg, param), call)
				}
			}

			return builtin.ReturnType
		}
	}

	callVar := typeCheckExpression(call.Left, manager)
	if callVar.String() == "TypeError" {
		return callVar
	}

	if structType, isStruct := callVar.(*types.TupleStruct); isStruct {
		return typeCheckTupleStructExpression(structType, call, manager)
	}

	function, ok := callVar.(*types.Function)

	if !ok {
		return types.Error(fmt.Sprintf("%q is not a function", call.Left.String()), call)
	}

	name := function.Name

	if len(function.Parameters) != len(call.Args) {
		if len(call.Args) < len(function.Parameters) {

			return types.Error(fmt.Sprintf("Missing argument for function %q", name), call)
		}
		return types.Error(fmt.Sprintf("Extra argument passed to function %q", name), call)
	}

	for i, param := range function.Parameters {
		arg := typeCheckExpression(call.Args[i], manager)
		if arg.String() == "TypeError" {
			return arg
		}
		if !param.Valid(arg) {
			return types.Error(fmt.Sprintf("Invalid arguments passed to function %q: Type %q is not a valid argument for parameter of type %q", name, arg, param), call)
		}
	}

	return function.ReturnType
}

func typeCheckTupleStructExpression(tuple *types.TupleStruct, instance *ast.FunctionCall, manager *modules.ModuleManager) types.ValidType {
	if len(tuple.Members) != len(instance.Args) {
		return types.Error("Tuple struct expression incompatible with type", instance)
	}
	for i, arg := range instance.Args {
		argType := typeCheckExpression(arg, manager)
		if argType.String() == "TypeError" {
			return argType
		}
		if !tuple.Members[i].Valid(argType) {
			return types.Error("Tuple struct expression incompatible with type", instance)
		}
	}

	return tuple
}

func typeCheckList(list *ast.ListLiteral, manager *modules.ModuleManager) types.ValidType {
	listTypes := []types.ValidType{}

	for _, elem := range list.Elements {
		elemType := typeCheckExpression(elem, manager)
		if elemType.String() == "TypeError" {
			return elemType
		}
		newType := true
		for _, listType := range listTypes {
			if listType.Valid(elemType) {
				newType = false
				break
			}
		}

		if newType {
			listTypes = append(listTypes, elemType)
		}
	}

	return &types.ArrayLiteral{
		ElemType: types.MakeUnion(listTypes...),
		Length:   len(list.Elements),
		CanInfer: true,
	}
}

func typeCheckMap(maplit *ast.MapLiteral, manager *modules.ModuleManager) types.ValidType {
	keyTypes := []types.ValidType{}
	valueTypes := []types.ValidType{}

	for key, value := range maplit.Elements {
		keyType := typeCheckExpression(key, manager)
		if keyType.String() == "TypeError" {
			return keyType
		}
		newType := true
		for _, dataType := range keyTypes {
			if dataType.Valid(keyType) {
				newType = false
				break
			}
		}

		if newType {
			keyTypes = append(keyTypes, keyType)
		}

		valueType := typeCheckExpression(value, manager)
		if valueType.String() == "TypeError" {
			return valueType
		}

		newType = true
		for _, dataType := range valueTypes {
			if dataType.Valid(valueType) {
				newType = false
				break
			}
		}

		if newType {
			valueTypes = append(valueTypes, valueType)
		}
	}

	return &types.MapLiteral{
		KeyType:   types.MakeUnion(keyTypes...),
		ValueType: types.MakeUnion(valueTypes...),
	}
}

func typeCheckIndexExpression(indexExpr *ast.IndexExpression, manager *modules.ModuleManager) types.ValidType {
	leftType := typeCheckExpression(indexExpr.Left, manager)
	if leftType.String() == "TypeError" {
		return leftType
	}

	indexType := typeCheckExpression(indexExpr.Index, manager)
	if indexType.String() == "TypeError" {
		return indexType
	}

	resultType := leftType.IndexBy(indexType)
	if resultType == nil {
		return types.Error(fmt.Sprintf("Type %q is not indexable with type %q", leftType.String(), indexType.String()), indexExpr)
	}

	return resultType
}

func typeCheckMemberExpression(memberExpr *ast.MemberExpression, manager *modules.ModuleManager) types.ValidType {
	leftType := typeCheckExpression(memberExpr.Left, manager)
	if leftType.String() == "TypeError" {
		return leftType
	}

	resultType := types.Member(leftType, memberExpr.Member, memberExpr.IsNumberMember)
	if resultType == nil {
		return types.Error(fmt.Sprintf("Type %q does not have member %q", leftType.String(), memberExpr.Member), memberExpr)
	}

	return resultType
}

func typeCheckStructExpression(structExpr *ast.StructExpression, manager *modules.ModuleManager) types.ValidType {
	definedType := TypeCheckTypeExpression(structExpr.InstanceOf, manager)
	if definedType.String() == "TypeError" {
		return types.Error(fmt.Sprintf("Struct %q is undefined", structExpr.InstanceOf), structExpr)
	}

	structType, isStruct := definedType.(*types.Struct)
	if !isStruct {
		return types.Error(fmt.Sprintf("Cannot instantiate %q, it is not a struct", definedType), structExpr)
	}

	members := map[string]types.ValidType{}

	for name, member := range structExpr.Members {
		dataType := typeCheckExpression(member, manager)
		if dataType.String() == "TypeError" {
			return dataType
		}

		members[name] = dataType
	}

	instanceType := &types.Struct{
		Name:    structType.Name,
		Members: members,
	}

	if !structType.Valid(instanceType) {
		return types.Error("Struct expression incompatiable with type", structExpr)
	}

	return instanceType
}

func typeCheckTuple(tuple *ast.TupleExpression, manager *modules.ModuleManager) types.ValidType {
	members := []types.ValidType{}
	for _, member := range tuple.Members {
		memberType := typeCheckExpression(member, manager)
		if memberType.String() == "TypeError" {
			return memberType
		}
		members = append(members, memberType)
	}

	return &types.Tuple{Members: members}
}

func typeCheckCastExpression(cast *ast.CastExpression, manager *modules.ModuleManager) types.ValidType {
	leftType := typeCheckExpression(cast.Left, manager)
	if leftType.String() == "TypeError" {
		return leftType
	}

	castTo := TypeCheckType(cast.DataType, manager)
	if castTo.String() == "TypeError" {
		return castTo
	}

	if !types.CanCast(leftType, castTo) {
		return types.Error(fmt.Sprintf("Cannot cast type %q to type %q", leftType, castTo), cast)
	}

	return castTo
}

func typeCheckTypeCheckExpression(expr *ast.TypeCheckExpression, manager *modules.ModuleManager) types.ValidType {
	leftType := typeCheckExpression(expr.Left, manager)
	if leftType.String() == "TypeError" {
		return leftType
	}

	compType := TypeCheckType(expr.DataType, manager)
	if compType.String() == "TypeError" {
		return compType
	}

	if !leftType.Valid(compType) {
		return types.Error(fmt.Sprintf("Type %q can never be type %q", leftType, compType), expr)
	}

	return &types.BoolLiteral{}
}
