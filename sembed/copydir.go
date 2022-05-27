package sembed

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gofilepath"
	"github.com/sonnt85/gosystem"
)

// File implement
type FileReadDir struct {
	// fs ReadFS
	fs.ReadDirFS
	// fs.ReadFileFS
	// fs.StatFS

	fullpath string
	fs.FileInfo
	dirIdx int
}

type WriteFun func(name string, data []byte, perm os.FileMode) error

type MkDirFun func(srcPath string, srcFileInfo fs.FileMode) error

type CopyRecursive struct {
	// IsRecursive bool
	http.FileSystem
	IsVerbose bool

	// fs.ReadFileFS
	// fs.StatFS
	Open     func(name string) (http.File, error)
	Stat     func(root string) (finfo fs.FileInfo, err error)
	ReadFile func(name string) ([]byte, error)
	Writer   WriteFun
	Mkdir    MkDirFun

	IgnErr           bool
	srcPath          string
	dstPath          string
	srcPathSeparator string
	dstPathSeparator string
}

func (cr *CopyRecursive) mkdir(srcPath string, srcFileInfo fs.FileMode) error {
	return cr.Mkdir(srcPath, 0755)
}

// Readdir returns an empty slice of files, directory
// listing is disabled.
func (f *FileReadDir) Readdir(count int) ([]os.FileInfo, error) {
	var fis []os.FileInfo
	if !f.IsDir() {
		return fis, nil
	}
	var entryDirs []fs.DirEntry
	var err error

	entryDirs, err = f.ReadDir(f.fullpath)
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

func (cr *CopyRecursive) _Open(name string) (hf *FileReadDir, err error) {
	var httpf FileReadDir
	var fstat fs.FileInfo
	if fstat, err = cr.Stat(name); err != nil {
		return
	}
	// httpf.ReadDirFS
	httpf.fullpath = name
	httpf.FileInfo = fstat
	return &httpf, nil
}

func (cr *CopyRecursive) processDir(srcFilePath string, srcFileInfo os.FileInfo) (err error) {
	var relpath string
	relpath, err = gofilepath.Rel(cr.srcPath, srcFilePath)
	if err != nil {
		return
	}
	newdir := gofilepath.JointSmart(cr.dstPathSeparator, cr.dstPath, relpath)
	err = cr.mkdir(newdir, srcFileInfo.Mode())
	if err != nil {
		return err
	}
	dir, err := cr.Open(srcFilePath)
	if err != nil {
		return err
	}
	fis, err := dir.Readdir(0)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if fi.IsDir() {
			err = cr.processDir(gofilepath.JointSmart(cr.dstPathSeparator, srcFilePath, fi.Name()), fi)
			if err != nil {
				if cr.IgnErr {
					log.Warnf("processDir error [local ignore]: %v", err)
				} else {
					return err
				}
			}
		} else {
			err = cr.copyFile(gofilepath.JointSmart(cr.dstPathSeparator, srcFilePath, fi.Name()), fi)
			if err != nil {
				if cr.IgnErr {
					log.Warnf("sendFile error [local ignore]: %v", err)
				} else {
					return err
				}
			}
		}
	}
	return err
}

func (cr *CopyRecursive) copyFile(srcPath string, srcFileInfo os.FileInfo) (err error) {
	var relpath string
	relpath, err = gofilepath.Rel(cr.srcPath, srcPath)
	if err != nil {
		return
	}
	var filecontent []byte
	filecontent, err = cr.ReadFile(srcPath)
	if err != nil {
		return
	}
	cr.Writer(gofilepath.JointSmart(cr.dstPathSeparator, cr.dstPath, relpath), filecontent, srcFileInfo.Mode())
	return err
}

func (cr *CopyRecursive) MkdirAll(dstPath string, dstFileInfo fs.FileMode) (err error) {
	eles := strings.Split(dstPath, cr.dstPathSeparator)
	paths := ""
	for i := 0; i < len(eles); i++ {
		if len(paths) == 0 {
			paths = eles[i]
		} else if eles[i] != "" {
			paths = strings.Join([]string{paths, eles[i]}, cr.dstPathSeparator)
		} else {
			continue
		}
		err = cr.mkdir(paths, dstFileInfo)
		if err != nil {
			break
		}
	}
	return
}

