package endec

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"math/big"
	"path/filepath"
	"strings"

	//	"encoding/base64"
	"encoding/base64"

	//	"fmt"
	"io"
	"os"
)

// EncrypBytes encrypts a byte slice (data) using the provided passphrase and returns
// the encrypted ciphertext using AES in GCM (Galois/Counter Mode).
//
// Parameters:
//   - data []byte: The data to be encrypted.
//   - passphrase []byte: The passphrase used as the encryption key.
//
// Returns:
//   - retbytes []byte: The encrypted ciphertext.
//   - err error: An error, if any, encountered during encryption.
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
	//
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// EncrypBytesToString encrypts a byte slice (data) using the provided passphrase and returns
// the encrypted ciphertext as a base64-encoded string, using AES in GCM (Galois/Counter Mode).
//
// Parameters:
//   - data []byte: The data to be encrypted.
//   - passphrase []byte: The passphrase used as the encryption key.
//
// Returns:
//   - retbstring string: The encrypted ciphertext as a base64-encoded string.
//   - err error: An error, if any, encountered during encryption.
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

// EncryptBytesToFile encrypts a byte slice (data) using the provided passphrase and writes
// the encrypted ciphertext to a file specified by the 'filename' parameter. The 'filename'
// can be either a string (file path) or an io.Writer. If 'filename' is an io.Writer, the
// function writes the ciphertext directly to it. If 'filename' is a string, the function
// creates a new file with that name and writes the ciphertext to it.
// using AES in GCM (Galois/Counter Mode).
//
// Parameters:
//   - filename interface{}: Either a string (file path) or an io.Writer.
//   - data []byte: The data to be encrypted.
//   - passphrase []byte: The passphrase used as the encryption key.
//
// Returns:
//   - err error: An error, if any, encountered during encryption or file I/O.
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

// EncryptFileWithRandomPassword reads the content of a file specified by 'pathfile', generates
// a random passphrase, and encrypts the file's content with that passphrase. The encrypted
// content is then written to a new file with a name formed by appending the passphrase to the
// original file's name.
// using AES in GCM (Galois/Counter Mode).
// Parameters:
//   - pathfile string: The path to the file to be encrypted.
//   - dstpathprefixs ...string: Optional destination path prefix for the encrypted file.
//
// Returns:
//   - err error: An error, if any, encountered during file I/O or encryption.
func EncryptFileWithRandomPassword(pathfile string, dstpathprefixs ...string) error {
	data, err := os.ReadFile(pathfile)
	if err != nil {
		return err
	}
	passphrase := GenerateRandomAssci(16, []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"))
	pathprefix := pathfile
	if len(dstpathprefixs) != 0 {
		pathprefix = dstpathprefixs[0]
	}
	pathprefix = fmt.Sprintf("%ss-nt%s", pathprefix, passphrase)
	err = EncryptBytesToFile(pathprefix, data, []byte(passphrase))
	return err
}

// DecryptFileWithPasswordInFileToBytes attempts to decrypt the content of a file specified by
// 'pathfile' using a passphrase obtained from the filename itself. If the filename follows
// the format "<original_filename>s-nt<passphrase>", it extracts the passphrase and attempts to
// use it for decryption.
// using AES in GCM (Galois/Counter Mode).
// Parameters:
//   - pathfile string: The path to the file to be decrypted.
//
// Returns:
//   - data []byte: The decrypted data, if successful.
//   - err error: An error, if any, encountered during file I/O or decryption.
func DecryptFileWithPasswordInFileToBytes(pathfile string) (data []byte, err error) {
	bname := filepath.Base(pathfile)
	if _, passwd, found := strings.Cut(bname, "s-nt"); found {
		data, err = DecryptFileToBytes(pathfile, []byte(passwd))
	}
	return
}

// DecryptBytes decrypts a byte slice (data) using the provided passphrase and returns
// the decrypted plaintext using AES in GCM (Galois/Counter Mode).
//
// Parameters:
//   - data []byte: The data to be decrypted.
//   - passphrase []byte: The passphrase used as the decryption key.
//
// Returns:
//   - retbytes []byte: The decrypted plaintext.
//   - err error: An error, if any, encountered during decryption.
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

func Base64EncodeNoPadding(input string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(input))
	return strings.TrimRight(encoded, "=")
}

