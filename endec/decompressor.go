package endec

import (
	"archive/zip"
	"fmt"
	"io"
)

type aesDecrypter struct {
	*irc
}

type irc struct {
	key       []byte
	iv        []byte
	chunkSize int
	r         io.Reader
	buf       []byte
}

func (rc *irc) Close() (err error) {
	return
}

func (rc *irc) Read(b []byte) (n int, err error) {
	fbuf := make([]byte, cap(b))
	n, err = rc.r.Read(fbuf)
	// if err == io.EOF {
	// }
	if err != nil { // && err != io.EOF {
		return
	}
	if n > cap(b) {
		if s, ok := rc.r.(io.Seeker); ok {
			s.Seek(int64(-n), io.SeekCurrent)
		}
		return 0, fmt.Errorf("bank size greater than %d", rc.chunkSize)
	}

	if d, e := aesEncDecrypt(fbuf, rc.key, rc.iv); e == nil {
		copy(b, d)
	} else {
		return 0, e
	}
	return
}

//type Decompressor func(r io.Reader) io.ReadCloser

func (rcT *irc) Decompress(r io.Reader) (rc io.ReadCloser) {
	rct := new(irc)
	rct.r = r
	rct.buf = make([]byte, rcT.chunkSize)
	return rct
}

func NewAesDecrypter(password []byte, chunkSize int) (raes zip.Decompressor) {
	key, iv := deriveKeyAndIV(password)
	buf := make([]byte, chunkSize)
	rc := &irc{
		key,
		iv,
		chunkSize,
		nil,
		buf,
	}
	return rc.Decompress
}
