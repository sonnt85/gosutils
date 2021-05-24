package endec

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"

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
	return base64.StdEncoding.EncodeToString(base64Text, []byte(output))
}

func StringSimpleDecrypt(input, key string) (output string, err error) {
	data, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(data); i++ {
		output += string(data[i] ^ key[i%len(key)])
	}
	return output, nil
}
