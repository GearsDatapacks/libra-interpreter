package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gearsdatapacks/libra/lexer/token"
)

type Union struct {
	BaseType
	Types []ValidType
}

func MakeUnion(types ...ValidType) ValidType {
	if len(types) == 0 {
		return &Void{}
	}
	if len(types) == 1 {
		return types[0]
	}

	return &Union{Types: types}
}

func (u *Union) Valid(dataType ValidType) bool {
	union, isUnion := dataType.(*Union)

	// If it's a union, we want to make sure all possible values it could be are contained within this one
	if isUnion {
		for _, unionType := range union.Types {
			if !u.Valid(unionType) {
				return false
			}
		}

		return true
	}

	// Otherwise, we just make sure the value is contained within this one
	for _, unionType := range u.Types {
		if unionType.Valid(dataType) {
			return true
		}
	}

	return false
}

func (u *Union) String() string {
	typeStrings := []string{}

	for _, dataType := range u.Types {
		typeStrings = append(typeStrings, dataType.String())
	}

	return strings.Join(typeStrings, " | ")
}

type EnumMember struct {
	DataType ValidType
	Exported bool
}

type Enum struct {
	BaseType
	Name  string
	Types map[string]*EnumMember
}

func (e *Enum) Valid(dataType ValidType) bool {
	enum, isEnum := dataType.(*Enum)

	if isEnum {
		return e.Name == enum.Name
	}

	for _, member := range e.Types {
		if member.DataType.Valid(dataType) {
			return true
		}
	}

	return false
}

func (e *Enum) String() string {
	return e.Name
}

func (e *Enum) member(name string, moduleId int) ValidType {
	member, ok := e.Types[name]
	if !ok {
		return nil
	}
	if moduleId != e.module && !member.Exported {
		fmt.Println(member)
		return nil
	}
	return member.DataType
}

type Function struct {
	BaseType
	Name       string
	Parameters []ValidType
	ReturnType ValidType
	MethodOf   ValidType
	Exported   bool
}

func (fn *Function) Valid(dataType ValidType) bool {
	otherFn, isFn := dataType.(*Function)
	if !isFn {
		return false
	}

	if otherFn.Name != fn.Name {
		return false
	}

	if !fn.ReturnType.Valid(otherFn.ReturnType) {
		return false
	}

	for i, param := range fn.Parameters {
		if !param.Valid(otherFn.Parameters[i]) {
			return false
		}
	}

	return true
}

func (fn *Function) String() string {
	return "function"
}

type Any struct{ BaseType }

func (a *Any) Valid(dataType ValidType) bool {
	_, isVoid := dataType.(*Void)
	return !isVoid
}

func (a *Any) String() string {
	return "any"
}

type StructField struct {
	Type     ValidType
	Exported bool
}

type Struct struct {
	BaseType
	Name    string
	Members map[string]StructField
}

func (s *Struct) Valid(dataType ValidType) bool {
	struc, isStruct := dataType.(*Struct)
	if !isStruct || struc.Name != s.Name {
		return false
	}

	for name, field := range struc.Members {
		member, hasMember := s.Members[name]
		if !hasMember || !member.Type.Valid(field.Type) {
			return false
		}
	}

	return true
}

func (s *Struct) String() string {
	return s.Name
}

func (s *Struct) member(member string, moduleId int) ValidType {
	memberType, ok := s.Members[member]
	if !ok {
		return nil
	}

	if s.constant {
		memberType.Type.MarkConstant()
	}

	if s.IsForeign(moduleId) && !memberType.Exported {
		return nil
	}

	return memberType.Type
}

type UnitStruct struct {
	BaseType
	Name string
	Id   int
}

var unitStructId = 0

func MakeUnitStruct(name string) *UnitStruct {
	unitStructId++
	return &UnitStruct{
		Name: name,
		Id:   unitStructId,
	}
}

func (s *UnitStruct) Valid(dataType ValidType) bool {
	struc, isStruct := dataType.(*UnitStruct)
	if !isStruct || struc.Id != s.Id {
		return false
	}

	return true
}

func (s *UnitStruct) String() string {
	return s.Name
}

type Interface struct {
	BaseType
	Name    string
	Members map[string]ValidType
}

func (i *Interface) Valid(dataType ValidType) bool {
	for name, member := range i.Members {

		memberType := Member(dataType, name, false, 0)
		if memberType == nil {
			return false
		}

		if !member.Valid(memberType) {
			return false
		}
	}

	return true
}

func (i *Interface) String() string {
	return i.Name
}

func (i *Interface) member(member string, moduleId int) ValidType {
	memberType := i.Members[member]
	if i.constant && memberType != nil {
		memberType.MarkConstant()
	}
	return memberType
}

