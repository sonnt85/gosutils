package sutils

import (
	"context"
	"math/big"
	"runtime"
	"sort"

	"github.com/antchfx/jsonquery"
	"github.com/antchfx/xmlquery"
	"github.com/beevik/etree"
	"github.com/tidwall/sjson"

	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"

	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/fs"
	"io/ioutil"

	xj "github.com/basgys/goxml2json"

	log "github.com/sirupsen/logrus"

	"net"
	"net/http"
	"net/smtp"

	"os"
	"path/filepath"

	"github.com/jordan-wright/email"

	//	"io"
	"io"
	//	"mime"
	"bufio"
	"bytes"

	//	"errors"
	"errors"
	"os/exec"
	"os/user"
	"reflect"

	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rs/xid"
	"github.com/sonnt85/gosutils/gogrep"
	"github.com/sonnt85/gosutils/sregexp"

	"github.com/sonnt85/gosutils/gosed"
)

//func WindowsRunMeElevated() {
//	verb := "runas"
//	exe, _ := os.Executable()
//	cwd, _ := os.Getwd()
//	args := strings.Join(os.Args[1:], " ")
//
//	verbPtr, _ := syscall.UTF16PtrFromString(verb)
//	exePtr, _ := syscall.UTF16PtrFromString(exe)
//	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
//	argPtr, _ := syscall.UTF16PtrFromString(args)
//
//	var showCmd int32 = 1 //SW_NORMAL
//
//	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
//	if err != nil {
//		fmt.Println(err)
//	}
//}
var (
	GOOS       = runtime.GOARCH
	AppName    string
	AppVersion string
	NEWLINE    = "\n"
	Ipv4Regex  = `([0-9]+\.){3}[0-9]+`
)

////progressbar
const DEFAULT_FORMAT = "\r%s   %3d %%  %d kb %0.2f kb/s %v      "

type ProgressBar struct {
	Out       io.Writer
	Format    string
	Subject   string
	StartTime time.Time
	Size      int64
}

func FileIWriteable(path string) (isWritable bool) {
	isWritable = false

	if file, err := os.OpenFile(path, os.O_WRONLY, 0666); err == nil {
		defer file.Close()
		isWritable = true
	} else {
		if os.IsPermission(err) {
			return false
		}
	}

	return
}

// removeFile removes the specified file. Errors are ignored.
func FileremoveFile(path string) error {
	return os.Remove(path)
}

func FileCloneDate(dst, src string) bool {
	var err error
	var srcinfo os.FileInfo
	if srcinfo, err = os.Stat(src); err == nil {
		if err = os.Chtimes(dst, srcinfo.ModTime(), srcinfo.ModTime()); err == nil {
			return true
		}
	}
	//	fmt.Errorf("Cannot clone date file ", err)
	return false
}

func FileAllChild(directory string) (err error) {
	dirRead, _ := os.Open(directory)
	dirFiles, _ := dirRead.Readdir(0)
	for index := range dirFiles {
		fileHere := dirFiles[index]

		// Get name of file and its full path.
		nameHere := fileHere.Name()
		fullPath := directory + nameHere

		// Remove the file.
		os.Remove(fullPath)
		fmt.Println("Removed file:", fullPath)
	}
	return nil
}

func FileHashMd5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil
}