func (cr *CopyRecursive) Copy(dstName, srcName string) (err error) {
	if dstName == "" {
		return errors.New("dstName cannot empty")
	}

	if err != nil {
		return err
	}
	var srcFileInfo fs.FileInfo
	srcFileInfo, err = cr.Stat(srcName)
	if err != nil {
		return err
	}
	cr.dstPathSeparator = gofilepath.GetPathSeparator(dstName)
	if cr.dstPathSeparator == "" {
		cr.dstPathSeparator = string(os.PathSeparator)
	}

	cr.srcPathSeparator = gofilepath.GetPathSeparator(srcName)

	if srcFileInfo.IsDir() {
		if gofilepath.HasEndPathSeparators(dstName) {
			err = cr.mkdir(dstName, srcFileInfo.Mode())
			if err != nil {
				return
			}
		}

		if !gofilepath.HasEndPathSeparators(srcName) {
			dstName = gofilepath.JointSmart(cr.dstPathSeparator, dstName, gofilepath.Base(srcName))
			err = cr.mkdir(dstName, srcFileInfo.Mode())
			if err != nil {
				return
			}
		}
	}
	cr.srcPath = srcName
	cr.dstPath = dstName

	if srcFileInfo.IsDir() {
		cr.srcPath = srcName
		err = cr.processDir(srcName, srcFileInfo)
		if err != nil {
			if cr.IgnErr {
				log.Warnf("error [ignore]: %v", err)
			} else {
				return
			}
		}
	} else {
		err = cr.copyFile(srcName, srcFileInfo)
		return err
	}
	return
}

//copy file or directory from fsh  to fs dirName
func Copy(toDirPath, fromFshPath string) (err error) {
	cr := &CopyRecursive{IsVerbose: true,
		IgnErr:   false,
		ReadFile: os.ReadFile,
		Mkdir: func(srcPath string, srcFileInfo fs.FileMode) error {
			if gosystem.PathIsExist(srcPath) {
				os.Chmod(srcPath, 0755)
				return nil
			}
			return os.Mkdir(srcPath, srcFileInfo)
		},
		Writer: os.WriteFile,
		Stat:   os.Stat,
		Open:   os.Open,
	}
	toDirPath = gofilepath.FromSlash(toDirPath)
	if !gofilepath.IsAbs(toDirPath) {
		toDirPath, err = gofilepath.Abs(toDirPath)
		if err != nil {
			return err
		}
	}
	return cr.Copy(toDirPath, fromFshPath)
}

type ReadDirFile interface {
	getpath() string
	setpath(newpath string)
	setindex(int) int
	getindex() int
	fs.StatFS
	fs.FileInfo
	fs.ReadDirFS
}

func Readdir(f ReadDirFile, count int) ([]os.FileInfo, error) {
	var fis []os.FileInfo
	if !f.IsDir() {
		return fis, nil
	}
	var entryDirs []fs.DirEntry
	var err error

	entryDirs, err = f.ReadDir(f.getpath())
	if err != nil {
		return nil, err
	}
	flen := len(entryDirs)

	// If dirIdx reaches the end and the count is a positive value,
	// an io.EOF error is returned.
	// In other cases, no error will be returned even if, for example,
	// you specified more counts than the number of remaining files.
	start := f.getindex()
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
	f.setindex(f.getindex() + len(fis))
	return fis, nil
}

func Open(name string) (hf http.File, err error) {
	var file = File{}
	var fileConten []byte

	var fstat fs.FileInfo
	file.fullpath = fsh.fullName(name)
	// name = fsh.fullName(name)
	fstat, err = fsh.Stat(name)
	if err != nil {
		return
	}

	if !fstat.IsDir() {
		if fileConten, err = fsh.FS.ReadFile(file.fullpath); err != nil {
			return
		}
		file.reader = bytes.NewReader(fileConten)
	}
	file.ReadDirFS = fsh.FS
	file.FileInfo = fstat
	return &file, nil
}
