package gacklib

import (
	"errors"
	"fmt"

	"go/ast"

	"reflect"
	"strconv"
	"strings"

	"github.com/0xfaded/eval"
	"github.com/gobs/readline"
)

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
	deleteMe := true
	if history == nil {
		introText()
		deleteMe = false
	} else {
		for _, h := range history {
			readline.AddHistory(h)
		}
	}

	// As a party piece. add the package contents as a map to the env. We can get away with this
	// because eval checks for packages before vars for EACH scope. Therefore, if a local variable
	// masks a pkg, it will still be found first. In other words, I'm cheating here, it's a hack.
	for name, pkg := range env.Pkgs {
		e := pkg.(*eval.SimpleEnv)
		// Don't overwrite existing vars
		if _, ok := e.Vars[name]; ok {
			continue
		}
		m := map[string]reflect.Type{}
		for n, v := range e.Vars {
			m[n] = v.Type()
		}
		for n, c := range e.Consts {
			m[n] = c.Type()
		}
		for n, f := range e.Funcs {
			m[n] = f.Type()
		}
		for n, t := range e.Types {
			m[n] = t
		}
		env.Vars[name] = reflect.ValueOf(&m)
	}

	complete := func(test string, start, end int) []string {
		return []string{""}
	}
	readline.SetAttemptedCompletionFunction(complete)

	prompt := "go> "
	line := ""
	for line != "quit" {
		result := readline.ReadLine(&prompt)
		if result == nil {
			fmt.Printf("\n")
			break
		}
		line := *result
		history = append(history, line)
		readline.AddHistory(line)
		if err := handleImport(env, line, history, deleteMe); err != nil {
			fmt.Println(err)
		// TODO[crc] move into generalised error position formatting code when written
		} else if stmt, err := eval.ParseStmt(line); err != nil {
			if pair := eval.FormatErrorPos(line, err.Error()); len(pair) == 2 {
				fmt.Println(pair[0])
				fmt.Println(pair[1])
			}
			fmt.Printf("%s\n", err)
		} else if expr, ok := stmt.(*ast.ExprStmt); ok {
			if cexpr, errs := eval.CheckExpr(expr.X, env); errs != nil {
				for _, cerr := range errs {
					fmt.Printf("%v\n", cerr)
				}
			} else if vals, err := eval.EvalExpr(cexpr, env); err != nil {
				fmt.Printf("panic: %s\n", err)
			} else if len(vals) == 0 {
				fmt.Printf("Kind=Slice\nvoid\n")
			} else {
				// Success
				if len(vals) == 1 {
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
						fmt.Printf("it = %s\n", eval.Inspect(value))
						it := reflect.New(vals[0].Type())
						it.Elem().Set(vals[0])
						env.Vars["it"] = it
					} else {
						fmt.Printf("%s\n", value)
					}
				} else if len(vals) > 2 {
					fmt.Printf("Kind = Multi-Value\n")
					size := len(vals)
					it := make([]interface{}, len(vals))
					for i, v := range vals {
						fmt.Printf("%s", eval.Inspect(v))
						if i < size-1 { fmt.Printf(", ") }
						it[i] = vals[i].Interface()
					}
					fmt.Printf("\n")
					env.Vars["it"] = reflect.ValueOf(&it)
				}
			}
		} else {
			if cstmt, errs := eval.CheckStmt(stmt, env); len(errs) != 0 {
				for _, cerr := range errs {
					fmt.Printf("%v\n", cerr)
				}
			} else if err = eval.InterpStmt(cstmt, env); err != nil {
				fmt.Printf("panic: %s\n", err)
			}
		}
	}
	if history != nil {
		deleteSelf()
	}
}

// This will only return a nil error if the line isn't an import statement.
// If successful, a the process will exec a new interpreter with the imports
// in scope.
func handleImport(env *eval.SimpleEnv, line string, history []string, deleteMe bool) error {
	line = strings.Trim(line, " \n\t")
	parts := strings.Split(line, " ")
	if len(parts) == 0 || parts[0] != "import" {
		return nil
	}
	inScope := map[string]bool{}
	for _, p := range env.Pkgs {
		inScope[p.(*eval.SimpleEnv).Path] = true
	}
	imports := []string{}
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		if i, err := strconv.Unquote(part); err != nil {
			return errors.New("invalid import `"+part+"'")
		} else if inScope[i] {
			return errors.New("import `"+part+"' already in scope")
		} else {
			imports = append(imports, i)
		}
	}
	return Quine(env, imports, history, deleteMe)
}