// waitForFile waits for the specified file to exist before returning. If the an
// error, other than the file not existing, occurs, the error is returned. If,
// after 100 attempts, the file does not exist, an error is returned.
func FileWaitForFileExist(path string, timeoutms int) error {
	if timeoutms < 50 && timeoutms != 0 {
		timeoutms = 50
	}

	for i := 0; i < timeoutms/50; i++ {
		_, err := os.Stat(path)
		if err == nil || !os.IsNotExist(err) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("file does not exist: %s", path)
}

// readFile waits for the specified file to contain contents, and then returns
// those contents as a string. If an error occurs while reading the file, the
// error is returned. If the file has no content after 100 attempts, an error is
// returned.
func FileWaitContentsAndRead(path string, timeoutms int) (string, error) {
	if timeoutms < 50 && timeoutms != 0 {
		timeoutms = 50
	}
	for i := 0; i < timeoutms/50; i++ {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}
		if len(bytes) > 0 {
			return strings.TrimSpace(string(bytes)), err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return "", fmt.Errorf("file is empty: %s", path)
}

//pathfile, tocontents, grepstring, pattern string, literalGrepFlag bool, linesinserts ...int
func FileUpdateOrAdd(pathfile, tocontents, grepstring, pattern string, literalGrepFlag bool, linesinserts ...int) (err error) {
	linesinsert := 1
	if len(linesinserts) != 0 && linesinserts[0] != 0 {
		linesinsert = linesinserts[0]
	}
	if PathIsFile(pathfile) {
		if !gogrep.FileIsMatchLines(pathfile, grepstring, literalGrepFlag) {
			err := gosed.FileReplaceRegex(pattern, tocontents, pathfile)
			//				_, _, err := gosed.Sed(pattern, pathfile) //update
			if err != nil {
				//					fmt.Println("gosed.Sed", err)
			}
			if !gogrep.FileIsMatchLines(pathfile, grepstring, literalGrepFlag) { //not found
				return FileInsertStringAtLine(pathfile, tocontents, linesinsert)
			}
		}
		//not change
	} else { //create new file if has content
		if len(tocontents) != 0 {
			return ioutil.WriteFile(pathfile, []byte(tocontents), os.FileMode(0644))
		}
	}

	return nil
}

//pathfile, tocontents, grepstring, pattern string, literalGrepFlag bool, linesinserts ...int
func FileCreatenewIfDiff(pathfile, tocontents, grepstring string, literalGrepFlag bool) (err error) {
	if PathIsFile(pathfile) {
		if gogrep.FileIsMatchLines(pathfile, grepstring, literalGrepFlag) {
			return nil
		}
		//not change
	}
	//	if len(tocontents) != 0 {
	return ioutil.WriteFile(pathfile, []byte(tocontents), os.FileMode(0644))
	//	}
}

func FileGetSize(filepath string) (int64, error) {
	fi, err := os.Stat(filepath)
	if err != nil {
		return 0, err
	}
	// get the size
	return fi.Size(), nil
}

func FileWriteStringIfChange(pathfile string, contents []byte) (bool, error) {

	oldContents := []byte{}
	if _, err := os.Stat(pathfile); err == nil {
		oldContents, _ = ioutil.ReadFile(pathfile)
	}

	if bytes.Compare(oldContents, contents) != 0 {
		return true, ioutil.WriteFile(pathfile, contents, 0644)
	} else {
		return false, nil
	}
}

// copy file from src to dst
func FileCopy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, errors.New(fmt.Sprintf("%s is not a regular file ", src))
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	if err == nil {
		os.Chmod(dst, sourceFileStat.Mode())
		os.Chtimes(dst, sourceFileStat.ModTime(), sourceFileStat.ModTime())
	}
	return nBytes, err
}

func init() {
	GOOS := runtime.GOOS
	if GOOS != "windows" {
		NEWLINE = "\r\n"
	} else if GOOS != "darwin" {
		NEWLINE = "\r"
	}
}

func FileInsertStringAtLine(filePath, str string, index int) error {
	NEWLINE := "\n"
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	str = str + NEWLINE //add newline
	scanner := bufio.NewScanner(f)
	lines := ""
	linenum := 0
	inserted := false
	for scanner.Scan() {
		linenum = linenum + 1
		if linenum == index {
			inserted = true
			lines = lines + str
		}
		lines = lines + scanner.Text() + NEWLINE
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if !inserted {
		if index == -1 {
			index = linenum + 1
		}
		for i := linenum + 1; i < index; i++ {
			lines = lines + NEWLINE
		}
		lines = lines + str
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, []byte(lines), info.Mode().Perm())
}

func DirRemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func WindowsIsAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		//        fmt.Println("admin no")
		return false
	}
	//    fmt.Println("admin yes")
	return true
}

func GetHomeDir() (home string) {
	home, err := os.UserHomeDir()
	if err == nil {
		return home
	} else {
		return ""
	}
}

func SysGetHomeDir() (home string) {
	home, err := os.UserHomeDir()
	if err == nil {
		return home
	} else {
		return ""
	}
}

