package endec

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	//	"encoding/base64"
	"encoding/base64"
	"io/ioutil"

	//	"fmt"
	"io"
	"os"
)

func EncrypBytes(data []byte, passphrase []byte) (retbytes []byte, err error) {
	block, _ := aes.NewCipher(MD5Bytes(passphrase))
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

func EncrypBytesToString(data []byte, passphrase []byte) (retbstring string, err error) {
	var gcm cipher.AEAD
	block, _ := aes.NewCipher(MD5Bytes(passphrase))
	gcm, err = cipher.NewGCM(block)
	if err != nil {
		return
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	return base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

// enctyp file filename to byte array use hash
func EncryptBytesToFile(filename string, data []byte, passphrase []byte) (err error) {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	byteread, err := EncrypBytes(data, passphrase)
	if err != nil {
		return err
	}
	_, err = f.Write(byteread)
	return err
}

func DecryptBytes(data []byte, passphrase []byte) (retbytes []byte, err error) {
	key := MD5Bytes(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return retbytes, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return retbytes, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < (nonceSize + 1) {
		return retbytes, errors.New("data length not guaranteed")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return retbytes, err
	}
	return plaintext, nil
}

func Base64Encode(data []byte) string {
	return base64.RawStdEncoding.EncodeToString(data)
}

func Base64Decode(datastr string) ([]byte, error) {
	return base64.RawStdEncoding.DecodeString(datastr)
}

func DecryptBytesFromString(datastr string, passphrase []byte) (retbytes []byte, err error) {
	var data []byte
	data, err = base64.RawStdEncoding.DecodeString(datastr)
	if err != nil {
		return
	}
	return DecryptBytes(data, passphrase)
}

// decryp file filename to byte array use hash
func DecryptFileToBytes(filename string, passphrase []byte) (retbytes []byte, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return retbytes, err
	}
	return DecryptBytes(data, passphrase)
}

// decryp file filename to byte array use hash
func DecryptFileToFile(inputFile, outputFile string, passphrase []byte) (err error) {
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return err
	}
	var inputInfo os.FileInfo
	inputInfo, err = os.Stat(inputFile)
	if err != nil {
		return
	}
	data, err = DecryptBytes(data, passphrase)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(outputFile, data, inputInfo.Mode().Perm())
	return
}

func StringSimpleEncrypt(input, key string) (output string) {
	for i := 0; i < len(input); i++ {
		output += string(input[i] ^ key[i%len(key)])
	}
	output = base64.RawStdEncoding.EncodeToString([]byte(output))
	return
	// return strings.TrimRight(output, "=")
}

func StringSimpleDecrypt(input, key string) (output string, err error) {
	data := []byte{}
	// data, err = base64.StdEncoding.DecodeString(input)
	data, err = base64.RawStdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(data); i++ {
		output += string(data[i] ^ key[i%len(key)])
	}
	return output, nil
}

func BytesZipToString(input []byte) (retstring string, err error) {
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
	retstring = base64.RawStdEncoding.EncodeToString(b.Bytes())
	return strings.TrimRight(retstring, "="), nil
}

func StringUnzip(input string) (data []byte, err error) {
	input = strings.TrimRight(input, "=")
	for i := 0; i < 3; i++ {
		data, err = base64.RawStdEncoding.DecodeString(input)
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

func gunzipWrite(w io.Writer, data []byte) error {
	// Write gzipped data to the client
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	defer gr.Close()
	data, err = ioutil.ReadAll(gr)
	if err != nil {
		return err
	}
	w.Write(data)
	return nil
}

func GunzipFile(newfilename, gzipfilePath string, removeZipFile bool) (err error) {
	var gzipfile, writer *os.File
	var reader *gzip.Reader
	defer func() {
		if err == nil && removeZipFile {
			err = os.Remove(gzipfilePath)
		}
	}()
	gzipfile, err = os.Open(gzipfilePath)

	if err != nil {
		return
	}

	reader, err = gzip.NewReader(gzipfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer reader.Close()

	writer, err = os.Create(newfilename)

	if err != nil {
		return
	}

	defer writer.Close()

	_, err = io.Copy(writer, reader)
	return
}

// max compressLevel is 9
func ZipFile(dst, src string, removeSrc bool, compressLevel ...int) (err error) {
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

	var inputInfo os.FileInfo
	inputInfo, err = os.Stat(src)
	if err != nil {
		return
	}
	fw, err = os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, inputInfo.Mode().Perm())
	// fw, err = os.Create(dst)
	if err != nil {
		return err
	}
	defer fw.Close()
	clv := 9
	if len(compressLevel) != 0 && (compressLevel[0] <= gzip.BestCompression && compressLevel[0] >= gzip.HuffmanOnly) { //auto compressLevel
		clv = compressLevel[0]
	}
	w, _ := gzip.NewWriterLevel(fw, clv)
	if _, err = io.Copy(w, fr); err != nil {
		return err
	}
	w.Close()
	return
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}
	return b, nil
}

func RandUnt64() uint64 {
	val, err := rand.Int(rand.Reader, big.NewInt(int64(math.MaxInt64)))
	if err != nil {
		return 0
	}
	return val.Uint64()
}

func RandUnt32() uint32 {
	val, err := rand.Int(rand.Reader, big.NewInt(int64(math.MaxInt64)))
	if err != nil {
		return 0
	}
	return uint32(val.Uint64())
}

func RandInt32() int32 {
	return int32(RandUnt32())
}

func RandInt() int {
	return int(RandUnt32())
}

func Randint64() int64 {
	return int64(RandUnt64())
}

func RandRangeInterger(from, to int) int {
	delta := to - from
	if delta == 0 {
		return from
	}
	return from + RandInt()%delta
}

func RandRangeInt64(from, to int64) int64 {
	delta := to - from
	if delta == 0 {
		return from
	}
	return from + Randint64()%delta
}
