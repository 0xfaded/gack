package gack

import (
	"fmt"
	"testing"
)

func TestImporter(t *testing.T) {
	p, err := Import("/home/crc/src/gopath/src//github.com/0xfaded/gack")
	fmt.Printf("%v %v\n", p, err)
	p, err = Import("/home/crc/src/gopath/src/play/foobar")
	fmt.Printf("%v %v\n", p, err)
}