func SysGetUsername() string {
	if user, err := user.Current(); err == nil {
		return user.Username
	} else {
		return ""
	}
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

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func TouchFile(name string) error {
	file, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	return file.Close()
	//	return nil
}

func Cat(files ...string) (err error) {
	for _, fname := range files {
		fh, err := os.Open(fname)
		if err != nil {
			return err
		}
		_, err = io.Copy(os.Stdout, fh)
		if err != nil {
			return err
		}
	}
	return nil
}

//get arg at index 'index' of args
func ArgsGet(index int, args []string) string {
	if len(args) > index {
		return args[index]
	} else {
		return ""
	}
}

//func _IDget(listdir) (file, id string)
func IDGenerate() string {
	return xid.New().String()
}

func PathJointList(path, data string) string {
	//	data = data + string(os.PathSeparator)
	if len(path) == 0 {
		return data
	}
	return path + string(os.PathListSeparator) + data
	//	filepath.ListSeparator
}

func PathRemove(PATH, addpath string) string {
	if len(PATH) == 0 {
		return ""
	}
	newpath := ""
	for i, val := range strings.Split(PATH, string(os.PathListSeparator)) {
		if !(val == addpath) {
			if i == 0 {
				newpath = val
			} else {
				newpath = newpath + string(os.PathListSeparator) + val
			}
		}
	}
	return newpath
	//	filepath.ListSeparator
}

func PATHHasFile(filePath, PATH string) bool {
	execbasename := filepath.Base(filePath)
	for _, val := range strings.Split(PATH, string(os.PathListSeparator)) {
		if PathIsFile(filepath.Join(val, execbasename)) {
			return true
		}
	}
	return false
}

func PathGetEnvPathValue() string {
	for _, pathname := range []string{"PATH", "path"} {
		path := os.Getenv(pathname)
		if len(path) != 0 {
			return path
		}
	}
	return ""
}

func PATHArr() []string {
	envs := PathGetEnvPathValue()
	if len(envs) != 0 {
		return strings.Split(envs, string(os.PathListSeparator))
	}
	return []string{}
}

func PathGetEnvPathKey() string {
	for _, pathname := range []string{"PATH", "path"} {
		path := os.Getenv(pathname)
		if len(path) != 0 {
			return pathname
		}
	}
	return ""
}

func TempFileCreateInNewTemDir(filename string) string {

	rootdir, err := ioutil.TempDir("", "system")
	if err != nil {
		return ""
	} else {
		//			defer os.RemoveAll(dir)
	}

	return filepath.Join(rootdir, filename)
}

func TempFileCreateInNewTemDirWithContent(filename string, data []byte) string {
	rootdir, err := ioutil.TempDir("", "system")
	if err != nil {
		return ""
	}
	fPath := filepath.Join(rootdir, filename)
	err = os.WriteFile(fPath, data, 0755)
	if err != nil {
		os.RemoveAll(rootdir)
		return ""
	}
	return fPath
}

func TempFileCreate() string {
	if f, err := ioutil.TempFile("", "system"); err == nil {
		defer f.Close()
		return f.Name()
	} else {
		return ""
	}
}

func TempFileCreateWithContent(data []byte) string {
	if f, err := ioutil.TempFile("", "system"); err == nil {
		var n int
		if n, err = f.Write(data); err != nil && n == len(data) {
			f.Close()
			os.Remove(f.Name())
			return ""
		}
		f.Close()
		return f.Name()
	} else {
		return ""
	}
}

func IsContainer() bool {
	return gogrep.FileIsMatchLiteralLine("/proc/self/cgroup", "docker") || gogrep.FileIsMatchLiteralLine("/proc/self/cgroup", "lxc")
}

func IsPortOpen(addr string, port int, proto string, timeouts ...time.Duration) bool {
	timeout := time.Millisecond * 500
	if len(timeouts) != 0 {
		timeout = timeouts[0]
	}
	if len(proto) == 0 {
		proto = "tcp"
	}
	conn, err := net.DialTimeout(proto, fmt.Sprintf("%s:%d", addr, port), timeout)
	if err == nil {
		conn.Close()
		return true
	}
	return false
}

func IsPortAvailable(ip string, port int, timeout int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))

	if err == nil {
		conn.Close()
		return true
	} else {
		//		if timeout == 0 {
		//			return false
		//		}
		//		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), time.Duration(timeout)*time.Second)
		//
		//		if err, _ := err.(*net.OpError); err != nil {
		//			//ok && err.TimeOut()
		//			//			log.Printf("Timeout error: %s\n", err)
		//			return true
		//		}
		//
		//		if err != nil {
		//			// Log or report the error here
		//			//			log.Printf("Error: %s\n", err)
		//			return true
		//		} else {
		//			defer conn.Close()
		//		}
		return false
	}
}

func IsPortUsed(ip string, port int, timeout int) bool {
	return !IsPortAvailable(ip, port, timeout)
}

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func GetFreePorts(count int) ([]int, error) {
	var ports []int
	for i := 0; i < count; i++ {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return nil, err
		}

		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return nil, err
		}
		defer l.Close()
		ports = append(ports, l.Addr().(*net.TCPAddr).Port)
	}
	return ports, nil
}

func CreateSha1(data []byte) string {
	h := sha1.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func str2Sha1(data string) string {
	h := sha1.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func Int64ToBytes(number int64) []byte {
	big := new(big.Int)
	big.SetInt64(number)
	return big.Bytes()
}

func TokenCreate(key int) string {
	if key == 0 {
		key = 1985
	}
	nowtimestam := time.Now().Unix() + int64(key)
	return CreateSha1(Int64ToBytes(nowtimestam))
}

func TokenIsMatch(key int, token string) bool {
	if key == 0 {
		key = 1985
	}
	nowtimestam := time.Now().Unix() + int64(key)
	for i := -15; i < 15; i++ {
		if token == str2Sha1(strconv.FormatInt((nowtimestam+int64(i)), 10)) {
			return true
		}
	}
	return false
}

func PathIsExist(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func IoCopy(sourceFile, destinationFile string) bool {
	input, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		log.Println(err)
		return false
	}

	err = ioutil.WriteFile(destinationFile, input, 0644)
	if err != nil {
		log.Println("Error creating", destinationFile)
		log.Println(err)
		return false
	}
	return true
}

func PathIsDir(path string) bool {
	if finfo, err := os.Stat(path); err == nil {
		if finfo.IsDir() {
			return true
		}
	}
	return false
}

func PathIsFile(path string) bool {
	if finfo, err := os.Stat(path); err == nil {
		if !finfo.IsDir() {
			return true
		}
	}
	return false
}

func String2lines(str string) []string {
	scanner := bufio.NewScanner(strings.NewReader(str))

	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return []string{}
	}

	return lines
}

func File2lines(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return LinesFromReader(f)
}

func LinesFromReader(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func walk(filename string, linkDirname string, walkFn filepath.WalkFunc) error {
	symWalkFunc := func(path string, info os.FileInfo, err error) error {

		if fname, err := filepath.Rel(filename, path); err == nil {
			path = filepath.Join(linkDirname, fname)
		} else {
			return err
		}

		if err == nil && info.Mode()&os.ModeSymlink == os.ModeSymlink {
			finalPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			info, err := os.Lstat(finalPath)
			if err != nil {
				return walkFn(path, info, err)
			}
			if info.IsDir() {
				return walk(finalPath, path, walkFn)
			}
		}

		return walkFn(path, info, err)
	}
	return filepath.Walk(filename, symWalkFunc)
}

// Walk extends filepath.Walk to also follow symlinks
func SymWalk(path string, walkFn filepath.WalkFunc) error {
	return walk(path, path, walkFn)
}

func FindFileWithExt(pathS, ext string) (files []string) {
	ext = "." + ext
	//	pathR := pathS
	pathR, err := filepath.EvalSymlinks(pathS)
	if err != nil {
		return files
	}
	//	if _, err := os.Lstat(pathS); err == nil {
	//		if pathR, err = os.Readlink(pathS); err != nil {
	//			return files
	//		}
	//	}

	if !PathIsExist(pathR) {
		return files
	}

	if PathIsFile(pathR) {
		if filepath.Ext(pathS) == ext {
			files = append(files, pathS)
		}
		return files
	}

	if !PathIsDir(pathR) {
		return files
	}

	SymWalk(pathR, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			if extf := filepath.Ext(path); extf == ext || (ext == "." && extf == "") {
				files = append(files, path)
			}
		}
		return nil
	})

	for i, v := range files {
		regx := sregexp.New("^" + pathR)
		regx.Regexp()
		//		files[i] = pathS + strings.TrimLeft(files[i], pathR)
		files[i] = regx.ReplaceAllString(v, pathS)
		//		files[i] = strings.Replace(files[i], pathR, pathS, 1)
	}
	return files
}

