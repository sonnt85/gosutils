package endec

import (
	"archive/zip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

var AESZIPCHUNKSIZE = aes.BlockSize * 256

func deriveKeyAndIV(password []byte) ([]byte, []byte) {
	// Chuyển đổi mật khẩu thành một khóa và vector khởi tạo sử dụng PBKDF2
	// (Password-Based Key Derivation Function 2)
	salt := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	key := pbkdf2.Key([]byte(password), salt, 4096, 32, sha256.New)
	iv := pbkdf2.Key([]byte(password), salt, 4096, aes.BlockSize, sha256.New)[:aes.BlockSize]

	return key, iv
}

// Hàm aesEncrypt dùng để mã hóa dữ liệu với thuật toán AES-256.
func aesEncrypt(data []byte, key []byte, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Tạo một slice byte để lưu trữ dữ liệu đã mã hóa
	encrypted := make([]byte, len(data))

	// Sử dụng AES-256 để mã hóa dữ liệu
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, data)

	return encrypted, nil
}

// ZipEncrypt mã hóa file hoặc thư mục và đưa chúng vào một tệp tin ZIP được mã hóa với AES.
// Nếu password là chuỗi rỗng, thì không sử dụng mã hóa.
func ZipEncrypt(source string, destination interface{}, password ...[]byte) (err error) {
	var key, iv []byte
	var hasPwd bool
	if len(password) != 0 && len(password[0]) != 0 {
		hasPwd = true
		key, iv = deriveKeyAndIV(password[0])
	}

	var wt io.Writer
	var isWriter bool
	switch v := destination.(type) {
	case io.Writer:
		isWriter = true
		wt = v
	case string:
		var fw *os.File
		fw, err = os.OpenFile(v, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return
		}
		defer func() {
			fw.Close()
			if err != nil {
				os.Remove(v)
			}
		}()
		wt = fw
	default:
		return errors.New("param is of unknown type")
	}

	writer := zip.NewWriter(wt)
	defer writer.Close()
	var addFileToZip func(string, string) error
	addFileToZip = func(path string, name string) error {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		// if hasPwd {
		// header.Extra = make([]byte, 9)
		// header.Extra[0] = 0x01 // ID của extra field (0x01 = Zip64 extended information)
		// header.Extra[1] = 0x00 // Kích thước của extra field (bytes)
		// header.Extra[2] = 0x07
		// header.Extra[3] = 0x41 // A
		// header.Extra[4] = 0x45 // E
		// header.Extra[5] = 0x53 // S
		// header.Extra[6] = 0x02 // Flag (0 = không nén, 2 = nén bằng ZLIB)
		// header.Extra[7] = 0x02
		// header.Extra[8] = 0x00 // Checksum (nếu flag = 2)
		// }
		// Thêm đường dẫn vào tên của header
		if name != "" {
			header.Name = filepath.Join(name, filepath.Base(path))
		}

		if info.IsDir() {
			if isWriter {
				return fmt.Errorf("io.Writer only for file")
			}

			header.Method = zip.Store
			header.Name += string(os.PathSeparator)
			_, err := writer.CreateHeader(header)
			if err != nil {
				return err
			}

			files, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			for _, file := range files {
				err = addFileToZip(filepath.Join(path, file.Name()), header.Name)
				if err != nil {
					return err
				}
			}
		} else {
			header.Method = zip.Deflate
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			writerEntry, err := writer.CreateHeader(header)
			if err != nil {
				return err
			}

			if !hasPwd {
				_, err = io.Copy(writerEntry, file)
				if err != nil {
					return err
				}
			} else {
				blockSize := aes.BlockSize
				chunkSize := AESZIPCHUNKSIZE // Đọc và mã hóa 1024 byte một lần
				buf := make([]byte, chunkSize)
				for {
					n, err := file.Read(buf)
					if err != nil && err != io.EOF {
						return err
					}

					if n == 0 {
						break
					}

					encrypted, err := aesEncrypt(Pkcs7Padding(buf[:n], blockSize), key, iv)
					if err != nil {
						return err
					}
					_, err = writerEntry.Write(encrypted)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	err = addFileToZip(source, "")
	return
}

type lenSize interface {
	Size() int
}

type lenLen interface {
	Len() int
}

type lenLength interface {
	Length() int
}

// ZipDecrypt giải mã tệp tin ZIP và giải nén tất cả các file và thư mục.
// Nếu password là chuỗi rỗng, thì không sử dụng mã hóa.
//src path to src, or io.ReaderAt

func ZipDecrypt(src interface{}, dstPath string, removeSrc bool, password ...[]byte) (err error) {
	// Tạo khóa và vector khởi tạo từ mật khẩu (nếu password không rỗng)
	var key, iv []byte
	var hasPwd bool
	if len(password) != 0 && len(password[0]) != 0 {
		hasPwd = true
		key, iv = deriveKeyAndIV(password[0])
	}
	var rd io.ReaderAt
	var fr *os.File
	var zipSize int64
	// var filemod os.FileMode
	// filemod = 0755
	switch v := src.(type) {
	case io.ReaderAt:
		rd = v
		if sizer, ok := src.(lenSize); ok {
			zipSize = int64(sizer.Size())
		} else if sizer, ok := src.(lenLen); ok {
			zipSize = int64(sizer.Len())
		} else if sizer, ok := src.(lenLength); ok {
			zipSize = int64(sizer.Length())
		} else {
			return fmt.Errorf("io.ReaderAt has no size method size")
		}

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
		// filemod = inputInfo.Mode().Perm()
		zipSize = inputInfo.Size()
		rd = fr
	default:
		return errors.New("param is of unknown type")
	}
	// Tạo một đối tượng zip.Reader để đọc từ tệp tin ZIP
	zipReader, err := zip.NewReader(rd, zipSize)
	if err != nil {
		return err
	}

	// Đọc tất cả các file và thư mục trong tệp tin ZIP
	for _, file := range zipReader.File {
		// Tạo đường dẫn đầy đủ cho file hoặc thư mục
		path := filepath.Join(dstPath, file.Name)

		// Nếu đối tượng là một thư mục, tạo thư mục và tiếp tục giải nén các file và thư mục con
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, os.ModePerm)
			continue
		}

		// Nếu đối tượng là một file, giải mã dữ liệu và ghi vào file
		reader, err := file.Open()
		if err != nil {
			return err
		}
		defer reader.Close()

		// Nếu password rỗng, không sử dụng mã hóa
		if !hasPwd {
			writer, err := os.Create(path)
			if err != nil {
				return err
			}
			defer writer.Close()

			_, err = io.Copy(writer, reader)
			if err != nil {
				return err
			}
		} else {
			// Nếu password không rỗng, giải mã dữ liệu với AES-256 và ghi vào file
			writer, err := os.Create(path)
			if err != nil {
				return err
			}
			defer writer.Close()

			// Đọc dữ liệu từ tệp tin ZIP và giải mã với AES-256
			block, err := aes.NewCipher(key)
			if err != nil {
				return err
			}

			mode := cipher.NewCBCDecrypter(block, iv)

			blockSize := aes.BlockSize
			chunkSize := AESZIPCHUNKSIZE // Đọc và mã hóa 1024 byte một lần
			buf := make([]byte, chunkSize)
			decrypted := make([]byte, chunkSize)
			for {
				n, err := reader.Read(buf)
				if err != nil && err != io.EOF {
					return err
				}

				if n == 0 {
					break
				}

				// Giải mã dữ liệu với AES-256
				if n%blockSize != 0 {
					return fmt.Errorf("the number of bytes read is not a multiple of %d", blockSize)
				}
				mode.CryptBlocks(decrypted[:n], buf[:n])

				decrypted = Pkcs7Unpadding(decrypted[:n])
				// Điền các byte bị thiếu vào block cuối cùng (nếu có)
				// if n < aes.BlockSize {
				// copy(decrypted[n:aes.BlockSize], make([]byte, aes.BlockSize-n))
				// }

				_, err = writer.Write(decrypted)
				// copy(decrypted[:n], make([]byte, n))

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
