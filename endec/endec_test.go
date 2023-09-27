package endec

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

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
	// pwd := []byte("")
	os.Remove("datatest/d2/f2.txt.zip")
	err := GzipFile("datatest/d2/f2.txt.zip", "datatest/d1/f2.txt", false, -1)
	require.Nil(t, err)
	err = GunzipFile("datatest/d2/f2.txt", "datatest/d2/f2.txt.zip", false)
	require.Nil(t, err)
	buf := bytes.NewBuffer([]byte{})
	err = GzipFile(buf, "datatest/d1/f2.txt", false, -1)
	require.Nil(t, err)
	fmt.Print(buf.String())
}

func TestEncPass(t *testing.T) {
	// pwd := []byte("")
	viper.Reset()
	pathFile := "/home/user/workspace/go/src/stini/cmd/staticembed/varenc.json-En8Mi1ac6pV8X50p"
	if dataDecoded, e := DecryptFileWithPasswordInFileToBytes(pathFile); e == nil {
		fmt.Println(string(dataDecoded))
		viper.SetConfigType(strings.TrimPrefix(strings.Split(filepath.Ext(filepath.Base(pathFile)), "-")[0], "."))
		readerData := strings.NewReader(string(dataDecoded))
		if err := viper.ReadConfig(readerData); err == nil {
			if err = viper.Unmarshal(&Gvar); err == nil {
				fmt.Printf("%+v", Gvar)
				viper.Reset()
			}
		}
	}
}

var Gvar struct {
	TestVar string `mapstructure:"test_var"`
}

func TestEncPass1(t *testing.T) {
	// pwd := []byte("")
	dataDecoded := `{
    "test_var": "from varenc"
}`
	readerData := strings.NewReader(string(dataDecoded))
	v := viper.New()

	v.SetDefault("test_var", "default")
	v.SetConfigType("toml")
	var err error
	if err = v.ReadConfig(readerData); err == nil {
		if err = v.Unmarshal(&Gvar); err == nil {
			fmt.Printf("%v", Gvar)
			// v.Reset()
		}
	} else {
		fmt.Println(err)
	}
}