func FileFindMatchNameRegx(root, pattern string) []string {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil { //signaling that Walk will not walk into this directory.
			return err
		}
		if info.IsDir() {
			return nil
		}
		fname := filepath.Base(path)
		if sregexp.New(pattern).MatchString(fname) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return matches
}

func FileFindWithExtRegx(root, pattern string) []string {
	var matches []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})

	if err != nil {
		return nil
	}
	return matches
}

func FindFile(pathS string) (files []string) {
	if !PathIsExist(pathS) {
		return files
	}

	if PathIsFile(pathS) {
		files = append(files, pathS)
		return files
	}

	if !PathIsDir(pathS) {
		return files
	}

	filepath.WalkDir(pathS, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func HTTPDownLoadUrl(urlpath, httpmethod, username, password string, insecure_flag bool, timeouts ...time.Duration) (byterets []byte, err error) {
	byterets = make([]byte, 0, 0)
	// Generated reqby curl-to-Go: https://mholt.github.io/curl-to-go

	// TODO: This is insecure; use only in dev environments.
	timeout := time.Millisecond * 1000
	if len(timeouts) != 0 {
		timeout = timeouts[0]
	}
	client := &http.Client{Timeout: timeout}
	if insecure_flag {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = tr
	}

	req, err := http.NewRequest(httpmethod, urlpath, nil)
	if err != nil {
		// handle err
		return byterets, err
	}

	if len(username) != 0 {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
		// handle err
		return byterets, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	//	defer fmt.Println(urlpath, err)
	return bodyBytes, nil
}

func HTTPDownLoadUrlToFile(urlpath, username, password string, insecure_flag bool, filePath string, timeouts ...time.Duration) (err error) {
	// Create the file
	tmpFile := TempFileCreateInNewTemDir("httpd")
	defer os.RemoveAll(filepath.Dir(tmpFile))
	out, err := os.Create(tmpFile)

	if err != nil {
		return err
	}
	defer out.Close()
	// Generated reqby curl-to-Go: https://mholt.github.io/curl-to-go

	// TODO: This is insecure; use only in dev environments.
	timeout := time.Millisecond * 1000
	if len(timeouts) != 0 {
		timeout = timeouts[0]
	}
	client := &http.Client{Timeout: timeout}
	if insecure_flag {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = tr
	}

	req, err := http.NewRequest("GET", urlpath, nil)
	if err != nil {
		// handle err
		return err
	}

	if len(username) != 0 {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
		// handle err
		return err
	}
	defer resp.Body.Close()
	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return os.Rename(tmpFile, filePath)
}

func HTTPDownLoadUrlToTmp(urlpath, username, password string, insecure_flag bool, timeouts ...time.Duration) (tmpfile string, err error) {
	// Create the file
	tmpfile = TempFileCreate()
	if err := HTTPDownLoadUrlToFile(urlpath, username, password, insecure_flag, tmpfile, timeouts...); err == nil {
		return tmpfile, nil
	} else {
		return "", err
	}
}

func UniqueI(intSlice []interface{}) interface{} {
	var list []interface{}
	keys := make(map[interface{}]bool)
	for i := 0; i < len(intSlice); i++ {
		entry := intSlice[i]
		if _, ok := keys[entry]; !ok {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func IsProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func JsonGetPathNode(node *jsonquery.Node) string {
	retstr := node.Data
	if node.Type == jsonquery.TextNode {
		retstr = "/@" + retstr
	} else {
		retstr = "/" + retstr
	}
	tmpnode := node
	for {
		if tmpnode = tmpnode.Parent; tmpnode == nil {
			retstr = strings.Replace(retstr, "//", "/", 1)
			break
		} else {
			retstr = "/" + tmpnode.Data + retstr
		}
	}

	return retstr
}

func JsonSet(jsonstring string, elementPath string, val any) (string, error) {
	return sjson.Set(jsonstring, elementPath, val)
}

func JsonStringFindElements(strjson *string, pathSearch string) (map[string]string, error) {
	var retmap = map[string]string{}
	doc, err := jsonquery.Parse(strings.NewReader(*strjson))
	if err != nil {
		//		fmt.Println("xmlquery.Parse:", err)
		return retmap, err
	}

	nodes, err := jsonquery.QueryAll(doc, pathSearch)
	if err != nil {
		return retmap, err
	}
	if len(nodes) == 0 {
		return retmap, errors.New("missing keypath")
	}
	//	found := false
	//	fmt.Println("scan nodes", err)
	id := 0
	//	numnodes := len(nodes)
	for k := 0; k < len(nodes); k++ {
		v := nodes[k]
		key := JsonGetPathNode(v)
		//		fmt.Println("key:", key, v)
		exist := func() bool {
			for i := 0; i < k; i++ {
				if key == JsonGetPathNode(nodes[i]) {
					return true
				}
			}
			return false
		}()

		//		found = true
		//		fmt.Println("xmlStringFindElement:", v.NamespaceURI, v.InnerText())
		if exist {
			key = key + "[" + strconv.Itoa(id) + "]"
			id = id + 1
		}
		//|| v.Type == xmlquery.AttributeNode
		if v.FirstChild == nil || v.FirstChild.FirstChild == nil {
			retmap[key] = v.InnerText()
		} else {
			xml := strings.NewReader(v.OutputXML())
			json, err := xj.Convert(xml)
			if err != nil {
				retmap[key] = v.OutputXML()
			} else {
				retmap[key] = json.String()
			}
			//			retmap[key] = v.OutputXML(true)
		}
		//		retmap[strconv.Itoa(id)] = v.InnerText()
	}

	//	if found {
	//		fmt.Println("retmap", retmap)
	return retmap, nil
	//	} else {
	//		return retmap, errors.New("Can not found")
	//	}
}

func JsonStringFindElementsSlide(strjson *string, pathSearch string) ([]string, error) {
	var retslide = []string{}
	doc, err := jsonquery.Parse(strings.NewReader(*strjson))
	if err != nil {
		//		fmt.Println("xmlquery.Parse:", err)
		return retslide, err
	}

	nodes, err := jsonquery.QueryAll(doc, pathSearch)
	if err != nil {
		return retslide, err
	}
	//	numnodes := len(nodes)
	for k := 0; k < len(nodes); k++ {
		v := nodes[k]

		if v.FirstChild == nil || v.FirstChild.FirstChild == nil {
			retslide = append(retslide, v.InnerText())
		} else {
			xml := strings.NewReader(v.OutputXML())
			json, err := xj.Convert(xml)
			if err != nil {
				retslide = append(retslide, v.OutputXML())
			} else {
				retslide = append(retslide, json.String())
			}
			//			retmap[key] = v.OutputXML(true)
		}
		//		retmap[strconv.Itoa(id)] = v.InnerText()
	}
	return retslide, nil
}

func JsonStringFindElement(strjson *string, pathSearch string) (string, error) {
	if retmap, err := JsonStringFindElements(strjson, pathSearch); err == nil && len(retmap) != 0 {
		keys := make([]string, 0, len(retmap))
		for k := range retmap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return retmap[keys[0]], nil
	} else {
		return "", err
	}
}

func XmlGetPathNode(node *xmlquery.Node) string {
	retstr := node.Data
	if node.Type == xmlquery.AttributeNode {
		retstr = "/@" + retstr
	} else {
		retstr = "/" + retstr
	}
	tmpnode := node
	for {
		if tmpnode = tmpnode.Parent; tmpnode == nil {
			retstr = strings.Replace(retstr, "//", "/", 1)
			break
		} else {
			retstr = "/" + tmpnode.Data + retstr
		}
	}

	return retstr
}

func StringIsXml(input *string) bool {
	_, err := xmlquery.Parse(strings.NewReader(*input))
	return err == nil
}

func XmlStringFindElements(strxml *string, pathSearch string) (map[string]string, error) {
	var retmap = map[string]string{}
	doc, err := xmlquery.Parse(strings.NewReader(*strxml))
	if err != nil {
		//		fmt.Println("xmlquery.Parse:", err)
		return retmap, err
	}
	nodes, err := xmlquery.QueryAll(doc, pathSearch)
	if err != nil {
		return retmap, err
	}
	found := false
	//	fmt.Println("scan nodes", err)
	id := 0
	//	numnodes := len(nodes)
	nodesText := ""
	for k := 0; k < len(nodes); k++ {
		v := nodes[k]
		nodesText += v.InnerText()

		//	for k, v := range nodes {
		key := XmlGetPathNode(v)
		//		fmt.Println("key:", key)
		exist := func() bool {
			for i := 0; i < k; i++ {
				if key == XmlGetPathNode(nodes[i]) {
					return true
				}
			}
			return false
		}()

		found = true
		//		fmt.Println("xmlStringFindElement:", v.NamespaceURI, v.InnerText())
		if exist {
			key = key + "[" + strconv.Itoa(id) + "]"
			id = id + 1
		}
		//|| v.Type == xmlquery.AttributeNode
		if v.FirstChild == nil || v.FirstChild.FirstChild == nil {
			retmap[key] = v.InnerText()
		} else {
			xml := strings.NewReader(v.OutputXML(true))
			json, err := xj.Convert(xml)
			if err != nil {
				retmap[key] = v.OutputXML(true)
			} else {
				retmap[key] = json.String()
			}
			//			retmap[key] = v.OutputXML(true)
		}
		//		retmap[strconv.Itoa(id)] = v.InnerText()
	}

	if found {
		//		fmt.Println("retmap", retmap)
		return retmap, nil
	} else {
		return retmap, fmt.Errorf(`can not found from respone: %s`, nodesText)
	}
}

func XmlStringFindElementsSlide(strxml *string, pathSearch string) ([]string, error) {
	var retslide = []string{}
	doc, err := xmlquery.Parse(strings.NewReader(*strxml))
	if err != nil {
		//		fmt.Println("xmlquery.Parse:", err)
		return retslide, err
	}

	nodes, err := xmlquery.QueryAll(doc, pathSearch)
	if err != nil {
		return retslide, err
	}
	//	fmt.Println("scan nodes", err)
	//	numnodes := len(nodes)
	for k := 0; k < len(nodes); k++ {
		v := nodes[k]
		//		fmt.Println("xmlStringFindElement:", v.NamespaceURI, v.InnerText())
		//|| v.Type == xmlquery.AttributeNode
		if v.FirstChild == nil || v.FirstChild.FirstChild == nil {
			retslide[k] = v.InnerText()
		} else {
			xml := strings.NewReader(v.OutputXML(true))
			json, err := xj.Convert(xml)
			if err != nil {
				retslide[k] = v.OutputXML(true)
			} else {
				retslide[k] = json.String()
			}
			//			retmap[key] = v.OutputXML(true)
		}
		//		retmap[strconv.Itoa(id)] = v.InnerText()
	}

	if len(retslide) != 0 {
		//		fmt.Println("retmap", retmap)
		return retslide, nil
	} else {
		return retslide, errors.New("Can not found")
	}
}

func XmlStringFindElement(strxml *string, pathSearch string) (string, error) {
	doc, err := xmlquery.Parse(strings.NewReader(*strxml))
	if err != nil {
		//		fmt.Println("xmlquery.Parse:", err)
		return "", err
	}

	nodes, err := xmlquery.QueryAll(doc, pathSearch)
	if err != nil {
		return "", err
	}
	//	fmt.Println("scan nodes", err)
	//	numnodes := len(nodes)
	for k := 0; k < len(nodes); k++ {
		v := nodes[k]
		//		fmt.Println("xmlStringFindElement:", v.NamespaceURI, v.InnerText())
		//|| v.Type == xmlquery.AttributeNode
		if v.FirstChild == nil || v.FirstChild.FirstChild == nil {
			return v.InnerText(), nil
		} else {
			xml := strings.NewReader(v.OutputXML(true))
			json, err := xj.Convert(xml)
			if err != nil {
				return v.OutputXML(true), nil
			} else {
				return json.String(), nil
			}
			//			retmap[key] = v.OutputXML(true)
		}
		//		retmap[strconv.Itoa(id)] = v.InnerText()
	}
	return "", errors.New("Can not found")
}

func XmlEtreeStringFindElement(strxml, pathSearch *string) ([]string, error) {
	var retstr = []string{}
	doc := etree.NewDocument()
	fmt.Println("pathSearch", *pathSearch)
	if err := doc.ReadFromString(*strxml); err != nil {
		//		fmt.Println("xmlStringFindElement/ReadFromString:", err)
		return retstr, err
	} else {
		docroot := doc.Root()
		//		fmt.Println("xmlStringFindElementstrxml:", docroot.Text(), ":end")
		found := false
		for _, v := range docroot.FindElements(*pathSearch) {
			found = true
			//			fmt.Println("xmlStringFindElement:", v.Text())
			retstr = append(retstr, v.Text())
		}

		if found {
			fmt.Println("retstr", retstr)
			return retstr, nil
		} else {
			return retstr, errors.New("Can not found")
		}
	}
}

func StringEncrypt(stringToEncrypt string, keyString string) (reterr error, encryptedString string) {

	//Since the key is in string, we need to convert decode it to bytes
	key, _ := hex.DecodeString(keyString)
	plaintext := []byte(stringToEncrypt)

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return err, encryptedString
	}

	//Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	//https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err, encryptedString
	}

	//Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return err, encryptedString
	}

	//Encrypt the data using aesGCM.Seal
	//Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return nil, fmt.Sprintf("%x", ciphertext)
}

func StringCreateKeyFromString(strkeys string, keylen int) (decryptedString string) {
	bytes := make([]byte, keylen)
	if len(strkeys) > keylen {
		strkeys = strkeys[0:(keylen - 1)]
	}
	for i := 0; i < len(strkeys); i++ {
		bytes[i] = byte(strkeys[i])
	}
	return hex.EncodeToString(bytes)
}

func StringDecrypt(encryptedString string, keyString string) (reterr error, decryptedString string) {
	decryptedString = ""
	key, _ := hex.DecodeString(keyString)
	enc, _ := hex.DecodeString(encryptedString)

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return err, encryptedString
	}

	//Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err, encryptedString
	}

	//Get the nonce size
	nonceSize := aesGCM.NonceSize()

	//Extract the nonce from the encrypted data
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	//Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err, encryptedString
	}

	return nil, fmt.Sprintf("%s", plaintext)
}

