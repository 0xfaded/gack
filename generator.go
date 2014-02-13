package gack

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"unicode"

	"go/ast"
	"go/token"

	"github.com/0xfaded/eval"
)

var Foobar int
const Foobaz = "abc"
var (
	A int
	B int
	E, F int
)
const (
	C = iota
	D
)

type X int

type (
	Y int
	Z int
)

// For tracking memory layout
type Mem struct {
	ptrs map[uintptr]string
	vars map[string]string
}

func WriteEnv(w io.Writer, env *eval.SimpleEnv, imports []string) error {
	// All pointers that point to the same place in the previous env must point to the
	// same place in the new env. We can't simply make a new value for each pointer as
	// we can for other types. Even worse, the pointer may point somewhere inside a compiled
	// package. Every pointer in the environment must be tracked and duplicates detected.
	mem := &Mem{make(map[uintptr]string), make(map[string]string)}

	_, err := fmt.Fprint(w,
	`root := &Eval.SimpleEnv{
		Var: map[string]reflect.Value{},
		Consts: map[string]reflect.Value{},
		Funcs: map[string]reflect.Value{},
		Types: map[string]reflect.Type{},
		Pkgs: map[string]eval.Env{
`);
	if err != nil {
		return err
	}
	for _, i := range imports {
		if pkg, err := Import(i); err != nil {
			return err
		} else if err := writeImport(w, pkg); err != nil {
			return err
		} else {
			delete(env.Pkgs, i)
		}
	}
	for name, pkg := range env.Pkgs {
		if err := writePkg(w, pkg.(*eval.SimpleEnv), name, mem); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\t\t},\n\t}\n"); err != nil {
		return err
	}

	// Convert the dynamic environment, as best we can, into variable declarations. Unfortunately,
	// some types can't be easily represented as literals, such as Chans and Func literals.
	var i int
	vars := make(map[string]string, len(env.Vars))
	for k, v := range env.Vars {
		e := v.Elem()
		if lit, err := sprintLiteral(e, mem); err != nil {
			fmt.Fprintf(os.Stderr, "warning! %v\n", err)
		} else {
			vars[k] = lit
			i += 1
		}
	}
	for k, v := range mem.vars {
		if _, err := fmt.Fprintf(w, "\t%s := %s\n", k, v); err != nil {
			return err
		}
	}
	for k, v := range vars {
		if _, err := fmt.Fprintf(w, "\troot.Vars[\"%s\"] = reflect.ValueOf(&%s)\n", k, v); err != nil {
			return err
		}
	}

	return nil
}

