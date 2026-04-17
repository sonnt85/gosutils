package slogrus

import (
	"testing"

	"github.com/sonnt85/gosutils/sutils"
)

func TestSlogMsgJson(t *testing.T) {
	InitStandardLogger(LevelTrace, true, false, "testdata/testslog.log")
	str := "0123456789\n"
	str = sutils.StringDuplicate(str, 1023)
	Print(str)
	Flush()
}
