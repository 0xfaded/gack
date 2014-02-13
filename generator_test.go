package gack

import (
	"os"
	"testing"
)

func TestWriteImport(t *testing.T) {
	p, _ := Import("/home/crc/src/gopath/src//github.com/0xfaded/gack")
	writeImport(os.Stdout, p, "gack")
}
