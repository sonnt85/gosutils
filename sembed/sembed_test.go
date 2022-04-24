package sembed

import (
	"embed"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed statictest
var efs embed.FS

func TestHttpSystemFS(t *testing.T) {
	var retbyte = make([]byte, 100)
	var err error
	fs := NewHttpSystemFS(&efs, "statictest")
	// fs.Setsub("statictest")
	retbyte, err = fs.ReadFile("dir1/hello1.txt")
	require.Nil(t, err)
	fmt.Println(string(retbyte))
	return
	f, err := fs.Open("hello.txt")
	f.Read(retbyte)
	require.Nil(t, err)
	fmt.Println(string(retbyte))
	hsy, err := fs.Open("statictest")
	require.Nil(t, err)
	finfos, err := hsy.Readdir(1)
	require.Nil(t, err)
	fmt.Println(finfos)
}
