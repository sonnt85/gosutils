package slogrus

import (
	"testing"

	"github.com/sonnt85/gosutils/sutils"
)

func TestSlogMsgJson(t *testing.T) {
	InitStandardLogger(LevelTrace, true, false, "testdata/testslog.log")
	// TraceStack("test trace")
	str := "0123456789\n"
	str = sutils.StringDuplicate(str, 1023)
	Print(str)
	// for i := 0; i < 1024; i++ {
	// 	Print(str)
	// }
	// b := bytes.Repeat([]byte{'c'}, 1024*11)
	// Print(string(b))
	Flush()
	// require.Nil(t, err)
}