func StringDuplicate(s string, n int) string {
	ret := ""
	for i := 0; i < n; i++ {
		ret += s
	}
	return ret
}

func GmailSend(usermail, password, from, subject string, to []string, msg []byte, attach interface{}, filename string) (err error) {
	return EmailSend("smtp.gmail.com:587", usermail, password, from, subject, to, msg, attach, filename)
}

func EmailSend(smtphost, usermail, password, from, subject string, to []string, msg []byte, attach interface{}, filename string) (err error) {
	if len(usermail) == 0 && len(password) == 0 {
		usermail = "cloudiotecloud@gmail.com"
		password = "cloudiot123cloud"
	}
	if len(smtphost) == 0 {
		smtphost = "smtp.gmail.com:587"
	}
	e := email.NewEmail()
	e.From = from //"Jordan Wright <test@gmail.com>"
	e.To = to     //[]string{"test@example.com"}
	attachtype := reflect.TypeOf(attach)
	//	log.Println(attachtype.Kind())
	var mimetype = "application/octet-stream"
	if len(filename) == 0 {
		filename = "dummyfilename"
	}
	if attachbytes, ok := attach.([]byte); ok {
		// use b as []byte
		attachreader := bytes.NewReader(attachbytes)
		mimetype := http.DetectContentType(attachbytes)

		e.Attach(attachreader, filename, mimetype)
	} else if attachtype.Kind() == reflect.String {
		attachstr, _ := attach.(string)
		//		log.Println(attachstr)
		if len(attachstr) > 0 {
			if PathIsFile(attachstr) {
				e.AttachFile(attachstr)
			} else {
				attachreader := bytes.NewReader([]byte(attachstr))
				mimetype := http.DetectContentType([]byte(attachstr))
				e.Attach(attachreader, filename, mimetype)
			}
		}
	} else if attachtype.Kind() == reflect.Ptr {
		//		e.Attachments
		attachreader, ok := attach.(io.Reader)

		if !ok {
			return err
		}

		//		defer attachreader.Close()
		//		io.TeeReader
		buff, err := ioutil.ReadAll(attachreader)

		if err != nil {
			return err
		}
		attachbytereader := bytes.NewReader(buff)

		mimetype = http.DetectContentType(buff)

		e.Attach(attachbytereader, filename, mimetype)
	}
	//	e.Bcc = []string{"test_bcc@example.com"}
	//	e.Cc = []string{"test_cc@example.com"}
	e.Subject = subject //"Awesome Subject"
	e.Text = msg        //[]byte("Text Body is, of course, supported!")
	//	e.HTML = []byte("<h1>Fancy HTML is supported, too!</h1>")
	reg := regexp.MustCompile(`:.+$`)
	smtphostnoport := reg.ReplaceAllString(smtphost, "")

	if err = e.Send(smtphost, smtp.PlainAuth("", usermail, password, smtphostnoport)); err != nil {
		log.Printf("Can not send email: %s", err)
		return err
	}
	return err
}

