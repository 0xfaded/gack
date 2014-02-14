package gack

import (
	"bufio"
	"errors"
	"fmt"
	"go/parser"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/0xfaded/eval"
)

// Simple replacement for GNU readline
func readline(prompt string, in *bufio.Reader) (string, error) {
	fmt.Printf(prompt)
	line, err := in.ReadString('\n')
	if err == nil {
		line = strings.TrimRight(line, "\r\n")
	}
	return line, err
}

func introText() {
	fmt.Printf(`=== A hacky Go eval REPL ===

Results of expression are stored in variable "it".
Full and short variable declarations and assignments are supported.
Import is also supported with limited support and a non-standard syntax.
Multiple packages may be specified, but they can not be qualified. E.g.

        import "fmt"
        import "pkg/a" "pkg/b" "pkg/c"

Enter expressions to be evaluated at the "go>" prompt.

To quit, enter: "quit" or Ctrl-D (EOF).
`)

}

func Repl(env *eval.SimpleEnv, history []string) {
	if history != nil {
		introText()
	}

	var err error

	// A place to store result values of expressions entered
	// interactively
	results := make([]interface{}, 0, 10)
	env.Vars["results"] = reflect.ValueOf(&results)

	exprs := 0
	in := bufio.NewReader(os.Stdin)
	line, err := readline("go> ", in)
	for line != "quit" {
		if err != nil {
			if err != io.EOF {
				fmt.Printf("gack error: %v", err)
			}
			break
		}
		if err := handleImport(env, line, history); err != nil {
			fmt.Println(err)
		} else if expr, err := parser.ParseExpr(line); err != nil {
			if pair := eval.FormatErrorPos(line, err.Error()); len(pair) == 2 {
				fmt.Println(pair[0])
				fmt.Println(pair[1])
			}
			fmt.Printf("parse error: %s\n", err)
		} else if cexpr, errs := eval.CheckExpr(expr, env); len(errs) != 0 {
			for _, cerr := range errs {
				fmt.Printf("check error: %v\n", cerr)
			}
		} else if vals, err := eval.EvalExpr(cexpr, env); err != nil {
			fmt.Printf("panic: %s\n", err)
		} else if len(vals) == 0 {
			fmt.Printf("Kind=Slice\nvoid\n")
		} else if len(vals) == 1 {
			value := (vals)[0]
			if value.IsValid() {
				kind := value.Kind().String()
				typ  := value.Type().String()
				if typ != kind {
					fmt.Printf("Kind = %v\n", kind)
					fmt.Printf("Type = %v\n", typ)
				} else {
					fmt.Printf("Kind = Type = %v\n", kind)
				}
				fmt.Printf("results[%d] = %s\n", exprs, eval.Inspect(value))
				exprs += 1
				results = append(results, (vals)[0].Interface())
			} else {
				fmt.Printf("%s\n", value)
			}
		} else {
			fmt.Printf("Kind = Multi-Value\n")
			size := len(vals)
			for i, v := range vals {
				fmt.Printf("%s", eval.Inspect(v))
				if i < size-1 { fmt.Printf(", ") }
			}
			fmt.Printf("\n")
			exprs += 1
			results = append(results, vals)
		}

		line, err = readline("go> ", in)
	}
}

// This will only return a nil error if the line isn't an import statement.
// If successful, a the process will exec a new interpreter with the imports
// in scope.
func handleImport(env *eval.SimpleEnv, line string, history []string) error {
	line = strings.Trim(line, " \n\t")
	parts := strings.Split(line, " ")
	if len(parts) == 0 || parts[0] != "import" {
		return nil
	}
	imports := []string{}
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		if i, err := strconv.Unquote(part); err != nil {
			return errors.New("Invalid import `"+part+"'")
		} else {
			imports = append(imports, i)
		}
	}
	return Quine(env, imports, history)
}

