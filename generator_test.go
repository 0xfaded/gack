package gack

import (
	"os"
	"reflect"
	"testing"

	"github.com/0xfaded/eval"
)

func TestSprintLiteralEnvPointers(t *testing.T) {
	env := eval.MakeSimpleEnv()
	a := new(int)
	b := a
	env.Vars["a"] = reflect.ValueOf(&a)
	env.Vars["b"] = reflect.ValueOf(&b)
	mem := &Mem{make(map[uintptr]string), make(map[string]string)}
	aa, _ := sprintLiteral(reflect.ValueOf(a), mem)
	bb, _ := sprintLiteral(reflect.ValueOf(b), mem)
	if bb != "ptr0" {
		t.Fatalf("Identical pointer b should reuse a '%s' '%s'", aa, bb)
	}
}

func TestSprintLiteralPkgPointers(t *testing.T) {
	env := eval.MakeSimpleEnv()
	a := &os.Stdout
	b := a
	mem := &Mem{make(map[uintptr]string), make(map[string]string)}
	mem.ptrs[reflect.ValueOf(&os.Stdout).Pointer()] = "os.Stdout"
	env.Vars["a"] = reflect.ValueOf(&a)
	env.Vars["b"] = reflect.ValueOf(&b)
	aa, _ := sprintLiteral(reflect.ValueOf(a), mem)
	bb, _ := sprintLiteral(reflect.ValueOf(b), mem)
	if !(aa == "os.Stdout" && bb == "os.Stdout") {
		t.Fatalf("Identical pointers have different identifiers '%s' '%s'", aa, bb)
	}
}

func TestSprintLiteralEnvInterfaces(t *testing.T) {
	env := eval.MakeSimpleEnv()
	a := new(int)
	var b interface{} = a
	mem := &Mem{make(map[uintptr]string), make(map[string]string)}
	env.Vars["a"] = reflect.ValueOf(&a)
	env.Vars["b"] = reflect.ValueOf(&b)
	bb, _ := sprintLiteral(reflect.ValueOf(&b).Elem(), mem)
	aa, _ := sprintLiteral(reflect.ValueOf(a), mem)
	if aa != "ptr0" {
		t.Fatalf("Identical pointers and interface should reuse identifiers '%s' '%s'", aa, bb)
	}
}

func TestSprintLiteralPkgInterfaces(t *testing.T) {
	env := eval.MakeSimpleEnv()
	var i interface{} = &os.Stdout
	mem := &Mem{make(map[uintptr]string), make(map[string]string)}
	mem.ptrs[reflect.ValueOf(&os.Stdout).Pointer()] = "os.Stdout"
	env.Vars["i"] = reflect.ValueOf(&i)
	ii, _ := sprintLiteral(reflect.ValueOf(&i).Elem(), mem)
	if ii != "[]interface {}{os.Stdout}[0]" {
		t.Fatalf("Interface to package pointer should use package ident '%s'", ii)
	}
}

