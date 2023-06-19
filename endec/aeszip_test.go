package endec

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZipEnc(t *testing.T) {
	pwd := []byte("pwd12")
	os.Remove("datatest/d2/d1.txt.zip")
	err := ZipEncrypt("/home/user/workspace/go/bin/gostools", "datatest/d2/d1.zip", pwd)
	require.Nil(t, err)
	err = ZipDecrypt("datatest/d2/d1.zip", "datatest/d2/", false, pwd)
	require.Nil(t, err)
}
