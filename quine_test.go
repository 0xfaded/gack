package gack

import (
	"reflect"
	"testing"

	"github.com/0xfaded/eval"
)

func TestQuine(t *testing.T) {
	env := eval.MakeSimpleEnv()
	v0, v1, v2, v3, v4, v5 := 1, 1.4, 2 + 0i, "abc", byte(8), &[2]int{}
	v6 := S{1}
	v7 := []*S{&v6, &v6, &v6}
	v8 := map[*byte]*byte{&v4: &v4}
	env.Vars["v0"] = reflect.ValueOf(&v0)
	env.Vars["v1"] = reflect.ValueOf(&v1)
	env.Vars["v2"] = reflect.ValueOf(&v2)
	env.Vars["v3"] = reflect.ValueOf(&v3)
	env.Vars["v4"] = reflect.ValueOf(&v4)
	env.Vars["v5"] = reflect.ValueOf(&v5)
	env.Vars["v6"] = reflect.ValueOf(&v6)
	env.Vars["v7"] = reflect.ValueOf(&v7)
	env.Vars["v8"] = reflect.ValueOf(&v8)
	imports := []string{"github.com//0xfaded/gack"}
	Quine(env, imports)

}