var ErrorInterface = &Interface{
	Name: "error",
	Members: map[string]ValidType{
		"error": &Function{
			Name:       "error",
			Parameters: []ValidType{},
			ReturnType: &StringLiteral{},
		},
	},
}

type ErrorType struct {
	BaseType
	ResultType ValidType
}

func (e *ErrorType) Valid(dataType ValidType) bool {
	if err, ok := dataType.(*ErrorType); ok {
		return e.ResultType.Valid(err.ResultType)
	}

	return e.ResultType.Valid(dataType) || ErrorInterface.Valid(dataType)
}

func (e *ErrorType) String() string {
	return e.ResultType.String() + "!"
}

type TypeError struct {
	BaseType
	Message string
	Line    int
	Column  int
}

func (err TypeError) Error() string {
	if err.Line == -1 || err.Column == -1 {
		return "TypeError: " + err.Message
	}
	return fmt.Sprintf("TypeError at line %d, column %d: %s", err.Line, err.Column, err.Message)
}

func (*TypeError) Valid(ValidType) bool {
	return false
}

func (*TypeError) String() string {
	return "TypeError"
}

type node interface {
    GetToken() token.Token
}

func Error(message string, errorNodes ...node) *TypeError {
	if len(errorNodes) == 0 {
		return &TypeError{
			Line:    -1,
			Column:  -1,
			Message: message,
		}
	}

	errorNode := errorNodes[0]
	return &TypeError{
		Line:    errorNode.GetToken().Line,
		Column:  errorNode.GetToken().Column,
		Message: message,
	}
}

type Tuple struct {
	BaseType
	Members []ValidType
}

func (t *Tuple) Valid(dataType ValidType) bool {
	tuple, isTuple := dataType.(*Tuple)
	if !isTuple {
		return false
	}

	if len(tuple.Members) != len(t.Members) {
		return false
	}

	for i, member := range tuple.Members {
		memberType := t.Members[i]
		if !memberType.Valid(member) {
			return false
		}
	}

	return true
}

func (tuple *Tuple) String() string {
	result := "("

	for i, member := range tuple.Members {
		if i != 0 {
			result += ", "
		}
		result += member.String()
	}

	result += ")"

	return result
}

func (tuple *Tuple) numberMember(member string) ValidType {
	number, _ := strconv.ParseInt(member, 10, 32)
	if int(number) < len(tuple.Members) {
		return tuple.Members[number]
	}
	return nil
}

type TupleStruct struct {
	BaseType
	Name    string
	Members []ValidType
}

func (t *TupleStruct) Valid(dataType ValidType) bool {
	tuple, isTuple := dataType.(*TupleStruct)
	if !isTuple {
		return false
	}

	if tuple.Name != t.Name {
		return false
	}

	if len(tuple.Members) != len(t.Members) {
		return false
	}

	for i, member := range tuple.Members {
		memberType := t.Members[i]
		if !memberType.Valid(member) {
			return false
		}
	}

	return true
}

func (tuple *TupleStruct) String() string {
	return tuple.Name
}

func (tuple *TupleStruct) numberMember(member string) ValidType {
	number, _ := strconv.ParseInt(member, 10, 32)
	if int(number) < len(tuple.Members) {
		return tuple.Members[number]
	}
	return nil
}

type Module struct {
	BaseType
	Name    string
	Exports map[string]ValidType
}

func (m *Module) Valid(dataType ValidType) bool {
	return false
}

func (m *Module) String() string {
	return m.Name
}

func (s *Module) member(member string, moduleId int) ValidType {
	memberType := s.Exports[member]
	if s.constant && memberType != nil {
		memberType.MarkConstant()
	}
	return memberType
}

// Marks a type rather than a value of type x
type Type struct {
	BaseType
	DataType ValidType
}

func (t *Type) Valid(dataType ValidType) bool {
	return t.DataType.Valid(dataType)
}

func (t *Type) String() string {
	return t.DataType.String()
}

func (t *Type) SetModule(moduleId int) {
	t.module = moduleId
	t.DataType.SetModule(moduleId)
}

func (t *Type) member(name string, moduleId int) ValidType {
	return Member(t.DataType, name, false, moduleId)
}

type ExplicitType struct {
	BaseType
	Name     string
	Id       int
	DataType ValidType
}

var explicitTypeId = 0

func MakeExplicitType(name string, dataType ValidType) *ExplicitType {
	explicitTypeId++
	return &ExplicitType{
		Name:     name,
		Id:       explicitTypeId,
		DataType: dataType,
	}
}

func (t *ExplicitType) Valid(dataType ValidType) bool {
	explicit, ok := dataType.(*ExplicitType)
	if !ok {
		return false
	}
	return t.Id == explicit.Id
}

func (t *ExplicitType) String() string {
	return t.Name
}

func (t *ExplicitType) CanCastFrom(dataType ValidType) bool {
	return CanCast(dataType, t.DataType)
}

