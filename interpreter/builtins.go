package interpreter

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/gearsdatapacks/libra/errors"
	"github.com/gearsdatapacks/libra/interpreter/environment"
	"github.com/gearsdatapacks/libra/interpreter/values"
)

func toPrintString(value values.RuntimeValue) string {
	printStr := value.ToString()
	
	if _, ok := value.(*values.StringLiteral); ok {
		printStr = printStr[1:len(printStr)-1]
	}

	return printStr
}

func print(args []values.RuntimeValue, env *environment.Environment) values.RuntimeValue {
	fmt.Println(toPrintString(args[0]))

	return values.MakeNull()
}

func printil(args []values.RuntimeValue, env *environment.Environment) values.RuntimeValue {
	fmt.Print(toPrintString(args[0]))

	return values.MakeNull()
}

var reader = bufio.NewReader(os.Stdin)
func prompt(args []values.RuntimeValue, env *environment.Environment) values.RuntimeValue {
	fmt.Print(toPrintString(args[0]))

	result, _, _ := reader.ReadLine()

	return values.MakeString(string(result))
}

func toString(args []values.RuntimeValue, env *environment.Environment) values.RuntimeValue {
	return values.MakeString(args[0].ToString())
}

func parseInt(args []values.RuntimeValue, env *environment.Environment) values.RuntimeValue {
	stringValue := args[0].(*values.StringLiteral).Value
	intValue, err := strconv.ParseInt(stringValue, 10, 32)

	if err != nil {
		errors.LogError(fmt.Sprintf("parseInt: Invalid integer syntax: %q", stringValue))
	}

	return values.MakeInteger(int(intValue))
}

func parseFloat(args []values.RuntimeValue, env *environment.Environment) values.RuntimeValue {
	stringValue := args[0].(*values.StringLiteral).Value
	floatValue, err := strconv.ParseFloat(stringValue, 32)

	if err != nil {
		errors.LogError(fmt.Sprintf("parseFloat: Invalid float syntax: %q", stringValue))
	}

	return values.MakeFloat(floatValue)
}
