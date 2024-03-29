package types

type ValidType interface {
	Valid(ValidType) bool
	String() string
	WasVariable() bool
	MarkVariable()
	Constant() bool
	MarkConstant()
	IndexBy(ValidType) ValidType
	IsForeign(int) bool
	SetModule(int)
}

type hasMembers interface {
	member(string, int) ValidType
}

type hasNumberMembers interface {
	numberMember(string) ValidType
}

type BaseType struct {
	wasVariable bool
	constant    bool
	module      int
}

func (b *BaseType) WasVariable() bool {
	return b.wasVariable
}

func (b *BaseType) MarkVariable() {
	b.wasVariable = true
}

func (b *BaseType) Constant() bool {
	return b.constant
}

func (b *BaseType) MarkConstant() {
	b.constant = true
}

func (*BaseType) IndexBy(ValidType) ValidType {
	return nil
}

func (b *BaseType) IsForeign(moduleId int) bool {
	// Allow bypassing of visibility system
	if b.module == 0 || moduleId == 0 {
		return false
	}

	return b.module != moduleId
}

func (b *BaseType) SetModule(moduleId int) {
	b.module = moduleId
}

type CastableTo interface {
	CanCastTo(ValidType) bool
}

type CastableFrom interface {
	CanCastFrom(ValidType) bool
}

func CanCast(from, to ValidType) bool {
	if castable, ok := from.(CastableTo); ok {
		if castable.CanCastTo(to) {
			return true
		}
	}

	if castable, ok := to.(CastableFrom); ok {
		if castable.CanCastFrom(from) {
			return true
		}
	}

	return from.Valid(to) || to.Valid(from)
}

type PartialType interface {
	ValidType
	Infer(ValidType) (ValidType, bool)
}


var typeTable = map[string]ValidType{
	"int":      &IntLiteral{},
	"float":    &FloatLiteral{},
	"boolean":  &BoolLiteral{},
	"null":     &NullLiteral{},
	"function": &Function{},
	"string":   &StringLiteral{},
}

type TypeTable interface {
	GetType(string) ValidType
}

func FromString(typeString string, table TypeTable) ValidType {
	dataType, ok := typeTable[typeString]
	if !ok {
		return table.GetType(typeString)
	}

	return dataType
}

func Member(memberOf ValidType, name string, isNumberMember bool, moduleId int) ValidType {
	method := getMethod(memberOf, name, moduleId)
	if method != nil {
		return method
	}

	if !isNumberMember {
		hasMembers, ok := memberOf.(hasMembers)
		if ok {
			return hasMembers.member(name, moduleId)
		}
	} else {
		hasNumberMembers, ok := memberOf.(hasNumberMembers)
		if ok {
			return hasNumberMembers.numberMember(name)
		}
	}

	return nil
}

var methods = map[string][]*Function{}

func AddMethod(name string, method *Function) {
	overloads, ok := methods[name]
	if !ok {
		methods[name] = []*Function{method}
	}
	overloads = append(overloads, method)
	methods[name] = overloads
}

func getMethod(methodOf ValidType, name string, moduleId int) *Function {
	overloads, ok := methods[name]
	if !ok {
		return nil
	}

	for _, overload := range overloads {
		if overload.MethodOf.Valid(methodOf) {
			if methodOf.IsForeign(moduleId) && !overload.Exported {
				return nil
			}
			return overload
		}
	}

	return nil
}

type PseudoType interface {
	ToReal() ValidType
}
