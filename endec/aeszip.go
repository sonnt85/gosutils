package endec

import (
	"archive/zip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

var AESZIPCHUNKSIZE = aes.BlockSize * 256

const ZIPTYPEAES = 0x66

func deriveKeyAndIV(password []byte) ([]byte, []byte) {
	// Convert passwords into a key and vector to use PBKDF2
	// (Password-Based Key Derivation Function 2)
	salt := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	key := pbkdf2.Key([]byte(password), salt, 4096, 32, sha256.New)
	iv := pbkdf2.Key([]byte(password), salt, 4096, aes.BlockSize, sha256.New)[:aes.BlockSize]

	return key, iv
}

func AesEncDecryptViaPassord(data []byte, password []byte, decryp ...bool) ([]byte, error) {
	key, iv := deriveKeyAndIV(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	// Create a slice byte to store encrypted data
	encrypted := make([]byte, len(data))

	// Use AES-256 to encrypt data
	var mode cipher.BlockMode

	if len(decryp) != 0 && decryp[0] {
		mode = cipher.NewCBCDecrypter(block, iv)
	} else {
		mode = cipher.NewCBCEncrypter(block, iv)
	}
	mode.CryptBlocks(encrypted, data)
	return encrypted, nil
}

// The AES Encrypt function is used to encrypt data with AES-256 algorithm.
func aesEncDecrypt(data []byte, key []byte, iv []byte, decryp ...bool) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create a slice byte to store encrypted data
	encrypted := make([]byte, len(data))

	// Use AES-256 to encrypt data
	var mode cipher.BlockMode

	if len(decryp) != 0 && decryp[0] {
		mode = cipher.NewCBCDecrypter(block, iv)
	} else {
		mode = cipher.NewCBCEncrypter(block, iv)
	}
	mode.CryptBlocks(encrypted, data)
	return encrypted, nil
}

type zipWriter struct {
	*zip.Writer
	key, iv   []byte
	isWriter  bool
	blockSize uint8
	chunkSize int
	password  []byte
}

func (zw *zipWriter) addFileToZip(sourceToAdd string, name string) error {
	hasPwd := false
	if len(zw.password) != 0 {
		hasPwd = true
	}
	info, err := os.Stat(sourceToAdd)
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	if header.Extra == nil {
		header.Extra = make([]byte, 0)
	}

	// permissions := info.Mode()
	if owner, group, err := getFileOwnership(info); err == nil {
		size := byte(12)
		hd := make([]byte, size+2)                                    // Start array with size 20 byte to store field and data ID
		hd[0] = 0x07                                                  // Set the field ID as 0x07 (linux permistion)
		hd[1] = size                                                  // Set the field size of 10 (8 bytes for data + 2 bytes for ID and size)
		binary.LittleEndian.PutUint32(hd[2:6], uint32(owner))         // Write the owner in byte 2-5
		binary.LittleEndian.PutUint32(hd[6:10], uint32(group))        // Write Group in byte 6-9
		binary.LittleEndian.PutUint32(hd[10:14], uint32(info.Mode())) // Write linux permistion
		header.Extra = append(header.Extra, hd...)
	}
	if hasPwd {
		// size := byte(7)
		// hd := make([]byte, size+2)
		// binary.LittleEndian.PutUint16(hd[0:2], uint16(0x9901)) //Extra field header ID (0x9901)
		// binary.LittleEndian.PutUint16(hd[2:4], uint16(size))   //Data size
		// binary.LittleEndian.PutUint16(hd[4:6], uint16(0x0123)) //Integer version number specific to the zip vendor
		// binary.LittleEndian.PutUint16(hd[4:6], uint16(0x2345)) //Integer version number specific to the zip vendor
		// hd[6] = 8                                              //Integer mode value indicating AES encryption strength
		// binary.LittleEndian.PutUint16(hd[4:6], uint16(0x2345)) //The actual compression method used to compress the file
		// header.Extra = append(header.Extra, hd...)
		///
		size := byte(len(zw.password))
		hd := make([]byte, size+2)
		hd[0] = ZIPTYPEAES // ID of the secondary school (0x01 = Zip64 extension information)
		hd[1] = size       //The size of Extra Field (Bytes)
		copy(hd[2:], zw.password)
		header.Extra = append(header.Extra, hd...)
	}
	lastChunk := int64(1)
	npading := uint8(0)
	if !info.IsDir() {
		header.Extra = append(header.Extra, []byte{0xEE, 1, 0}...)
		t := uint8(info.Size() % int64(zw.blockSize))
		if t != 0 {
			npading = zw.blockSize - t
			header.Extra[len(header.Extra)-1] = byte(npading)
		}
		lastChunk = info.Size() / int64(zw.chunkSize)
		if info.Size()%int64(zw.chunkSize) != 0 {
			lastChunk++
		}
		// lastChunk = (info.Size()+int64(zw.chunkSize))/int64(zw.chunkSize) - 1
	}
	if name != "" {
		header.Name = filepath.Join(name, filepath.Base(sourceToAdd))
	}

	// 	numpading := 0
	// 		if numpading > zw.blockSize {
	// 			return fmt.Errorf("[%d] pading > blocksize %d > %d ", k, numpading, blockSize)
	// 		}
	// header.Extra[padingIndex] = byte(s.Size()%int64(zw.blockSize))

	if info.IsDir() {
		if zw.isWriter {
			return fmt.Errorf("io.Writer only for file")
		}

		header.Method = zip.Store
		header.Name += string(os.PathSeparator)
		_, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		files, err := os.ReadDir(sourceToAdd)
		if err != nil {
			return err
		}
		for _, file := range files {
			err = zw.addFileToZip(filepath.Join(sourceToAdd, file.Name()), header.Name)
			if err != nil {
				return err
			}
		}
	} else {
		header.Method = zip.Deflate
		file, err := os.Open(sourceToAdd)
		if err != nil {
			return err
		}
		defer file.Close()

		writerEntry, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		if !hasPwd {
			_, err = io.Copy(writerEntry, file)
			if err != nil {
				return err
			}
		} else {
			k := int64(0)
			buf := make([]byte, zw.chunkSize)
			for {
				k++
				n, err := file.Read(buf)

				if err != nil && err != io.EOF {
					return err
				}
				if n == 0 {
					break
				}

				// encrypted, err := aesEncrypt(Pkcs7Padding(buf[:n], blockSize), key, iv)
				// Pkcs7Padding(buf[:n]
				npad := uint8(0)
				if lastChunk == k {
					npad = npading
				} else {
					if n != zw.chunkSize {
						return fmt.Errorf("number of bytes reading %d is not equal to chunksize %d", n, zw.chunkSize)
					}
				}

				encrypted, err := aesEncDecrypt(buf[:n+int(npad)], zw.key, zw.iv)
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

// Zip Encrypt encodes files or folders and puts them in a zip file encrypted with AES.
// If the password is an empty chain, do not use encryption.
func ZipEncrypt(source string, destination interface{}, password ...[]byte) (err error) {
	var key, iv []byte
	pwd := []byte{}
	if len(password) != 0 && len(password[0]) != 0 {
		pwd = password[0]
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
	zw := new(zipWriter)
	zw.isWriter = isWriter
	zw.key = key
	zw.iv = iv
	zw.Writer = writer
	zw.blockSize = aes.BlockSize
	zw.chunkSize = AESZIPCHUNKSIZE // Read and encrypt 4096 bytes once
	zw.password = pwd
	err = zw.addFileToZip(source, "")
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

func ZipDecrypt(src interface{}, dstPath string, removeSrc bool, passwords ...interface{}) (err error) {
	// Create locks and vector initialized from the password (if the password is not empty)	var key, iv []byte
	var hasPwd bool
	var key, iv []byte
	var password []byte
	ignoreIfExits := false
	for _, i := range passwords {
		switch v := i.(type) {
		case []byte:
			if len(v) != 0 {
				password = v
				hasPwd = true
				key, iv = deriveKeyAndIV(v)
			}
		case bool:
			ignoreIfExits = true
		}
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
	// Create an object of zip Reader to read from Zip file
	zipReader, err := zip.NewReader(rd, zipSize)
	// zipReader.RegisterDecompressor(method uint16, dcomp zip.Decompressor)
	if err != nil {
		return err
	}

	// Read all files and folders in the Zip file
	firstCheck := true
	for _, file := range zipReader.File {
		// Create a complete path for files or folders
		err := func() (err error) {
			path := filepath.Join(dstPath, file.Name)
			extra := file.FileHeader.Extra
			pading := uint8(0)
			var extraFun func()
			if len(extra) != 0 {
				i := 0
				for {
					if i >= len(extra) {
						break
					}
					length := int(extra[i+1])
					if length+i > len(extra) {
						break
					}
					// Get the ID and data of the Extra segment
					id := extra[i]
					data := extra[i+2 : i+length+2]
					//Extra segment processing based on ID
					switch id {
					case ZIPTYPEAES: // Zip64 extended information
						//Data processing Zip64
						if firstCheck {
							if len(password) == 0 {
								hasPwd = true
								key, iv = deriveKeyAndIV(data)
								firstCheck = false
							}
						}
					case 0xEE: // pading
						if length == 1 {
							pading = data[0]
						}
					case 0x07: // Unix permissions
						if length == 12 {
							extraFun = func() {
								os.Chown(path, int(binary.LittleEndian.Uint32(data[0:4])), int(binary.LittleEndian.Uint32(data[4:8])))
								os.Chmod(path, fs.FileMode(binary.LittleEndian.Uint32(data[8:12])))
							}
						}
					default:
						// skip the unknown ID
					}
					// move to the next Extra
					i += length + 2
				}
			}
			// If the object is a file, decode data and write it in the file
			reader, err := file.Open()
			if err != nil {
				return err
			}
			defer reader.Close()
			// If the object is a folder, create folders and continue to extract files and sub -folders
			if file.FileInfo().IsDir() {
				os.MkdirAll(path, file.Mode())
				return nil
			}
			// If the password is empty, do not use encryption
			if ignoreIfExits {
				if s, err := os.Stat(path); err == nil && !s.IsDir() {
					return nil
				}
			}
			writer, err := os.Create(path)
			if err != nil {
				return err
			}
			if extraFun != nil {
				extraFun()
			}
			defer func() {
				writer.Close()
				os.Chmod(path, file.Mode())
			}()
			if !hasPwd {
				_, err = io.Copy(writer, reader)
				if err != nil {
					return err
				}
			} else {
				blockSize := aes.BlockSize
				chunkSize := AESZIPCHUNKSIZE // Read and encrypt AESZIPCHUNKSIZE + 1 byte pading bytes once
				// If the password is not empty, decipher data with AES-256 and write in the file
				// Read data from Zip and decoded files with AES-256
				// block, err := aes.NewCipher(key)
				// mode := cipher.NewCBCDecrypter(block, iv)
				// decrypted := make([]byte, chunkSize)

				if err != nil {
					return err
				}
				// lastChunk := (file.FileInfo().Size()+int64(chunkSize))/int64(chunkSize) - 1
				lastChunk := file.FileInfo().Size() / int64(chunkSize)
				if file.FileInfo().Size()%int64(chunkSize) != 0 {
					lastChunk++
				}
				k := int64(0)
				buf := make([]byte, chunkSize)
				for {
					k++
					n, err := reader.Read(buf)
					if err != nil && err != io.EOF {
						return err
					}

					if n == 0 {
						break
					}

					// Data decoding with AES-256
					if n%blockSize != 0 {
						return fmt.Errorf("the number of bytes read [time %d] is %d not a multiple of %d", k, n, blockSize)
					}
					var decrypted []byte
					decrypted, err = aesEncDecrypt(buf[:n], key, iv, true)
					if err != nil {
						return err
					}
					numpading := uint8(0)
					if k == lastChunk {
						numpading = pading
					}
					_, err = writer.Write(decrypted[:n-int(numpading)])
					if err != nil {
						return err
					}
				}
			}
			return err
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

type File struct {
	getpwd func() []byte
	*zip.File
}

func (f *File) Open() (reader io.ReadCloser, err error) {
	var key, iv []byte
	var hasPwd bool
	password := []byte{}
	if f.getpwd != nil {
		password = f.getpwd()
	}
	if len(password) != 0 {
		hasPwd = true
		key, iv = deriveKeyAndIV(password)
	}
	// Nếu đối tượng là một file, giải mã dữ liệu và ghi vào file
	reader, err = f.File.Open()
	if err != nil {
		return
	}
	// defer reader.Close()

	var writer *os.File
	if hasPwd {
		// Đọc dữ liệu từ tệp tin ZIP và giải mã với AES-256
		var block cipher.Block
		block, err = aes.NewCipher(key)
		if err != nil {
			return
		}

		mode := cipher.NewCBCDecrypter(block, iv)

		blockSize := aes.BlockSize
		chunkSize := AESZIPCHUNKSIZE // Đọc và mã hóa 1024 byte một lần
		buf := make([]byte, chunkSize)
		decrypted := make([]byte, chunkSize)
		for {
			var n int
			n, err = reader.Read(buf)
			if err != nil && err != io.EOF {
				return
			}

			if n == 0 {
				break
			}

			// Giải mã dữ liệu với AES-256
			if n%blockSize != 0 {
				err = fmt.Errorf("the number of bytes read is not a multiple of %d", blockSize)
				return
			}
			mode.CryptBlocks(decrypted[:n], buf[:n])

			decrypted = Pkcs7Unpadding(decrypted[:n])
			// Fill the missing bytes into the last block (if any)
			// if n < aes.BlockSize {
			// copy(decrypted[n:aes.BlockSize], make([]byte, aes.BlockSize-n))
			// }

			_, err = writer.Write(decrypted)
			// copy(decrypted[:n], make([]byte, n))

			if err != nil {
				return
			}
		}
	}
	return
}

type ZipReader struct {
	password []byte
	close    func() error
	*zip.Reader
	File []*File
}

func (z *ZipReader) GetPassword() []byte {
	return z.password
}
func (z *ZipReader) Close() error {
	// z.Reader.Close()
	if z.close != nil {
		z.close()
	}
	return nil
}

func NewZipReader(r *zip.Reader, password ...[]byte) (zr *ZipReader) {
	if r == nil {
		return nil
	}
	zr = &ZipReader{
		Reader: r,
	}
	if len(password) != 0 && len(password[0]) != 0 {
		zr.RegisterDecompressor(ZIPTYPEAES, NewAesDecrypter(password[0], AESZIPCHUNKSIZE))
	}
	return
}

func ZipOpen(src interface{}, password ...[]byte) (zipReader *ZipReader, err error) {
	// Tạo khóa và vector khởi tạo từ mật khẩu (nếu password không rỗng)
	// archiver.
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
			err = fmt.Errorf("io.ReaderAt has no size method size")
			return
		}

	case string:
		fr, err = os.Open(v)
		if err != nil {
			return
		}
		// defer func() {
		// 	fr.Close()
		// }()
		var inputInfo os.FileInfo
		inputInfo, err = os.Stat(v)
		if err != nil {
			return
		}
		// filemod = inputInfo.Mode().Perm()
		zipSize = inputInfo.Size()
		rd = fr
	default:
		err = errors.New("param is of unknown type")
		return
	}
	// Tạo một đối tượng zip.Reader để đọc từ tệp tin ZIP
	var zr *zip.Reader
	zr, err = zip.NewReader(rd, zipSize)
	if err != nil {
		return
	}

	zipReader = NewZipReader(zr, password...)
	if fr != nil {
		zipReader.close = fr.Close
	}
	var dcomp zip.Decompressor
	zipReader.RegisterDecompressor(zip.Deflate, dcomp)
	// Đọc tất cả các file và thư mục trong tệp tin ZIP

	return
}
