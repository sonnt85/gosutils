package funcmap

import (
	"fmt"
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
	str12 := str1 + str2
	fmt.Println(str12)
	return 9, str12
}
func TestTask(t *testing.T) {
	task, err := NewTask[uint32]("concatstring", nil, ftest, "xinchao ", "cac ban")
	require.Nil(t, err)
	retval, err := task.Call()
	var str string
	var ok bool
	// value := reflect.New(v.(reflect.Type)).Elem().Interface()
	if retval != nil {
		fmt.Println(retval)
	}
	// return
	fmt.Println("task id: ", task.Id)
	if str, ok = retval[0].(string); ok {
		fmt.Println(str)
		for k, v := range retval {
			fmt.Printf("[%d] %#v\n", k, v)
		}
	}

}
