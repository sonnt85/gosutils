package funcmap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testcases = map[string]interface{}{
		"hello":      func() { print("hello") },
		"foobar":     func(a, b, c int) int { return a + b + c },
		"errstring":  "Can not call this as a function",
		"errnumeric": 123456789,
	}
)

func ftest(str1, str2 string) (int, string) {
	return 9, str1 + str2
}

func TestTask(t *testing.T) {
	task, err := NewTask[uint32]("concatstring", nil, ftest, "xinchao ", "cac ban")
	require.Nil(t, err)

	retval, err := task.Call()
	require.Nil(t, err)
	require.NotNil(t, retval)
	t.Logf("task id: %d, retval: %v", task.Id, retval)

	num, ok := retval[0].(int)
	require.True(t, ok, "retval[0] is not an int")
	require.Equal(t, 9, num)

	str, ok := retval[1].(string)
	require.True(t, ok, "retval[1] is not a string")
	require.Equal(t, "xinchao cac ban", str)
}
