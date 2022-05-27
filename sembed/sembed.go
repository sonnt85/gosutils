package sembed

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/sonnt85/gofilepath"
	"github.com/sonnt85/gosutils/sregexp"
	"github.com/sonnt85/gosystem"
)

// http.File
type File struct {
	reader *bytes.Reader
	FileReadDir
}

type HttpSystemFS struct {
	fs.SubFS
	fs.StatFS
	subPath string
	*embed.FS
}

func NewHttpSystemFS(efs *embed.FS, sub ...string) *HttpSystemFS {
	subPath := ""
	if len(sub) != 0 {
		subPath = sub[0]
	}
	return &HttpSystemFS{
		FS:      efs,
		subPath: subPath,
	}
}

func (fsh *HttpSystemFS) ReadFile(name string) ([]byte, error) {
	return fsh.FS.ReadFile(fsh.fullName(name))
}

func (fsh *HttpSystemFS) fullName(name string) string {
	if len(fsh.subPath) != 0 {
		name = path.Join(fsh.subPath, name)
		// name = fsh.subPath + "/" + name
	}
	return name
}

func (fsh *HttpSystemFS) Open(name string) (hf http.File, err error) {
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

func (fsh *HttpSystemFS) Setsub(name string) {
	fsh.subPath = name
}

func (fsh *HttpSystemFS) FindFilesMatchPathFromRoot(rootSearch, pattern string, maxdeep int, matchfile, matchdir bool, matchFunc func(pattern, relpath string) bool) (matches []string) {
	matches = make([]string, 0)
	if matchFunc == nil {
		return
	}
	// rootSearch := gofilepath.FromSlash(root1)
	if finfo, err := fsh.Stat(rootSearch); err == nil {
		if !finfo.IsDir() { //is file
			if matchFunc(pattern, rootSearch) {
				matches = []string{rootSearch}
			}
			return
		}
	}
	// pattern = gofilepath.ToSlash(pattern)
	var relpath string
	var deep int
	if nil != fsh.WalkDir(rootSearch, func(path string, d fs.DirEntry, err error) error {
		if err != nil { //signaling that Walk will not walk into this directory.
			// return err
			return nil
		}
		relpath, err = gofilepath.RelSmart(rootSearch, path)
		if err != nil {
			return nil
		}
		if maxdeep > -1 {
			deep = gofilepath.CountPathSeparator(relpath)
			if deep > maxdeep {
				if d.IsDir() {
					return fs.SkipDir
				} else {
					return nil
				}
			}
		}
		if (d.IsDir() && matchdir) || (!d.IsDir() && matchfile) {
			if matchFunc(pattern, relpath) {
				matches = append(matches, path)
			}
		}
		return nil
	}) {
		return nil
	}
	return matches
}

// maxdeep: 0 ->
// func (fsh *HttpSystemFS) _FindFilesMatchPathFromRoot(root, pattern string, maxdeep int, matchfile, matchdir bool, matchFunc func(pattern, relpath string) bool) (matches []string) {
// 	return gofilepath.FindFilesMatchPathFromRoot(root, pattern, maxdeep, matchfile, matchdir, matchFunc, fsh.WalkDir)
// }

func (fsh *HttpSystemFS) FindFilesMatchRegexpPathFromRoot(root, pattern string, maxdeep int, matchfile, matchdir bool) (matches []string) {
	matchFunc := func(pattern, relpath string) bool {
		return sregexp.New(pattern).MatchString(relpath)
	}
	return fsh.FindFilesMatchPathFromRoot(root, pattern, maxdeep, matchfile, matchdir, matchFunc)
}

func (fsh *HttpSystemFS) FindFilesMatchRegexpName(root, pattern string, maxdeep int, matchfile, matchdir bool) (matches []string) {
	matchFunc := func(pattern, relpath string) bool {
		return sregexp.New(pattern).MatchString(filepath.Base(relpath))
	}
	return fsh.FindFilesMatchPathFromRoot(root, pattern, maxdeep, matchfile, matchdir, matchFunc)
}

func (fsh *HttpSystemFS) FindFilesMatchName(root, pattern string, maxdeep int, matchfile, matchdir bool) (matches []string) {
	matchFunc := func(pattern, relpath string) bool {
		if match, err := filepath.Match(pattern, filepath.Base(relpath)); err == nil && match {
			return true
		}
		return false
	}
	return fsh.FindFilesMatchPathFromRoot(root, pattern, maxdeep, matchfile, matchdir, matchFunc)
}

// WalkDirFunc is the type of the function called by WalkDir to visit
// each file or directory.
//
// The path argument contains the argument to WalkDir as a prefix.
// That is, if WalkDir is called with root argument "dir" and finds a file
// named "a" in that directory, the walk function will be called with
// argument "dir/a".
//
// The d argument is the fs.DirEntry for the named path.
//
// The error result returned by the function controls how WalkDir
// continues. If the function returns the special value SkipDir, WalkDir
// skips the current directory (path if d.IsDir() is true, otherwise
// path's parent directory). Otherwise, if the function returns a non-nil
// error, WalkDir stops entirely and returns that error.
//
// The err argument reports an error related to path, signaling that
// WalkDir will not walk into that directory. The function can decide how
// to handle that error; as described earlier, returning the error will
// cause WalkDir to stop walking the entire tree.
//
// WalkDir calls the function with a non-nil err argument in two cases.
//
// First, if the initial fs.Stat on the root directory fails, WalkDir
// calls the function with path set to root, d set to nil, and err set to
// the error from fs.Stat.
//
// Second, if a directory's ReadDir method fails, WalkDir calls the
// function with path set to the directory's path, d set to an
// fs.DirEntry describing the directory, and err set to the error from
// ReadDir. In this second case, the function is called twice with the
// path of the directory: the first call is before the directory read is
// attempted and has err set to nil, giving the function a chance to
// return SkipDir and avoid the ReadDir entirely. The second call is
// after a failed ReadDir and reports the error from ReadDir.
// (If ReadDir succeeds, there is no second call.)
//
// The differences between WalkDirFunc compared to filepath.WalkFunc are:
//
//   - The second argument has type fs.DirEntry instead of fs.FileInfo.
//   - The function is called before reading a directory, to allow SkipDir
//     to bypass the directory read entirely.
//   - If a directory read fails, the function is called a second time
//     for that directory to report the error.
//
//type WalkDirFunc func(path string, d DirEntry, err error) error
type WalkDirFunc fs.WalkDirFunc

// type WalkDirFunc func(path string, d DirEntry, err error) error

// WalkDir walks the file tree rooted at root, calling fn for each file or
// directory in the tree, including root.
//
// All errors that arise visiting files and directories are filtered by fn:
// see the fs.WalkDirFunc documentation for details.
//
// The files are walked in lexical order, which makes the output deterministic
// but requires WalkDir to read an entire directory into memory before proceeding
// to walk that directory.
//
// WalkDir does not follow symbolic links.
type statDirEntry struct {
	info fs.FileInfo
}

func (d *statDirEntry) Name() string               { return d.info.Name() }
func (d *statDirEntry) IsDir() bool                { return d.info.IsDir() }
func (d *statDirEntry) Type() fs.FileMode          { return d.info.Mode().Type() }
func (d *statDirEntry) Info() (fs.FileInfo, error) { return d.info, nil }

func (fsh *HttpSystemFS) Stat(root string) (finfo fs.FileInfo, err error) {
	root = fsh.fullName(root)
	return fs.Stat(fsh.FS, root)
}

func (fsh *HttpSystemFS) WalkDir(root string, fn WalkDirFunc) (err error) {
	info, err := fsh.Stat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = fsh.walkDir(root, &statDirEntry{info}, fn)
	}
	if err == fs.SkipDir {
		return nil
	}
	return err
}