func Base64DecodeNoPadding(encoded string) ([]byte, error) {
	padding := len(encoded) % 4
	if padding > 0 {
		encoded += strings.Repeat("=", 4-padding)
	}
	return base64.StdEncoding.DecodeString(encoded)
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
		data, err = os.ReadFile(v)
	default:
		return data, errors.New("param is of unknown type")
	}

	if err != nil {
		return data, err
	}
	return DecryptBytes(data, passphrase)
}

// DecryptFileToFile decrypts the content of a file specified by 'inputFile' using the provided
// passphrase and writes the decrypted content to another file specified by 'outputFile'. The
// 'inputFile' is read, decrypted using AES in GCM (Galois/Counter Mode), and the resulting
// plaintext is written to 'outputFile'. The file permissions of 'outputFile' are set to match
// those of 'inputFile'.
//
// Parameters:
//   - inputFile string: The path to the input file to be decrypted.
//   - outputFile string: The path to the output file where decrypted data will be written.
//   - passphrase []byte: The passphrase used as the decryption key.
//
// Returns:
//   - err error: An error, if any, encountered during file I/O, decryption, or writing to the output file.
func DecryptFileToFile(inputFile, outputFile string, passphrase []byte) (err error) {
	data, err := os.ReadFile(inputFile)
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
	err = os.WriteFile(outputFile, data, inputInfo.Mode().Perm())
	return
}

// EncryptFileToFile encrypts the content of a file specified by 'inputFile' using the provided
// passphrase and writes the encrypted content to another file specified by 'outputFile' (string
// path) or an io.Writer. The 'inputFile' is read, encrypted using AES in GCM (Galois/Counter Mode),
// and the resulting ciphertext is written to 'outputFile'. The file permissions of 'outputFile' (if
// it is a string path) are set to match those of 'inputFile'.
//
// Parameters:
//   - inputFile string: The path to the input file to be encrypted.
//   - outputFile interface{}: Either a string (file path) or an io.Writer.
//   - passphrase []byte: The passphrase used as the encryption key.
//
// Returns:
//   - err error: An error, if any, encountered during file I/O, encryption, or writing to the output.
func EncryptFileToFile(inputFile string, outputFile interface{}, passphrase []byte) (err error) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}

	var inputInfo os.FileInfo
	inputInfo, err = os.Stat(inputFile)
	if err != nil {
		return
	}

	encryptedData, err := EncrypBytes(data, passphrase)
	if err != nil {
		return err
	}

	switch v := outputFile.(type) {
	case string:
		err = os.WriteFile(v, encryptedData, inputInfo.Mode().Perm())
	case io.Writer:
		_, err = v.Write(encryptedData)
	default:
		return errors.New("outputFile is of an unsupported type")
	}
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
		data, err := io.ReadAll(r)
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
	clv := gzip.BestCompression
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

func GenerateRandomAssci(n int, runs ...[]rune) string {
	b := make([]rune, n)
	chars := letterRunes
	if len(runs) != 0 {
		chars = runs[0]
	}
	for i := range b {
		b[i] = chars[RandRangeInterger(0, len(chars)-1)]
	}
	return string(b)
}

func RandUint64() uint64 {
	val, err := rand.Int(rand.Reader, big.NewInt(int64(math.MaxInt64)))
	if err != nil {
		return 0
	}
	return val.Uint64()
}

func RandInt64() int64 {
	return int64(RandUint64())
}

func RandUint32() uint32 {
	val, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return 0
	}
	return uint32(val.Uint64())
}

func RandInt32() int32 {
	return int32(RandUint32())
}

func RandInt() int {
	return int(RandUint64())
}

func RandUint() uint {
	return uint(RandUint64())
}

func RandRangeInterger(from, to int) (ret int) {
	delta := to - from
	if delta == 0 {
		return from
	}
	randi := RandInt32()
	if randi > 0 {
		return from + int(randi)%delta
	} else {
		return from - int(randi)%delta
	}
}

func RandRangeInt64(from, to int64) int64 {
	delta := to - from
	if delta == 0 {
		return from
	}
	randi64 := RandUint64()
	if randi64 > 0 {
		return from + int64(randi64)%delta
	} else {
		return from - int64(randi64)%delta
	}
}
