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
	"strings"

	"github.com/sonnt85/gofilepath"
	"github.com/sonnt85/gosutils/sregexp"
	"github.com/sonnt85/gosystem"
)

// http.File
//
//	type File interface {
//		io.Closer x
//		io.Reader x
//		io.Seeker x
//		Readdir(count int) ([]fs.FileInfo, error) x
//		Stat() (fs.FileInfo, error)
//	}
type File struct {
	reader *bytes.Reader
	FileReadDir
}

func NewFile() (f *File) {
	f = new(File)
	return
}

type HttpSystem interface {
	SubFS
	fs.StatFS
	fs.ReadDirFS
	fs.ReadFileFS
}

type embedWrap struct {
	efs *embed.FS
}

func (ewrap *embedWrap) Open(name string) (fs.File, error) {
	return ewrap.efs.Open(name)
}

type embedSub struct {
	*embedWrap
	dir string
}

func NewEmbedSub() *embedSub {
	return &embedSub{embedWrap: new(embedWrap), dir: ""}
}

// fullName maps name to the fully-qualified name dir/name.
func (f *embedSub) fullName(op string, name string) (string, error) {
	name = strings.TrimPrefix(name, "./")
	name = strings.TrimPrefix(name, "/")
	if !fs.ValidPath(name) {
		return "", &fs.PathError{Op: op, Path: name, Err: errors.New("invalid name")}
	}
	return path.Join(f.dir, name), nil
}

// shorten maps name, which should start with f.dir, back to the suffix after f.dir.
// func (f *embedSub) shorten(name string) (rel string, ok bool) {
// 	if name == f.dir {
// 		return ".", true
// 	}
// 	if len(name) >= len(f.dir)+2 && name[len(f.dir)] == '/' && name[:len(f.dir)] == f.dir {
// 		return name[len(f.dir)+1:], true
// 	}
// 	return "", false
// }

func (esub *embedSub) ReadDir(name string) ([]fs.DirEntry, error) {
	fullname, err := esub.fullName("readdir", name)
	if err != nil {
		return nil, err
	}
	return esub.efs.ReadDir(fullname)
}
func (esub *embedSub) ReadFile(name string) ([]byte, error) {
	fullname, err := esub.fullName("readfile", name)
	if err != nil {
		return nil, err
	}
	return esub.efs.ReadFile(fullname)
}

// func (esub *embedSub) Sub(dir string) (subem fs.FS, err error) {
func (esub *embedSub) Sub(dir string) (subem *embedSub, err error) {
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: errors.New("invalid name")}
	}
	if dir == "." {
		return esub, nil
	}
	if _, err := esub.Open(dir); err != nil {
		return nil, err
	}
	newEsub := *esub
	newEsub.dir = path.Join(esub.dir, dir)
	return &newEsub, nil
}

func (esub *embedSub) SubSet(dir string) (err error) {
	if !fs.ValidPath(dir) {
		return &fs.PathError{Op: "sub", Path: dir, Err: errors.New("invalid name")}
	}
	if dir == "." {
		return nil
	}
	esub.dir = dir
	return nil
}

func (esub *embedSub) Open(name string) (fs.File, error) { //implement FS
	fullname, err := esub.fullName("open", name)
	if err != nil {
		return nil, err
	}
	return esub.efs.Open(fullname)
}

type HttpSystemFS struct {
	// fs.SubFS
	// fs.StatFS
	rootDir string
	*embedSub
}

func NewHttpSystemFS(efs *embed.FS, rootDir string, sub ...string) (*HttpSystemFS, error) {
	hfs := HttpSystemFS{embedSub: NewEmbedSub()}
	hfs.embedWrap.efs = efs
	hfs.rootDir = rootDir
	if len(sub) != 0 {
		return hfs.Sub(sub[0])
	}
	return &hfs, nil
}

func (fsh *HttpSystemFS) RootDir() string {
	return fsh.rootDir
}

func (fsh *HttpSystemFS) SetRootDir(rd string) {
	fsh.rootDir = rd
}

func (fsh *HttpSystemFS) Open(name string) (hf http.File, err error) {
	var file = File{}
	var fileConten []byte

	var fstat fs.FileInfo
	fstat, err = fsh.Stat(name)
	if err != nil {
		return
	}
	file.FileInfo = fstat
	file.shortName = name
	file.ReadDirFS = fsh.embedSub

	if !fstat.IsDir() {
		if fileConten, err = fsh.ReadFile(name); err != nil {
			return
		}
		file.reader = bytes.NewReader(fileConten)
	}
	return &file, nil
}

func (fsh *HttpSystemFS) OpenFile(name string) (hf *File, err error) {
	var file = File{}
	var fileConten []byte

	var fstat fs.FileInfo
	fstat, err = fsh.Stat(name)
	if err != nil {
		return
	}

	if !fstat.IsDir() {
		if fileConten, err = fsh.ReadFile(file.Name()); err != nil {
			return
		}
		file.reader = bytes.NewReader(fileConten)
	}
	file.ReadDirFS = fsh.embedSub
	file.FileInfo = fstat
	file.shortName = name
	return &file, nil
}
func (fsh *HttpSystemFS) FindFilesMatchPathFromRoot(rootSearch, pattern string, maxdeep int, matchfile, matchdir bool, matchFunc func(pattern, relpath string) bool) (matches []string) {
	matches = make([]string, 0)
	if matchFunc == nil {
		return
	}
	if len(rootSearch) == 0 || rootSearch == "/" || rootSearch == "." {
		rootSearch = fsh.rootDir
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
		return sregexp.New(pattern).MatchString(gofilepath.Base(relpath))
	}
	return fsh.FindFilesMatchPathFromRoot(root, pattern, maxdeep, matchfile, matchdir, matchFunc)
}

func (fsh *HttpSystemFS) FindFilesMatchName(root, pattern string, maxdeep int, matchfile, matchdir bool) (matches []string) {
	matchFunc := func(pattern, relpath string) bool {
		if match, err := filepath.Match(pattern, gofilepath.Base(relpath)); err == nil && match {
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
// type WalkDirFunc func(path string, d DirEntry, err error) error
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
	if len(root) == 0 || root == "/" || root == "." {
		root = fsh.rootDir
	}
	file, err := fsh.embedSub.Open(root)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.Stat()
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
	dirs, err := fsh.ReadDir(pathdir)
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

// copy file or directory from fsh  to fs dirName
func (fsh *HttpSystemFS) Copy(toDirPath, fromFshPath string, mods ...fs.FileMode) (err error) {
	// defer func() {
	// 	if err != nil {
	// 		//cleanup function
	// 	}
	// }()
	// fromFshPath = fsh.Getfullpath(fromFshPath)
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
		Open:   fsh.OpenFile,
	}
	toDirPath = gofilepath.FromSlash(toDirPath)
	// if !gofilepath.IsAbs(toDirPath) {
	// 	toDirPath, err = gofilepath.Abs(toDirPath)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	return cr.Copy(toDirPath, fromFshPath, mods...)
}

func (fsh *HttpSystemFS) Sub(dir string) (sub *HttpSystemFS, err error) {
	// return fsh.embedSub.Sub(dir)
	sub = new(HttpSystemFS)
	*sub = *fsh
	var subem *embedSub
	subem, err = fsh.embedSub.Sub(dir)
	if err != nil {
		return nil, err
	}
	sub.embedSub = subem
	return
}

func (fsh *HttpSystemFS) SubSet(dir string) (err error) {
	fsh.embedSub.SubSet(dir)
	return
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
