package gack

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"io/ioutil"
	"os"

	"go/ast"
	"go/build"
	"github.com/0xfaded/eval"
)

func Quine(env *eval.SimpleEnv, imports []string) error {
	required := []string{
		"reflect",
		"github.com/0xfaded/eval",
		"github.com/0xfaded/gack",
	}

	f, err := ioutil.TempFile("/tmp", "gack")
	if err != nil {
		return err
	}
	//defer os.Remove(f.Name())

	if _, err = fmt.Fprint(f, "package main\nimport (\n"); err != nil {
		return err
	}

	imported := map[string]bool{}
	pkgs := make(map[string]*ast.Package, len(imports))
	for _, i := range(imports) {
		if absolute, clean, err := findImport(i); err != nil {
			return err
		} else if pkg, err := Import(absolute); err != nil {
			return err
		} else if _, err := fmt.Fprintf(f, "\t\"%s\"\n", clean); err != nil {
			return err
		} else {
			imported[clean] = true
			pkgs[clean] = pkg
		}
	}
	for _, r := range required {
		if !imported[r] {
			if _, err := fmt.Fprintf(f, "\t\"%s\"\n", r); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprint(f, ")\nfunc main() {\n"); err != nil {
		return err
	}

	if err := WriteEnv(f, env, pkgs); err != nil {
		return err
	}

	if _, err := fmt.Fprint(f, "\tRepl(root)\n}"); err != nil {
		return err
	}


	/*
	// -e prints all errors
	cmd := exec.Command(build.ToolDir + "/8g", "-e", "-o", "/dev/null", f.Name())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	buf := bufio.NewReader(stdout)

	line, rerr := buf.ReadString('\n')
	for rerr == nil {
		if strings.Index(line, ": ") != -1 {
			// Remove filename prefix
			s := strings.SplitN(line, ": ", 2)[1]
			// Remove trailing \n
			s = s[:len(s)-1]
			compileErrors = append(compileErrors, s)
		}
		line, rerr = buf.ReadString('\n')
	}
	if rerr != io.EOF {
		return nil, rerr
	} else {
		return compileErrors, nil
	}
	*/
	return nil
}

func findImport(pkgPath string) (absolutePath, cleanedPkgPath string, err error) {
	// Spec allows unicode. Also, is there a better IsAscii somewhere?
	if len(pkgPath) == 0 || len(pkgPath) != len([]byte(pkgPath)) {
		return "", "", errors.New("bad package path: " + pkgPath)
	}
	parts := strings.Split(pkgPath, "/")
	if parts[0] == "" {
		return "", "", errors.New("cannot import absolute path: " + pkgPath)
	}

	for i := 0; i < len(parts); i += 1 {
		if parts[i] == "" {
			parts = append(parts[:i], parts[i+1:]...)
			i -= 1
		} else {
			parts[i] = strings.Trim(parts[i], " \n\t")
			if parts[i] == "" {
				return "", "", errors.New("bad import path: " + pkgPath)
			}
		}
	}
	clean := strings.Join(parts, "/")
	gopath := path.Join(build.Default.GOPATH, "src", clean)
	fmt.Printf("%v\n", gopath)
	if fi, _ := os.Stat(gopath); fi != nil && fi.IsDir() {
		return gopath, clean, nil
	}
	goroot := path.Join(build.Default.GOROOT, "src", clean)
	fmt.Printf("%v\n", goroot)
	if fi, _ := os.Stat(goroot); fi != nil && fi.IsDir() {
		return goroot, clean, nil
	}
	return "", "", errors.New(fmt.Sprintf(`cannot find package "%s" in any of:
	%s (from $GOROOT)
	%s (from $GOPATH)`, clean, goroot, gopath))
}
