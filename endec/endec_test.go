package endec

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

//	"encoding/base64"

// "fmt"
func TestEncrypFile(t *testing.T) {
	var data, datatoencryp []byte
	var err error
	filePath := "/tmp/test.txt"
	password := []byte("nguyenthanhson")
	datatoencryp = []byte("012345678910")
	// password1 := []byte("nguyenthanhson1")
	data, err = EncrypBytes(datatoencryp, password)
	require.Nil(t, err)
	fmt.Println("EncrypBytes: ", data)

	data, err = DecryptBytes(data, password)
	require.Nil(t, err)
	fmt.Println("DecryptBytes ", string(data))

	// err = EncryptBytesToFile(filePath, datatoencryp, password)
	// require.Nil(t, err)
	// data, _ = ioutil.ReadFile(filePath)
	// fmt.Println("EncryptBytesToFile: ", data)

	data, err = DecryptFileToBytes(filePath, password)
	require.Nil(t, err)
	fmt.Println("DecryptFileToBytes: ", string(data))
}

func TestStringEndec(t *testing.T) {
	var err error
	var retdata string
	data := "nguyenthanhson1"
	password := "password"
	retdata = StringSimpleEncrypt(data, password)
	// require.Nil(t, err)
	fmt.Println(retdata)
	retdata, err = StringSimpleDecrypt(retdata, password)
	require.Nil(t, err)
	fmt.Println(retdata)
}

func TestGzip(t *testing.T) {
	pwd := []byte("pwd12")
	os.Remove("datatest/d2/f2.txt.zip")
	err := GzipFile("datatest/d2/f2.txt.zip", "datatest/d1/f2.txt", false, -1, pwd)
	require.Nil(t, err)
	err = GunzipFile("datatest/d2/f2.txt", "datatest/d2/f2.txt.zip", false, pwd)
	require.Nil(t, err)
	buf := bytes.NewBuffer([]byte{})
	err = GzipFile(buf, "datatest/d1/f2.txt", false, -1, pwd)
	require.Nil(t, err)
	fmt.Print(buf.String())
}
