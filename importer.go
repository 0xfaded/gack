package gack

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"path"

	"go/ast"
	"go/parser"
	"go/token"
)

func Import(pkgPath string) (*ast.Package, error) {
	fset := token.NewFileSet()
	if pkgs, err := parser.ParseDir(fset, pkgPath, testFilter, 0); err != nil {
		return nil, err
	} else if len(pkgs) == 0 {
		return nil, errors.New(fmt.Sprintf("no buildable Go source files in %s", pkgPath))
	} else if len(pkgs) != 1 {
		// The actual error message produced by gc lists the packages in the order
		// they were found, usually alphabetically by file name. This is cumbersome to
		// replicate using the parse api, so here the packages are listed alphabetically.
		keys := make([]string, len(pkgs))
		i := 0
		for k := range pkgs {
			keys[i] = k
			i += 1
		}
		sort.Strings(keys)
		var namess [2][]string
		for j := range namess {
			files := pkgs[keys[j]].Files
			namess[j] = make([]string, len(files))
			i = 0
			for n := range files {
				namess[j][i] = path.Base(n)
				i += 1
			}
			sort.Strings(namess[j])
		}
		return nil, errors.New(fmt.Sprintf("found pacakages %s (%s) and %s (%s) in %s",
			keys[0], namess[0][0], keys[1], namess[1][0], pkgPath))
	} else {
		for _, pkg := range pkgs {
			return pkg, nil
		}
		// Keep the compiler happy
		panic("impossible")
	}
}

func testFilter(fi os.FileInfo) bool {
	return !strings.HasSuffix(fi.Name(), "_test.go")
}
