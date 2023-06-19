// Copyright 2020 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package endec

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// AESGCMEncrypt encrypts plaintext with the given key using AES in GCM mode.
func AESGCMEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

func AESGCMEncryptToBase64(key, plaintext []byte) (str string, err error) {
	var b []byte
	b, err = AESGCMEncrypt(key, plaintext)
	if err != nil {
		return
	}
	str = base64.RawStdEncoding.EncodeToString(b)
	return
}

// AESGCMDecrypt decrypts ciphertext with the given key using AES in GCM mode.
func AESGCMDecrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	size := gcm.NonceSize()
	if len(ciphertext)-size <= 0 {
		return nil, errors.New("ciphertext is empty")
	}

	nonce := ciphertext[:size]
	ciphertext = ciphertext[size:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

func AESGCMDecryptFromBase64(key []byte, str string) (data []byte, err error) {
	data, err = base64.RawStdEncoding.DecodeString(str)
	if err != nil {
		return
	}
	return AESGCMDecrypt(key, data)
}

func AESCBCEncrypt(key, plaintext []byte, ivs ...[]byte) (result []byte, err error) {
	// Generate a new AES cipher block using the given key
	var block cipher.Block
	block, err = aes.NewCipher(key)
	if err != nil {
		return
	}

	// Generate a random initialization vector (IV)
	iv := make([]byte, aes.BlockSize)
	if len(ivs) != 0 {
		iv = ivs[0]
	} else {
		if _, err = io.ReadFull(rand.Reader, iv); err != nil {
			return
		}
	}

	// Pad the plaintext to a multiple of the block size
	plaintext = Pkcs7Padding(plaintext, aes.BlockSize)

	// Create a new CBC mode block cipher using the AES cipher block and IV
	mode := cipher.NewCBCEncrypter(block, iv)

	// Encrypt the padded plaintext
	ciphertext := make([]byte, len(plaintext))
	mode.CryptBlocks(ciphertext, plaintext)

	// Append the IV to the ciphertext and return the result as base64
	if len(ivs) == 0 {
		result = append(ciphertext, iv...)
	} else {
		result = ciphertext
	}
	return
}

func AESCBCDecrypt(key, ciphertext []byte, ivs ...[]byte) (plaintext []byte, err error) {
	// Decode the ciphertext from base64
	// data, err := base64.StdEncoding.DecodeString(ciphertext)
	// if err != nil {
	// 	return nil, err
	// }

	// Separate the ciphertext and IV
	if len(ciphertext) < aes.BlockSize {
		err = errors.New("ciphertext length is smaller than AES Blocksize")
		return
	}
	ciphertextBytes := ciphertext
	iv := make([]byte, aes.BlockSize)
	if len(ivs) != 0 {
		if len(ivs[0]) != aes.BlockSize {
			err = errors.New("iv length is smaller than AES Blocksize")
			return
		}
		iv = ivs[0]
	} else {
		ciphertextBytes = ciphertext[:len(ciphertext)-aes.BlockSize]
		iv = ciphertext[len(ciphertext)-aes.BlockSize:]
	}
	// Generate a new AES cipher block using the given key
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create a new CBC mode block cipher using the AES cipher block and IV
	mode := cipher.NewCBCDecrypter(block, iv)

	// Decrypt the ciphertext
	plaintext = make([]byte, len(ciphertextBytes))
	mode.CryptBlocks(plaintext, ciphertextBytes)

	// Remove any padding from the plaintext
	plaintext = Pkcs7Unpadding(plaintext)

	return plaintext, nil
}

func AESCBCEncryptToBase64(key, plaintext []byte, ivs ...[]byte) (end string, err error) {
	var enb []byte
	enb, err = AESCBCEncrypt(key, plaintext, ivs...)
	if err != nil {
		return
	}
	return base64.StdEncoding.EncodeToString(enb), nil
}

func AESCBCDecryptFromBase64(key []byte, ciphertext string, ivs ...[]byte) ([]byte, error) {
	// Decode the ciphertext from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}
	return AESCBCDecrypt(key, data, ivs...)
}

func Pkcs7Padding(input []byte, blockSize int) []byte {
	paddingSize := blockSize - len(input)%blockSize
	// if paddingSize == blockSize {
	// 	return input
	// }
	padding := make([]byte, paddingSize)
	for i := range padding {
		padding[i] = byte(paddingSize)
	}
	return append(input, padding...)
}

func Pkcs7Unpadding(input []byte, blockSize ...int) []byte {
	if len(input) == 0 {
		return []byte{}
	}
	padding := input[len(input)-1]
	if (len(input) - int(padding)) >= 0 {
		for i := len(input) - int(padding); i < len(input); i++ {
			if input[i] != padding {
				// return []byte{}
				return input
			}
		}
		return input[:len(input)-int(padding)]
	} else {
		return input
	}
}
