package filecp

import (
	"testing"
)

//go test -v -run TestA
func TestA(c *testing.T) {
	t := NewCpTask("./test1", "./test1.bak")
	t.Copy()
}
