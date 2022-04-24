package sembed

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
)

// httpFile implement http.File.
type httpFile struct {
	fs.FileInfo
	fpath  string
	reader *bytes.Reader
	fs     *HttpSystemFS
	dirIdx int
}

type HttpSystemFS struct {
	fs.SubFS
	subPath string
	embedfs *embed.FS
}

func NewHttpSystemFS(efs *embed.FS, sub ...string) *HttpSystemFS {
	subPath := ""
	if len(sub) != 0 {
		subPath = sub[0]
	}
	return &HttpSystemFS{
		embedfs: efs,
		subPath: subPath,
	}
}

func (fsh *HttpSystemFS) ReadFile(name string) ([]byte, error) {
	return fsh.embedfs.ReadFile(fsh.fullName(name))
}

func (fsh *HttpSystemFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fsh.embedfs.ReadDir(name)
}

func (fsh *HttpSystemFS) fullName(name string) string {
	if len(fsh.subPath) != 0 {
		name = path.Join(fsh.subPath, name)
		// name = fsh.subPath + "/" + name
	}
	return name
}

func (fsh *HttpSystemFS) Open(name string) (hf http.File, err error) {
	var httpf httpFile
	var fileConten []byte
	var f fs.File
	httpf.fs = fsh
	name = fsh.fullName(name)
	if f, err = fsh.embedfs.Open(name); err != nil {
		return
	}
	var fstat fs.FileInfo
	if fstat, err = f.Stat(); err != nil {
		return
	}
	httpf.FileInfo = fstat

	if !fstat.IsDir() {
		if fileConten, err = fsh.embedfs.ReadFile(name); err != nil {
			return
		}
		httpf.reader = bytes.NewReader(fileConten)
	}

	httpf.fpath = name
	return &httpf, nil
}

func (fsh *HttpSystemFS) Setsub(name string) {
	fsh.subPath = name
}

// Sub(dir string) (FS, error)
func (fsh *HttpSystemFS) Sub(dir string) (sub *HttpSystemFS, err error) {
	sub = new(HttpSystemFS)
	*sub = *fsh
	sub.subPath = path.Join(sub.subPath, dir)
	var f fs.File
	if f, err = sub.embedfs.Open(sub.subPath); err != nil {
		return nil, err
	} else {
		var stat fs.FileInfo
		if stat, err = f.Stat(); err == nil {
			if !stat.IsDir() {
				return nil, errors.New(sub.subPath + " is not dir")
			}
		}
	}
	return
}

//clone new sub
func (fsh *HttpSystemFS) NewSub(dir string) *HttpSystemFS {
	sub := *fsh
	sub.subPath = path.Join(sub.subPath, dir)
	return &sub
}

func (f *httpFile) Close() error {
	return nil
}

// Read reads bytes into p, returns the number of read bytes.
func (f *httpFile) Read(p []byte) (n int, err error) {
	n, err = f.reader.Read(p)
	return
}

// Seek seeks to the offset.
func (f *httpFile) Seek(offset int64, whence int) (ret int64, err error) {
	return f.reader.Seek(offset, whence)
}

func (f *httpFile) Stat() (os.FileInfo, error) {
	return f.FileInfo, nil
}

// IsDir returns true if the file location represents a directory.
func (f *httpFile) IsDir() bool {
	return f.FileInfo.IsDir()
}

// Readdir returns an empty slice of files, directory
// listing is disabled.
func (f *httpFile) Readdir(count int) ([]os.FileInfo, error) {
	var fis []os.FileInfo
	if !f.IsDir() {
		return fis, nil
	}
	var entryDirs []fs.DirEntry
	var err error

	entryDirs, err = f.fs.embedfs.ReadDir(f.fpath)
	if err != nil {
		return nil, err
	}
	flen := len(entryDirs)

	// If dirIdx reaches the end and the count is a positive value,
	// an io.EOF error is returned.
	// In other cases, no error will be returned even if, for example,
	// you specified more counts than the number of remaining files.
	start := f.dirIdx
	if start >= flen && count > 0 {
		return fis, io.EOF
	}
	var end int
	if count <= 0 {
		end = flen
	} else {
		end = start + count
	}
	if end > flen {
		end = flen
	}
	var finfo fs.FileInfo
	for i := start; i < end; i++ {
		if finfo, err = entryDirs[i].Info(); err == nil {
			fis = append(fis, finfo)
		}
	}
	f.dirIdx += len(fis)
	return fis, nil
}
