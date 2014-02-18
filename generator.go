package gack

import (
	"fmt"
	"io"
	"reflect"
	"strconv"

	"go/ast"
	"go/token"

	"github.com/0xfaded/eval"
)

func WriteEnv(w io.Writer, env *eval.SimpleEnv, imports map[string]*ast.Package) error {
	// All pointers that point to the same place in the previous env must point to the
	// same place in the new env. We can't simply make a new value for each pointer as
	// we can for other types. Even worse, the pointer may point somewhere inside a compiled
	// package. Every pointer in the environment must be tracked and duplicates detected.

	_, err := fmt.Fprint(w,
`	root := &eval.SimpleEnv{
		Vars: map[string]reflect.Value{},
		Consts: map[string]reflect.Value{},
		Funcs: map[string]reflect.Value{},
		Types: map[string]reflect.Type{},
		Pkgs: map[string]eval.Env{
`);
	if err != nil {
		return err
	}
	for pkgPath, pkg := range imports {
		if err := writeImport(w, pkg, pkgPath); err != nil {
			return err
		}
	}
	for name, pkg := range env.Pkgs {
		if err := writePkg(w, pkg.(*eval.SimpleEnv), name); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\t\t},\n\t}\n"); err != nil {
		return err
	}
	return nil
}

func writePkg(w io.Writer, pkg *eval.SimpleEnv, pkgName string) error {
	if _, err := fmt.Fprintf(w, "\t\t\t\"%s\": &eval.SimpleEnv{\n", pkgName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\t\t\t\tPath: %s,\n", strconv.Quote(pkg.Path)); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, "\t\t\t\tVars: map[string]reflect.Value{\n"); err != nil {
		return err
	}
	for k := range pkg.Vars {
		_, err := fmt.Fprintf(w, "\t\t\t\t\t\"%s\": reflect.ValueOf(&%s.%s),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\t\t\t\t},\n\t\t\t\tConsts: map[string]reflect.Value{\n"); err != nil {
		return err
	}
	for k := range pkg.Consts {
		_, err := fmt.Fprintf(w, "\t\t\t\t\t\"%s\": reflect.ValueOf(%s.%s),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\t\t\t\t},\n\t\t\t\tFuncs: map[string]reflect.Value{\n"); err != nil {
		return err
	}
	for k := range pkg.Funcs {
		_, err := fmt.Fprintf(w, "\t\t\t\t\t\"%s\": reflect.ValueOf(%s.%s),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\t\t\t\t},\n\t\t\t\tTypes: map[string]reflect.Type{\n"); err != nil {
		return err
	}
	for k := range pkg.Types {
		_, err := fmt.Fprintf(w, "\t\t\t\t\t\"%s\": reflect.TypeOf(new(%s.%s)).Elem(),\n", k, pkgName, k)
		if err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\t\t\t\t},\n\t\t\t},\n"); err != nil {
		return err
	}
	return nil
}

func writeImport(w io.Writer, pkg *ast.Package, pkgPath string) error {
	var files []*ast.File
	for _, f := range pkg.Files {
		if ast.FileExports(f) {
			files = append(files, f)
		}
	}
	env := eval.MakeSimpleEnv()
	env.Path = pkgPath
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
			} else if fun, ok := d.(*ast.FuncDecl); ok && fun.Recv == nil {
				env.Funcs[fun.Name.Name] = reflect.Value{}
			}
		}
	}
	return writePkg(w, env, pkg.Name)
}