func TeeReadWriter(cmd_stdin_pipe io.Writer, cmd_stdoout_pipe, cmd_stderr_pipe io.Reader, ptyFile io.ReadWriter, ptyErr, tee io.Writer, closefunc func()) {
	var once sync.Once
	stdout_pr, stdout_pw := io.Pipe()
	stderr_pr, stderr_pw := io.Pipe()
	stdin_pr, stdin_pw := io.Pipe()

	deffunc := func() {
		once.Do(func() {
			if closefunc != nil {
				closefunc()
			}
			stdin_pr.Close()
			stdout_pr.Close()
			stderr_pr.Close()
			//			stdindest.Close()
		})
	}

	defer deffunc()

	stderr := io.TeeReader(cmd_stderr_pipe, stderr_pw)
	go func() {
		io.Copy(ptyErr, stderr) //err
		deffunc()
	}()

	stdout := io.TeeReader(cmd_stdoout_pipe, stdout_pw)
	go func() {
		io.Copy(ptyFile, stdout) //out,
		deffunc()
	}()

	if true { //duplicate stdin
		stdin := io.TeeReader(ptyFile, stdin_pw)
		go func() {
			io.Copy(cmd_stdin_pipe, stdin) //stdin
			deffunc()
		}()

		go func() {
			io.Copy(tee, stdin_pr)
			deffunc()
		}()

		go func() {
			io.Copy(tee, stdin_pr)
			deffunc()
		}()
	} else {
		go func() {
			io.Copy(cmd_stdin_pipe, ptyFile)
			deffunc()
		}()
	}

	//tee err, in , out

	go func() {
		io.Copy(tee, stdout_pr)
		deffunc()
	}()
	io.Copy(tee, stderr_pr)

	return
}

