package endec

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
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

// enctyp file filename (string path file or io.Writer) to byte array use hash
func EncryptBytesToFile(filename interface{}, data []byte, passphrase []byte) (err error) {
	var f io.Writer
	// *os.File
	switch v := filename.(type) {
	case io.Writer:
		f = v
		// f.Truncate(0)
	case string:
		var file *os.File
		file, err = os.Create(v)
		if err != nil {
			return err
		}
		f = io.Writer(file)
		defer file.Close()
	default:
		return errors.New("param is of unknown type")
	}

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

// decryp file filename (String) or io.Reader to byte array use hash
func DecryptFileToBytes(filename interface{}, passphrase []byte) (data []byte, err error) {
	// *os.File
	switch v := filename.(type) {
	case io.Reader:
		data, err = io.ReadAll(v)
		// f.Truncate(0)
	case string:
		data, err = ioutil.ReadFile(v)
	default:
		return data, errors.New("param is of unknown type")
	}

	if err != nil {
		return data, err
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
	var data []byte
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

// This function GunzipFile takes in two parameters: newfilename and gzipfilePath, which can be either a file path or an io.Writer/io.Reader interface.
// It also takes in a boolean variable removeZipFile which indicates whether the original gzip file should be removed after decompression.
// Additionally, an optional password parameter can be passed as a byte slice to decrypt the gzip file if it is password-protected.
// The function returns an error if any issues arise during the decompression process.
// Overall, this function is useful for decompressing gzip files, optionally removing the original file, and decrypting password-protected files if necessary.
func GunzipFile(newfilename, gzipfilePath interface{}, removeZipFile bool, password ...[]byte) (err error) {
	var fw, fr *os.File
	var rd io.Reader
	var wt io.Writer
	var filemod os.FileMode
	filemod = 0755
	switch v := gzipfilePath.(type) {
	case io.Reader:
		rd = v
	case string:
		fr, err = os.Open(v)
		if err != nil {
			return
		}
		defer func() {
			fr.Close()
			if err == nil && removeZipFile {
				err = os.Remove(v)
			}
		}()
		var inputInfo os.FileInfo
		inputInfo, err = os.Stat(v)
		if err != nil {
			return
		}
		filemod = inputInfo.Mode().Perm()
		rd = fr
	default:
		return errors.New("param is of unknown type")
	}

	switch v := newfilename.(type) {
	case io.Writer:
		wt = v
	case string:
		fw, err = os.OpenFile(v, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filemod)
		if err != nil {
			return
		}
		wt = fw
		defer func() {
			fw.Close()
		}()
	default:
		return errors.New("param is of unknown type")
	}

	var reader *gzip.Reader

	var key []byte
	if len(password) != 0 && len(password[0]) != 0 {
		var block cipher.Block
		hash := sha256.Sum256(password[0])
		key = hash[:]
		block, err = aes.NewCipher(key)
		if err != nil {
			return err
		}

		iv := make([]byte, aes.BlockSize)
		_, err = io.ReadFull(rd, iv)
		if err != nil {
			return err
		}
		stream := cipher.NewCTR(block, iv)
		reader, err = gzip.NewReader(rd)
		if err != nil {
			return
		}
		defer reader.Close()
		readeren := &cipher.StreamReader{
			S: stream,
			R: reader,
		}
		_, err = io.Copy(wt, readeren)
	} else {
		reader, err = gzip.NewReader(rd)
		if err != nil {
			return
		}
		defer reader.Close()
		_, err = io.Copy(wt, reader)
	}
	return
}

func ZipFile(source, destination string, removeSrc bool) error {
	return GzipFile(destination, source, removeSrc, -1)
}

// GzipFile compresses the source file or input stream to the destination file or output stream using gzip compression.
// The maximum compressLevel is 9 (default).
// The "dst" and "src" parameters specify the file paths or the input/output streams to read/write the data.
// The "removeSrc" parameter indicates whether to remove the source file after compression.
// The "compressLevel" parameter sets the level of compression to be used (0-9, where 0 is no compression and 9 is maximum compression).
// The optional "password" parameter is a slice of bytes that represents the password to use for encryption, if any.
// The function returns an error if any operation fails.
func GzipFile(dst, src interface{}, removeSrc bool, compressLevel int, password ...[]byte) (err error) {
	var fw, fr *os.File
	var rd io.Reader
	var wt io.Writer
	var gzipWriter *gzip.Writer

	var filemod os.FileMode
	filemod = 0755
	switch v := src.(type) {
	case io.Reader:
		rd = v
	case string:
		fr, err = os.Open(v)
		if err != nil {
			return
		}
		defer func() {
			fr.Close()
			if err == nil && removeSrc {
				err = os.Remove(v)
			}
		}()
		var inputInfo os.FileInfo
		inputInfo, err = os.Stat(v)
		if err != nil {
			return
		}
		filemod = inputInfo.Mode().Perm()
		rd = fr
	default:
		return errors.New("param is of unknown type")
	}

	switch v := dst.(type) {
	case io.Writer:
		wt = v
	case string:
		fw, err = os.OpenFile(v, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filemod)
		if err != nil {
			return
		}
		wt = fw
		defer func() {
			fw.Close()
		}()
	default:
		return errors.New("param is of unknown type")
	}

	// fw, err = os.Create(dst)
	var key []byte
	if len(password) > 0 && len(password[0]) != 0 {
		hash := sha256.Sum256(password[0])
		key = hash[:]
	}
	clv := 9
	if compressLevel <= gzip.BestCompression && compressLevel >= gzip.HuffmanOnly {
		clv = compressLevel
	}
	if len(key) > 0 {
		var block cipher.Block
		block, err = aes.NewCipher(key)
		if err != nil {
			return err
		}

		iv := make([]byte, aes.BlockSize)
		_, err = io.ReadFull(rand.Reader, iv)
		if err != nil {
			return err
		}

		_, err = wt.Write(iv)
		if err != nil {
			return err
		}

		gzipWriter, err = gzip.NewWriterLevel(wt, clv)
		if err != nil {
			return err
		}
		defer gzipWriter.Close()
		// gzipWriter.Header.Comment = "AES encrypted data"
		// gzipWriter.Header.Extra = []byte("AES-256")
		// gzipWriter.Write(nil)
		stream := cipher.NewCTR(block, iv)
		writer := cipher.StreamWriter{
			S: stream,
			W: gzipWriter,
		}

		_, err = io.Copy(writer, rd)
		if err != nil {
			return err
		}
	} else {
		gzipWriter, err = gzip.NewWriterLevel(wt, clv)
		if err != nil {
			return err
		}
		defer gzipWriter.Close()
		if _, err = io.Copy(gzipWriter, rd); err != nil {
			return err
		}
	}

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

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func GenerateRandomAssci(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[RandRangeInterger(0, len(letterRunes)-1)]
	}
	return string(b)
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
