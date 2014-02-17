package gacklib

import (
	"errors"

	"go/ast"
	"go/build"

	"fmt"
	"path"
	"strconv"
	"strings"
	"syscall"

	"io/ioutil"

	"os"
	"os/exec"

	"github.com/0xfaded/eval"
)

func Quine(env *eval.SimpleEnv, imports, history []string, deleteMe bool) error {
	required := map[string]bool{
		"reflect" : true,
		"github.com/0xfaded/eval" : true,
		"github.com/0xfaded/gacklib" : true,
	}
	if deleteMe {
		required["os"] = true
	}

	for _, i := range env.Pkgs {
		required[i.(*eval.SimpleEnv).Path] = true
	}

	f, err := ioutil.TempFile("/tmp", "gack")
	if err != nil {
		return err
	}
	srcDeleted := false
	_ = srcDeleted
	// if the exec is successful this will never be run
	defer (func() {
		if !srcDeleted {
			f.Close()
			os.Remove(f.Name())
		}
	})()

	if _, err = fmt.Fprint(f, "package main\nimport (\n"); err != nil {
		return err
	}

	imported := map[string]bool{}
	names := map[string]string{}
	pkgs := make(map[string]*ast.Package, len(imports))
	for _, i := range(imports) {
		if absolute, clean, err := findImport(i); err != nil {
			return err
		} else if pkg, err := Import(absolute); err != nil {
			return err
		} else if at, ok := names[pkg.Name]; ok {
			return fmt.Errorf("%v redeclared as imported package name\n" +
				"\tprevious declaration at %v", pkg.Name, at)
		} else if _, err := fmt.Fprintf(f, "\t%s\n", strconv.Quote(clean)); err != nil {
			return err
		} else {
			imported[clean] = true
			pkgs[clean] = pkg
			for f := range pkg.Files {
				names[pkg.Name] = f
				break
			}
		}
	}
	for  r := range required {
		if !imported[r] {
			if _, err := fmt.Fprintf(f, "\t%s\n", strconv.Quote(r)); err != nil {
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


	if _, err := fmt.Fprint(f, "\thistory := []string{}\n"); err != nil {
		return err
	}

	// Replay the previous session
	for _, h := range history {
		h := strconv.Quote(h)
		if _, err := fmt.Fprintf(f, "\teval.Interpret(%s, root)\n\thistory = append(history, %s)\n", h, h); err != nil {
			return err
		}
	}

	// Delete the previous binary. history == nil implies this is the first invokation,
	// we ought not to delete that one ;p
	if deleteMe {
		if _, err := fmt.Fprintf(f, "\tos.Remove(%s)\n", strconv.Quote(os.Args[0])); err != nil {
			return err
		}
	}

	// Enter the repl
	if _, err := fmt.Fprint(f, "\tgacklib.Repl(root, history)\n}"); err != nil {
		return err
	}

	// Compile the new program
	o, err := ioutil.TempFile("/tmp", "gack")
	if err != nil {
		return err
	}
	o.Close()

	srcDeleted = true
	f.Close()

	compiler := path.Join(build.ToolDir, "8g")
	linker := path.Join(build.ToolDir, "8l")
	if strings.HasPrefix(build.Default.GOARCH, "amd64") {
		compiler = path.Join(build.ToolDir, "6g")
		linker = path.Join(build.ToolDir, "6l")
	}

	platform := build.Default.GOOS + "_" + build.Default.GOARCH
	gopathlibs := path.Join(os.Getenv("GOPATH"), "pkg", platform)
	gorootlibs := path.Join(os.Getenv("GOROOT"), "pkg", platform)
	cmd := exec.Command(compiler, "-o", o.Name(), "-I", gopathlibs, "-I", gorootlibs, f.Name())
	if output, err := cmd.Output(); err != nil {
		fmt.Fprintf(os.Stdout, "Generated src failed to compile. Please file a bug report " +
			"with %s attached\n", f.Name())
		os.Stdout.Write(output)
		return err
	}

	// Delete the generated source
	os.Remove(f.Name())

	e, err := ioutil.TempFile("/tmp", "gack")
	if err != nil {
		return err
	}
	e.Close()
	cmd = exec.Command(linker, "-o", e.Name(), "-L", gopathlibs, "-L", gorootlibs, o.Name())
	if output, err := cmd.Output(); err != nil {
		fmt.Fprintf(os.Stdout, "Generated src failed to compile. Please file a bug report " +
			"with %s attached\n", f.Name())
		os.Stdout.Write(output)
		return err
	}

	// Delete the object file
	os.Remove(o.Name())

	// Go for the kill :)
	return syscall.Exec(e.Name(), []string{e.Name()}, os.Environ())
}

func deleteSelf() {
	if rm, err := exec.LookPath("rm"); err == nil {
		syscall.Exec(rm, []string{rm, os.Args[0]}, os.Environ())
	}
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
	gopath := path.Join(os.Getenv("GOPATH"), "src", clean)
	if fi, _ := os.Stat(gopath); fi != nil && fi.IsDir() {
		return gopath, clean, nil
	}
	goroot := path.Join(os.Getenv("GOROOT"), "src", "pkg", clean)
	if fi, _ := os.Stat(goroot); fi != nil && fi.IsDir() {
		return goroot, clean, nil
	}
	return "", "", fmt.Errorf(`cannot find package "%s" in any of:
	%s (from $GOROOT)
	%s (from $GOPATH)`, clean, goroot, gopath)
}