func TeeReadWriterCmd(cmd *exec.Cmd, ptyFile io.ReadWriter, ptyErr, tee io.Writer, closefunc func()) (err error) {
	cmd_stderr_pipe, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("cmd: StderrPipe: %v\n", err)
		return err
	}

	cmd_stdoout_pipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("cmd: StdoutPipe: %v\n", err)
		return err
	}

	cmd_stdin_pipe, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("cmd: StinPipe: %v\n", err)
		return err
	}
	go TeeReadWriter(cmd_stdin_pipe, cmd_stdoout_pipe, cmd_stderr_pipe, ptyFile, ptyErr, tee, closefunc)
	return err
}

func TeeReadWriterOsFile(f *os.File, ptyFile io.ReadWriter, ptyErr, tee io.Writer, closefunc func()) {
	TeeReadWriter(f, f, f, ptyFile, ptyErr, tee, closefunc)
	return
}

func CopyReadWriters(a, b io.ReadWriter, closefunc func()) {
	var once sync.Once
	//	if closefunc == nil {
	//		closefunc = func() {
	//			a.Close()
	//			b.Close()
	//		}
	//	}
	deffunc := func() {
		once.Do(func() {
			if closefunc != nil {
				closefunc()
			}
		})
	}
	go func() {
		io.Copy(a, b)
		deffunc()
	}()
	defer deffunc()

	io.Copy(b, a)
}

