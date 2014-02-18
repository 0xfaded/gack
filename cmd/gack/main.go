package main

import (
	"github.com/0xfaded/eval"
	"github.com/0xfaded/gack"
)

func main() {
	gack.Repl(eval.MakeSimpleEnv(), nil)
}

