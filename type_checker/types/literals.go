package types

import "fmt"

func isA[T ValidType](v ValidType) bool {
	_, ok := v.(T)
	return ok
}

type UntypedNumber struct {
	BaseType
	Default         ValidType
	IsIntAssignable bool
}

func (*UntypedNumber) Valid(t ValidType) bool {
	return isA[*UntypedNumber](t)
}

func (n *UntypedNumber) String() string {
	if n.IsIntAssignable {
		return "untyped int"
	}
	return "untyped float"
}

func (n *UntypedNumber) ToReal() ValidType {
	return n.Default
}

type IntLiteral struct{ BaseType }

func (i *IntLiteral) Valid(t ValidType) bool {
	if untyped, ok := t.(*UntypedNumber); ok {
		return untyped.IsIntAssignable
	}
	return isA[*IntLiteral](t)
}
func (i *IntLiteral) String() string         { return "int" }
func (i *IntLiteral) Infer(dataType ValidType) (ValidType, bool) {
	if n, ok := dataType.(*UntypedNumber); ok {
		return i, n.IsIntAssignable
	}

	return i, false
}
func (i *IntLiteral) CanCastTo(t ValidType) bool { return i.Valid(t) || (&FloatLiteral{}).Valid(t) }

type FloatLiteral struct{ BaseType }

func (f *FloatLiteral) String() string           { return "float" }
func (f *FloatLiteral) Valid(t ValidType) bool   { 
	if _, ok := t.(*UntypedNumber); ok {
		return true
	}
	return isA[*FloatLiteral](t)
}
func (f *FloatLiteral) Infer(dataType ValidType) (ValidType, bool) {
	if _, ok := dataType.(*UntypedNumber); ok {
		return f, true
	}

	return f, false
}
func (f *FloatLiteral) CanCastTo(t ValidType) bool { return f.Valid(t) || (&IntLiteral{}).Valid(t) }

type BoolLiteral struct{ BaseType }

func (b *BoolLiteral) String() string         { return "boolean" }
func (b *BoolLiteral) Valid(t ValidType) bool { return isA[*BoolLiteral](t) }

type NullLiteral struct{ BaseType }

func (n *NullLiteral) String() string         { return "null" }
func (n *NullLiteral) Valid(t ValidType) bool { return isA[*NullLiteral](t) }

type StringLiteral struct{ BaseType }

func (s *StringLiteral) String() string         { return "string" }
func (s *StringLiteral) Valid(t ValidType) bool { return isA[*StringLiteral](t) }

type ListLiteral struct {
	BaseType
	ElemType ValidType
}

func (list *ListLiteral) String() string {
	if isA[*Union](list.ElemType) {
		return fmt.Sprintf("(%s)[]", list.ElemType.String())
	}
	return list.ElemType.String() + "[]"
}
func (list *ListLiteral) Valid(t ValidType) bool {
	if l, isList := t.(*ListLiteral); isList {
		if isA[*Infer](l.ElemType) {
			return true
		}
		return list.ElemType.Valid(l.ElemType) && l.ElemType.Valid(list.ElemType)
	}
	if array, isArray := t.(*ArrayLiteral); isArray {
		if isA[*Infer](array.ElemType) {
			return array.CanInfer
		}
		return array.CanInfer && list.ElemType.Valid(array.ElemType)
	}
	return false
}

func (list *ListLiteral) Infer(dataType ValidType) (ValidType, bool) {
	if !list.Valid(dataType) {
		return list, false
	}

	if list.ElemType.String() != "Infer" {
		return list, true
	}

	if array, ok := dataType.(*ArrayLiteral); ok {
		if array.ElemType.String() == "Infer" {
			return &ListLiteral{
				ElemType: &Void{},
			}, true
		} 

		return &ListLiteral{
			ElemType: array.ElemType,
		}, true
	}

	return dataType, true
}

func (list *ListLiteral) IndexBy(dataType ValidType) ValidType {
	if !(&IntLiteral{}).Valid(dataType) {
		return nil
	}
	if list.constant {
		list.ElemType.MarkConstant()
	}
	return list.ElemType
}

type ArrayLiteral struct {
	BaseType
	ElemType ValidType
	Length   int
	CanInfer bool // For array literals to be type inferred
}

func (array *ArrayLiteral) String() string {
	length := "_"
	if array.Length != -1 {
		length = fmt.Sprint(array.Length)
	}
	if isA[*Union](array.ElemType) {
		return fmt.Sprintf("(%s)[%s]", array.ElemType.String(), length)
	}
	return fmt.Sprintf("%s[%s]", array.ElemType.String(), length)
}

func (array *ArrayLiteral) Valid(t ValidType) bool {
	if !isA[*ArrayLiteral](t) {
		return false
	}
	other := t.(*ArrayLiteral)
	lengthsMatch := array.Length == -1 || array.Length == other.Length
	return lengthsMatch && array.ElemType.Valid(other.ElemType) && other.ElemType.Valid(array.ElemType)
}

func (array *ArrayLiteral) Infer(dataType ValidType) (ValidType, bool) {
	if !array.Valid(dataType) {
		return array, false
	}

	if array.Length != -1 && array.ElemType.String() != "Infer" {
		return array, true
	}

	other := dataType.(*ArrayLiteral)
	var length int
	var elemType ValidType
	if array.Length != -1 {
		length = array.Length
	} else {
		length = other.Length
	}

	if array.ElemType.String() != "Infer" {
		elemType = array.ElemType
	} else {
		elemType = other.ElemType
	}

	return &ArrayLiteral{
		ElemType: elemType,
		Length:   length,
		CanInfer: false,
	}, true
}

func (array *ArrayLiteral) IndexBy(dataType ValidType) ValidType {
	if !(&IntLiteral{}).Valid(dataType) {
		return nil
	}
	if array.constant {
		array.ElemType.MarkConstant()
	}
	return array.ElemType
}

type MapLiteral struct {
	BaseType
	KeyType   ValidType
	ValueType ValidType
	Length    int
}

func (maplit *MapLiteral) String() string {
	return fmt.Sprintf("{%s: %s}", maplit.KeyType.String(), maplit.ValueType.String())
}

func (maplit *MapLiteral) Valid(t ValidType) bool {
	otherMap, isMap := t.(*MapLiteral)
	if !isMap {
		return false
	}
	return maplit.KeyType.Valid(otherMap.KeyType) && maplit.ValueType.Valid(otherMap.ValueType)
}

func (maplit *MapLiteral) IndexBy(dataType ValidType) ValidType {
	if !maplit.KeyType.Valid(dataType) {
		return nil
	}
	if maplit.constant {
		maplit.ValueType.MarkConstant()
	}
	return maplit.ValueType
}

type Pointer struct {
	BaseType
	DataType ValidType
}

func (p *Pointer) String() string {
	if isA[*Union](p.DataType) {
		return fmt.Sprintf("(%s)*", p.DataType.String())
	}
	return p.DataType.String() + "*"
}

func (p *Pointer) Valid(t ValidType) bool {
	ptr, ok := t.(*Pointer)
	if !ok {
		return false
	}
	return p.DataType.Valid(ptr.DataType)
}

type Void struct{ BaseType }

func (v *Void) String() string         { return "void" }
func (v *Void) Valid(t ValidType) bool { return isA[*Void](t) }

type Infer struct{ BaseType }

func (i *Infer) String() string         { return "Infer" }
func (i *Infer) Valid(t ValidType) bool { return true }
