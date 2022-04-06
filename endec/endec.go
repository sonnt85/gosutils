package endec

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"strings"

	//	"encoding/base64"
	"encoding/base64"
	"encoding/hex"
	"io/ioutil"

	//	"fmt"
	"io"
	"os"
)

func CreateHash(key []byte) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func EncryptHash(data []byte, passphrase []byte) (retbytes []byte, err error) {
	block, _ := aes.NewCipher([]byte(CreateHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return retbytes, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return retbytes, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func DecryptHash(data []byte, passphrase []byte) (retbytes []byte, err error) {
	key := []byte(CreateHash(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		return retbytes, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return retbytes, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return retbytes, err
	}
	return plaintext, nil
}

func EncryptHashFile(filename string, data []byte, passphrase []byte) (err error) {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	byteread, err := EncryptHash(data, passphrase)
	if err != nil {
		return err
	}
	_, err = f.Write(byteread)
	return err
}

func DecryptFile(filename string, passphrase []byte) (retbytes []byte, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return retbytes, err
	}
	return DecryptHash(data, passphrase)
}

func StringSimpleEncrypt(input, key string) (output string) {
	for i := 0; i < len(input); i++ {
		output += string(input[i] ^ key[i%len(key)])
	}
	output = base64.StdEncoding.EncodeToString([]byte(output))
	return strings.TrimRight(output, "=")
}

func StringSimpleDecrypt(input, key string) (output string, err error) {
	data := []byte{}
	input = strings.TrimRight(input, "=")
	for i := 0; i < 3; i++ {
		data, err = base64.StdEncoding.DecodeString(input)
		if err == nil {
			break
		} else {
			input = input + "="
		}
	}
	if err != nil {
		return "", err
	}

	for i := 0; i < len(data); i++ {
		output += string(data[i] ^ key[i%len(key)])
	}
	return output, nil
}

func StringZip(input []byte) (retstring string, err error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err = gz.Write([]byte(input)); err != nil {
		return
	}
	if err = gz.Flush(); err != nil {
		return
	}
	if err = gz.Close(); err != nil {
		return
	}
	retstring = base64.StdEncoding.EncodeToString(b.Bytes())
	return strings.TrimRight(retstring, "="), nil
}

func StringUnzip(input string) (data []byte, err error) {
	input = strings.TrimRight(input, "=")
	for i := 0; i < 3; i++ {
		data, err = base64.StdEncoding.DecodeString(input)
		if err == nil {
			break
		} else {
			input = input + "="
		}
	}

	if err != nil {
		return
	}
	rdata := bytes.NewReader(data)
	if r, err := gzip.NewReader(rdata); err == nil {
		data, err := ioutil.ReadAll(r)
		return data, err
	} else {
		return data, err
	}
}

func CompressFile(dst, src string, removeSrc bool) (err error) {
	var fw, fr *os.File
	fr, err = os.Open(src)
	if err != nil {
		return
	}
	defer func() {
		fr.Close()
		if err == nil && removeSrc {
			err = os.Remove(src)
		}
	}()

	fw, err = os.Create(dst)
	if err != nil {
		return err
	}
	defer fw.Close()
	w := gzip.NewWriter(fw)
	if _, err = io.Copy(w, fr); err != nil {
		return err
	}
	w.Close()
	return
}

// compressLogFile compresses the given log file, removing the
// uncompressed log file if successful.
func CompressLogFile(src, dst string) (err error) {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer f.Close()

	fi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat log file: %v", err)
	}

	// if err := chown(dst, fi); err != nil {
	// 	return fmt.Errorf("failed to chown compressed log file: %v", err)
	// }

	// If this file already exists, we presume it was created by
	// a previous attempt to compress the log file.
	gzf, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fi.Mode())
	if err != nil {
		return fmt.Errorf("failed to open compressed log file: %v", err)
	}
	defer gzf.Close()

	gz := gzip.NewWriter(gzf)

	defer func() {
		if err != nil {
			os.Remove(dst)
			err = fmt.Errorf("failed to compress log file: %v", err)
		}
	}()

	if _, err := io.Copy(gz, f); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := gzf.Close(); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return err
	}

	return nil
}