// walkDir recursively descends path, calling walkDirFn.
func (fsh *HttpSystemFS) walkDir(pathdir string, d fs.DirEntry, walkDirFn WalkDirFunc) error {
	if err := walkDirFn(pathdir, d, nil); err != nil || !d.IsDir() {
		if err == fs.SkipDir && d.IsDir() {
			// Successfully skipped directory.
			err = nil
		}
		return err
	}
	dirs, err := fs.ReadDir(fsh.FS, pathdir)
	if err != nil {
		// Second call, to report ReadDir error.
		err = walkDirFn(pathdir, d, err)
		if err != nil {
			return err
		}
	}

	for _, d1 := range dirs {
		path1 := path.Join(pathdir, d1.Name())
		if err := fsh.walkDir(path1, d1, walkDirFn); err != nil {
			if err == fs.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

//copy file or directory from fsh  to fs dirName
func (fsh *HttpSystemFS) Copy(toDirPath, fromFshPath string) (err error) {
	// defer func() {
	// 	if err != nil {
	// 		//cleanup function
	// 	}
	// }()
	// fromFshPath = fsh.fullName(fromFshPath)
	cr := &CopyRecursive{IsVerbose: true,
		IgnErr:   false,
		ReadFile: fsh.ReadFile,
		Mkdir: func(srcPath string, srcFileInfo fs.FileMode) error {
			if gosystem.PathIsExist(srcPath) {
				os.Chmod(srcPath, 0755)
				return nil
			}
			return os.Mkdir(srcPath, srcFileInfo)
		},
		Writer: os.WriteFile,
		Stat:   fsh.Stat,
		Open:   fsh.Open,
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

func (fsh *HttpSystemFS) Sub(dir string) (sub *HttpSystemFS, err error) {
	sub = new(HttpSystemFS)
	*sub = *fsh
	sub.subPath = path.Join(sub.subPath, dir)
	var f fs.File
	if f, err = sub.FS.Open(sub.subPath); err != nil {
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

func (f *File) Close() error {
	return nil
}

// Read reads bytes into p, returns the number of read bytes.
func (f *File) Read(p []byte) (n int, err error) {
	n, err = f.reader.Read(p)
	return
}

// Seek seeks to the offset.
func (f *File) Seek(offset int64, whence int) (ret int64, err error) {
	return f.reader.Seek(offset, whence)
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f.FileInfo, nil
}

// IsDir returns true if the file location represents a directory.
func (f *File) IsDir() bool {
	return f.FileInfo.IsDir()
}