func writePkg(w io.Writer, pkg *eval.SimpleEnv, pkgName string, mem *Mem) error {
	if _, err := fmt.Fprintf(w, "\t\t\t\"%s\": &eval.SimpleEnv{\n", pkgName); err != nil {
		return err
	}
	fmt.Fprint(w, "\t\t\t\tVars: map[string]reflect.Value{\n")
	for k, v := range pkg.Vars {
		if v.IsValid() {
			e := v.Elem()
			if e.Type().Kind() == reflect.Ptr {
				mem.ptrs[e.Pointer()] = fmt.Sprintf("%s.%s", pkgName, k)
			}
		}
		_, err := fmt.Fprintf(w, "\t\t\t\t\t\"%s\": reflect.ValueOf(&%s.%s),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(w, "\t\t\t\t},\n\t\t\t\tConsts: map[string]reflect.Value{\n")
	for k := range pkg.Consts {
		_, err := fmt.Fprintf(w, "\t\t\t\t\t\"%s\": reflect.ValueOf(%s.%s),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(w, "\t\t\t\t},\n\t\t\t\tFuncs: map[string]reflect.Value{\n")
	for k := range pkg.Funcs {
		_, err := fmt.Fprintf(w, "\t\t\t\t\t\"%s\": reflect.ValueOf(%s.%s),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(w, "\t\t\t\t},\n\t\t\t\tTypes: map[string]reflect.Type{\n")
	for k := range pkg.Types {
		_, err := fmt.Fprintf(w, "\t\t\t\t\t\"%s\": reflect.TypeOf(new(%s.%s)).Elem(),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(w, "\t\t\t\t},\n\t\t\t},\n")
	return nil
}

func writeImport(w io.Writer, pkg *ast.Package) error {
	var files []*ast.File
	for _, f := range pkg.Files {
		if ast.FileExports(f) {
			files = append(files, f)
		}
	}
	env := eval.MakeSimpleEnv()
	for _, f := range files {
		for _, d := range f.Decls {
			if gen, ok := d.(*ast.GenDecl); ok {
				for _, s := range gen.Specs {
					if v, ok := s.(*ast.ValueSpec); ok {
						for _, i := range v.Names {
							if gen.Tok == token.VAR {
								env.Vars[i.Name] = reflect.Value{}
							} else {
								env.Consts[i.Name] = reflect.Value{}
							}
						}
					} else if t, ok := s.(*ast.TypeSpec); ok {
						env.Types[t.Name.Name] = nil
					}
				}
			} else if fun, ok := d.(*ast.FuncDecl); ok {
				env.Funcs[fun.Name.Name] = reflect.Value{}
			}
		}
	}
	return writePkg(w, env, pkg.Name, nil)
}

func sprintLiteral(v reflect.Value, mem *Mem) (string, error) {
	t := v.Type()
	k := t.Kind()

	var untyped string
	switch k {
	case reflect.Bool:
		untyped = fmt.Sprint(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		untyped = fmt.Sprint(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		untyped = fmt.Sprint(v.Uint())
	case reflect.Float32, reflect.Float64:
		untyped = fmt.Sprint(v.Float())
	case reflect.Complex64, reflect.Complex128:
		untyped = fmt.Sprint(v.Complex())
	case reflect.String:
		untyped = strconv.Quote(v.String())
	case reflect.Array:
		lit, err := sprintCompositeLit(v, mem, arraySliceElemPrinter)
		return fmt.Sprintf("[...]%v%s", t.Elem(), lit), err
	case reflect.Slice:
		if v.IsNil() {
			return fmt.Sprintf("*new(%v)", t), nil
		}
		lit, err := sprintCompositeLit(v, mem, arraySliceElemPrinter)
		return fmt.Sprintf("%v%s", t, lit), err
	case reflect.Struct:
		for i := 0; i < t.NumField(); i += 1 {
			n := t.Field(i).Name
			if unicode.IsLower([]rune(n)[0]) {
				return "", errors.New(
					fmt.Sprintf("struct %v has private field '%v'\n", n))
			}
		}
		lit, err := sprintCompositeLit(v, mem, structFieldPrinter)
		return fmt.Sprintf("%v%s", t, lit), err
	case reflect.Map:
		if v.IsNil() {
			return fmt.Sprintf("*new(%v)", t), nil
		}
		keys := v.MapKeys()
		lit, err := sprintCompositeLit(v, mem, func (v reflect.Value, i int, mem *Mem) (string, error, bool) {
			if i >= len(keys) {
				return "", nil, false
			} else if k, err := sprintLiteral(keys[i], mem); err != nil {
				return "", err, false
			} else if v, err := sprintLiteral(v.MapIndex(keys[i]), mem); err != nil {
				return "", err, false
			} else {
				return fmt.Sprintf("%s: %s", k, v), nil, true
			}
		})
		return fmt.Sprintf("%v%s", t, lit), err
	case reflect.Ptr:
		if prev, ok := mem.ptrs[v.Pointer()]; ok {
			return prev, nil
		}

		var err error
		if untyped, err = sprintLiteral(v.Elem(), mem); err != nil {
			return "", err
		}
		ptrvar := fmt.Sprintf("ptr%d", len(mem.vars))
		ptrval := fmt.Sprintf("[]%v{&%s}[0]", t, untyped)
		mem.ptrs[v.Pointer()] = ptrvar
		mem.vars[ptrvar] = ptrval
		return ptrvar, nil
	case reflect.Interface:
		if v.IsNil() {
			return fmt.Sprintf("*new(%v)", t), nil
		}
		var err error
		if untyped, err = sprintLiteral(v.Elem(), mem); err != nil {
			return "", err
		}
	default:
		return "", errors.New(fmt.Sprintf("%v literals cannot be represented as go src", k))
	}
	return fmt.Sprintf("[]%v{%s}[0]", t, untyped), nil
}

func arraySliceElemPrinter(v reflect.Value, i int, mem *Mem) (string, error, bool) {
	if i >= v.Len() {
		return "", nil, false
	}
	s, err := sprintLiteral(v.Index(i), mem)
	return s, err, true
}

func structFieldPrinter(v reflect.Value, i int, mem *Mem) (string, error, bool) {
	if i >= v.NumField() {
		return "", nil, false
	}
	s, err := sprintLiteral(v.Field(i), mem)
	return s, err, true
}

type compositeElemPrinter func(reflect.Value, int, *Mem) (string, error, bool)
func sprintCompositeLit(v reflect.Value, mem *Mem, f compositeElemPrinter) (string, error) {
	s, sep, i := "", "{", 0
	elem, err, cont := f(v, i, mem)
	for cont && err == nil {
		s += sep + elem
		sep = ", "
		i += 1
		elem, err, cont = f(v, i, mem)
	}
	if sep == "{" {
		s += "{"
	}
	return s + "}", err
}

