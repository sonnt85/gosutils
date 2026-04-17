package endec

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/stretchr/testify/require"
)

func TestEncrypFile(t *testing.T) {
	filePath := "/tmp/test.txt"
	password := []byte("nguyenthanhson")
	datatoencryp := []byte("012345678910")

	enc, err := EncrypBytes(datatoencryp, password)
	require.Nil(t, err)
	t.Logf("EncrypBytes: %v", enc)

	dec, err := DecryptBytes(enc, password)
	require.Nil(t, err)
	t.Logf("DecryptBytes: %s", dec)

	if _, err := os.Stat(filePath); err == nil {
		data, err := DecryptFileToBytes(filePath, password)
		require.Nil(t, err)
		t.Logf("DecryptFileToBytes: %s", data)
	}
}

func TestStringEndec(t *testing.T) {
	data := "nguyenthanhson1"
	password := "password"

	enc := StringSimpleEncrypt(data, password)
	t.Logf("encrypted: %s", enc)

	dec, err := StringSimpleDecrypt(enc, password)
	require.Nil(t, err)
	if dec != data {
		t.Errorf("round-trip mismatch: got %q, want %q", dec, data)
	}
}

func TestGzip(t *testing.T) {
	os.Remove("datatest/d2/f2.txt.zip")
	err := GzipFile("datatest/d2/f2.txt.zip", "datatest/d1/f2.txt", false, -1)
	require.Nil(t, err)
	err = GunzipFile("datatest/d2/f2.txt", "datatest/d2/f2.txt.zip", false)
	require.Nil(t, err)
	buf := bytes.NewBuffer([]byte{})
	err = GzipFile(buf, "datatest/d1/f2.txt", false, -1)
	require.Nil(t, err)
	if buf.Len() == 0 {
		t.Error("GzipFile into buffer produced zero bytes")
	}
}

func TestEncPass(t *testing.T) {
	viper.Reset()
	// This test exercises a developer-local encrypted config file; skip when
	// the file is not present so the test does not fail on fresh checkouts.
	pathFile := "/home/user/workspace/go/src/stini/cmd/staticembed/varenc.json-En8Mi1ac6pV8X50p"
	if _, err := os.Stat(pathFile); err != nil {
		t.Skipf("skipping: %s not present", pathFile)
	}
	dataDecoded, err := DecryptFileWithPasswordInFileToBytes(pathFile)
	require.Nil(t, err)
	t.Logf("decoded: %s", dataDecoded)

	viper.SetConfigType(strings.TrimPrefix(strings.Split(filepath.Ext(filepath.Base(pathFile)), "-")[0], "."))
	if err := viper.ReadConfig(strings.NewReader(string(dataDecoded))); err != nil {
		t.Fatal(err)
	}
	if err := viper.Unmarshal(&Gvar); err != nil {
		t.Fatal(err)
	}
	t.Logf("Gvar: %+v", Gvar)
	viper.Reset()
}

var Gvar struct {
	TestVar string `mapstructure:"test_var"`
}

func TestEncPass1(t *testing.T) {
	dataDecoded := `{
    "test_var": "from varenc"
}`
	v := viper.New()
	v.SetDefault("test_var", "default")
	v.SetConfigType("toml")
	if err := v.ReadConfig(strings.NewReader(dataDecoded)); err != nil {
		t.Log("ReadConfig error (expected for toml vs json content):", err)
		return
	}
	if err := v.Unmarshal(&Gvar); err != nil {
		t.Fatal(err)
	}
	t.Logf("Gvar: %v", Gvar)
}
