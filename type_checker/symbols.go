package typechecker

import (
	"fmt"

	"github.com/gearsdatapacks/libra/errors"
	"github.com/gearsdatapacks/libra/type_checker/types"
	"github.com/gearsdatapacks/libra/utils"
)

type SymbolTable struct {
	parent    *SymbolTable
	symbols   map[string]types.ValidType
	constants []string
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		parent:  nil,
		symbols: map[string]types.ValidType{},
	}
}

func NewChildSymbolTable(parent *SymbolTable) *SymbolTable {
	return &SymbolTable{
		parent:  parent,
		symbols: map[string]types.ValidType{},
	}
}

func (st *SymbolTable) RegisterSymbol(name string, dataType types.ValidType, constant bool) {
	if _, ok := st.symbols[name]; ok {
		errors.TypeError(fmt.Sprintf("Cannot redeclare variable %q, it is already defined", name))
	}

	if constant {
		st.constants = append(st.constants, name)
	}

	st.symbols[name] = dataType
}

func (st *SymbolTable) GetSymbol(name string) types.ValidType {
	table, err := st.resolve(name)

	if err != "" {
		errors.TypeError(err)
	}

	return table.symbols[name]
}

func (st *SymbolTable) IsConstant(name string) bool {
	return utils.Contains(st.constants, name)
}

func (st *SymbolTable) Exists(name string) bool {
	_, err := st.resolve(name)

	return err == ""
}

func (st *SymbolTable) resolve(varName string) (table *SymbolTable, err string) {
	if _, ok := st.symbols[varName]; ok {
		return st, ""
	}

	if st.parent == nil {
		return nil, fmt.Sprintf("Variable %q is undefined", varName)
	}

	return st.parent.resolve(varName)
}
