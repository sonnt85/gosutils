package sexec

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExecBytes(t *testing.T) {
	path, err := exec.LookPath("env")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	out, _, err = ExecBytesEnvTimeout(b, "name", map[string]string{"TESTENV": "testenv"}, time.Second)
	require.Nil(t, err)
	fmt.Println(string(out))

}
