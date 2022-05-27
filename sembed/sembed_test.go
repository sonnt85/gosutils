package sembed

import (
	"embed"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed statictest
var efs embed.FS

func printFsInfo(finfo []fs.FileInfo) {
	for _, v := range finfo {
		fmt.Println(v.Name())
	}

}
func TestHttpSystemFS(t *testing.T) {
	var retbyte = make([]byte, 100)
	var err error
	fs := NewHttpSystemFS(&efs, "statictest")
	// fmt.Println(fs.FindFilesMatchRegexpPathFromRoot("/dir1", "hello*", -1, true, true))
	err = fs.Copy(`./`, "/dir1")
	require.Nil(t, err)
	// return
	f, err := fs.Open(`dir1`)
	require.Nil(t, err)
	finfos, err := f.Readdir(0)
	require.Nil(t, err)
	printFsInfo(finfos)
	// fmt.Printf("%#v", finfos)
	// return
	// fs.Setsub("statictest")
	retbyte, err = fs.ReadFile("/dir1/hello1.txt")
	require.Nil(t, err)
	fmt.Println(string(retbyte))
	return
	f, err = fs.Open("hello.txt")
	f.Read(retbyte)
	require.Nil(t, err)
	fmt.Println(string(retbyte))
	hsy, err := fs.Open("dir1")
	require.Nil(t, err)
	finfos, err = hsy.Readdir(0)
	require.Nil(t, err)
	fmt.Println(finfos)
}
