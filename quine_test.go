package gack

import (
	"testing"

	"github.com/0xfaded/eval"
)

func TestQuine(t *testing.T) {
	env := eval.MakeSimpleEnv()
	imports := []string{"github.com//0xfaded/gack"}
	history := []string{"a, b := 1, 2"}
	Quine(env, imports, history)

}
