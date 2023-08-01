package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gearsdatapacks/libra/interpreter"
	"github.com/gearsdatapacks/libra/interpreter/environment"
	"github.com/gearsdatapacks/libra/interpreter/values"
	"github.com/gearsdatapacks/libra/lexer"
	"github.com/gearsdatapacks/libra/parser"
)

func repl() {
	fmt.Println("Libra repl v0.1.0")
	nextLine := ""
	reader := bufio.NewReader(os.Stdin)

	for strings.ToLower(strings.TrimSpace(nextLine)) != "exit" {
		fmt.Print("> ")

		input, err := reader.ReadBytes('\n')
		nextLine = string(input)

		if err != nil {
			log.Fatal(err)
		}

		lexer := lexer.New(input)
		parser := parser.New()
		env := environment.New()

		env.DeclareVariable("x", values.MakeValue(100))

		tokens := lexer.Tokenise()
		ast := parser.Parse(tokens)

		result := interpreter.Evaluate(ast, env)
		fmt.Println(result.ToString())
	}
}

func run(file string) {
	code, err := os.ReadFile(file)

	if err != nil {
		log.Fatal(err)
	}

	lexer := lexer.New(code)
	tokens := lexer.Tokenise()
	fmt.Println(tokens)
}

func main() {
	if len(os.Args) == 1 {
		repl()
	} else {
		run(os.Args[1])
	}
}