func GetExecPath() (pathexe string, err error) {
	pathexe, err = os.Executable()
	if err != nil {
		// log.Println("Cannot  get binary")
		return "", err
	}
	pathexe, err = filepath.EvalSymlinks(pathexe)
	if err != nil {
		// log.Println("Cannot  get binary")
		return "", err
	}
	return
}

func StringReverseString(s string) string {
	runes := []rune(s)

	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

func StringContainsI(a string, b string) bool {
	return strings.Contains(
		strings.ToLower(a),
		strings.ToLower(b),
	)
}

func StringGetIpv4(input string) string {
	if ipv4 := sregexp.New(Ipv4Regex).FindString(input); len(ipv4) != 0 { //not is xaddr, is deviceid
		return ipv4
	}
	return ""
}

func StringTrimLeftRightNewlineSpace(input string) string {
	input = strings.TrimSpace(input)
	input = strings.Trim(input, "\n")
	input = strings.Trim(input, "\r")
	return input
}

func SlideHasSubstringInStrings(s []string, b string) bool {
	for i := 0; i < len(s); i++ {
		//		fmt.Println(s[i], b)
		if StringContainsI(b, s[i]) {
			return true
		}
	}
	return false
}

func NewProgressBarTo(subject string, size int64, outPipe io.Writer) ProgressBar {
	return ProgressBar{outPipe, DEFAULT_FORMAT, subject, time.Now(), size}
}

func NewProgressBar(subject string, size int64) ProgressBar {
	return NewProgressBarTo(subject, size, os.Stdout)
}

func (pb ProgressBar) Update(tot int64) {
	percent := int64(0)
	if pb.Size > int64(0) {
		percent = (int64(100) * tot) / pb.Size
	}
	totTime := time.Now().Sub(pb.StartTime)
	spd := float64(float64(tot)/1024) / totTime.Seconds()
	//TODO put kb size into format string
	fmt.Fprintf(pb.Out, pb.Format, pb.Subject, percent, tot, spd, totTime)
}

func TimeNowUTC() string {
	//			2021-03-11 01:49:58.968944707 +0000 UTC
	tar := strings.Split(time.Now().UTC().String(), " ")
	return fmt.Sprintf("%s %s", tar[0], tar[1])
}

func TimeTrack(start time.Time, fname ...string) time.Duration {
	var name string
	elapsed := time.Since(start)

	// Skip this function, and fetch the PC and file for its parent.
	pc, _, _, _ := runtime.Caller(1)

	// Retrieve a function object this functions parent.
	funcObj := runtime.FuncForPC(pc)

	// Regex to extract just the function name (and not the module path).
	if len(fname) != 0 {
		name = fname[0]
	} else {
		runtimeFunc := regexp.MustCompile(`^.*\.(.*)$`)
		name = runtimeFunc.ReplaceAllString(funcObj.Name(), "$1")
	}

	log.Warn(fmt.Sprintf("TimeTrack %s took %s", name, elapsed))
	return elapsed
}

func StructGetFieldFromName(it *interface{}, fieldName string) interface{} {
	r := reflect.Indirect(reflect.ValueOf(it))
	f := r.FieldByName(fieldName)
	return f.Interface()
}

func StructGetFieldName(structPoint, fieldPinter interface{}) (name string) {

	val := reflect.ValueOf(structPoint).Elem()
	val2 := reflect.ValueOf(fieldPinter).Elem()

	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		if valueField.Addr().Interface() == val2.Addr().Interface() {
			return val.Type().Field(i).Name
		}
	}
	return
}

func CheckIsDone(ctx context.Context) bool {
	select {
	case _, ok := <-ctx.Done():
		if ok {
			return true
		}
		return true
	default:
		return false
	}
}
