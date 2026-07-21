package endec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZipEnc(t *testing.T) {
	pwd := []byte("pwd12")
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "d1.txt")
	zipFile := filepath.Join(tmpDir, "d1.zip")
	outDir := filepath.Join(tmpDir, "out")

	err := os.WriteFile(srcFile, []byte("zip test data"), 0600)
	require.Nil(t, err)

	err = ZipEncrypt(srcFile, zipFile, pwd)
	require.Nil(t, err)
	err = os.MkdirAll(outDir, 0700)
	require.Nil(t, err)
	err = ZipDecrypt(zipFile, outDir, false, pwd)
	require.Nil(t, err)
}
