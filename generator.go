package gack

import (
	"fmt"
	"io"
	"reflect"

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

func WriteEnv(w io.Writer, env *eval.SimpleEnv, imports []string) error {
	if _, err := fmt.Fprintf(w, "\troot := MakeSimpleEnv()\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, "Pkgs: map[string]eval.Env{\n"); err != nil {
		return err
	}
	for name, pkg := range env.Pkgs {
		writePkg(w, pkg.(*eval.SimpleEnv), name)
	}
	return nil
}

func writePkg(w io.Writer, pkg *eval.SimpleEnv, pkgName string) error {
	if _, err := fmt.Fprintf(w, "\t\t\"%s\": &eval.SimpleEnv{\n", pkgName); err != nil {
		return err
	}
	fmt.Fprint(w, "\t\t\tVars: map[string]reflect.Value{\n")
	for k := range pkg.Vars {
		_, err := fmt.Fprintf(w, "\t\t\t\t\"%s\": reflect.ValueOf(&%s.%s),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(w, "\t\t\t},\n\t\t\tConsts: map[string]reflect.Value{\n")
	for k := range pkg.Consts {
		_, err := fmt.Fprintf(w, "\t\t\t\t\"%s\": reflect.ValueOf(%s.%s),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(w, "\t\t\t},\n\t\t\tFuncs: map[string]reflect.Value{\n")
	for k := range pkg.Funcs {
		_, err := fmt.Fprintf(w, "\t\t\t\t\"%s\": reflect.ValueOf(%s.%s),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(w, "\t\t\t},\n\t\t\tTypes: map[string]reflect.Type{\n")
	for k := range pkg.Types {
		_, err := fmt.Fprintf(w, "\t\t\t\t\"%s\": reflect.TypeOf(new(%s.%s)).Elem(),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(w, "\t\t\t},\n\t\t},\n")
	return nil
}

func writeImport(w io.Writer, pkg *ast.Package, pkgName string) error {
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
	return writePkg(w, env, pkgName)
}
