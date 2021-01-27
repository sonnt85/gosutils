package sutils

import (
	log "github.com/sirupsen/logrus"

	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io/ioutil"

	//	log "github.com/sirupsen/logrus"
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
	"path"
	"reflect"

	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	//"github.com/takama/daemon"
	//github.com/VividCortex/godaemon
	//"github.com/sonnt85/gosutils/sexec"
	//"github.com/sonnt85/gosutils/daemon"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rs/xid"
	"github.com/sonnt85/gosutils/gogrep"

	regexputil "github.com/sonnt85/gosutils/regexp"

	//	. "github.com/sonnt85/gosutils/gogrep"
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
func FileAddOrUpdate(pathfile, contents, grepstring, sedexpress string) (err error) {
	if PathIsFile(pathfile) {
		if !gogrep.GrepF(pathfile, grepstring) {
			_, _, _ = gosed.Sed(sedexpress, pathfile) //update
			if !gogrep.GrepF(pathfile, grepstring) {  //not found
				return FileInsertStringAtLine(pathfile, contents, 1)
			}
		}
		//not change
	} else {
		return ioutil.WriteFile(pathfile, []byte(contents), os.FileMode(0644))
	}

	return nil
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
	home, err := homedir.Dir()
	if err == nil {
		return home
	} else {
		return ""
	}
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
	return nBytes, err
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

func GetDescription() (des string) {
	des = os.Getenv("CINFO")
	if len(des) != 0 {
		return des
	}

	var iddes = ".iddes"
	var listdir = []string{"/mnt/hostvolume/"}
	HOME, err := homedir.Dir()
	if err == nil {
		listdir = append(listdir, HOME)
	}
	for _, dir := range listdir {
		if PathIsDir(dir) {
			filename := path.Join(dir, iddes)
			if len(des) != 0 {
				if err := ioutil.WriteFile(filename, []byte(des), os.FileMode(0644)); err == nil {
					return
				}
				continue
			}
			if data, err := ioutil.ReadFile(filename); err == nil {
				return string(data)
			}
		}
	}
	return ""
}

func IDGet() (id string) {
	var idname = ".idlock"
	var iddes = ".iddes"
	var listdir = []string{"/mnt/hostvolume/"}

	HOME, err := homedir.Dir()
	if err == nil {
		listdir = append(listdir, HOME)
	}

	if len(os.Getenv("APPID")) != 0 {
		id = os.Getenv("APPID")
		//		id = xid.New().String()
	}

	getIDFromListDir := func(cmd byte) (retid, filename string) {
		for _, dir := range listdir {
			if PathIsDir(dir) {
				filename = path.Join(dir, idname)
				if cmd == 0 { //find idlock
					if PathIsFile(filename) {
						if data, err := ioutil.ReadFile(filename); err == nil {
							if _, err := xid.FromString(string(data)); err == nil { //check conten is valid
								retid = string(data)
								return
							}
						}
					}
				} else if cmd == 1 { // write first id if allow
					if err := ioutil.WriteFile(filename, []byte(id), os.FileMode(0644)); err == nil {
						retid = id
						iddespath := path.Join(dir, iddes)
						if err := ioutil.WriteFile(iddespath, []byte(os.Getenv("CINFO")), os.FileMode(0644)); err == nil {

						}
						return
					}
				}
			}
		}
		return
	}

	if len(id) != 0 { //id from env
		if id1, filpath := getIDFromListDir(1); len(id1) != 0 && len(filpath) != 0 { //write id to file
			return id1
		}
		return id
	} else {
		if id2, filpath := getIDFromListDir(0); len(id2) != 0 && len(filpath) != 0 { //read id from file
			return id2
		} else { // no id in files
			id = xid.New().String()
			if id3, filpath := getIDFromListDir(1); len(id3) != 0 && len(filpath) != 0 { //write id from file
				return id3
			}
			return id
		}
	}
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

func PATHHasFile(filepath, PATH string) bool {
	execbasename := path.Base(filepath)
	for _, val := range strings.Split(PATH, string(os.PathListSeparator)) {
		if PathIsFile(path.Join(val, execbasename)) {
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

func IsContainer() bool {
	return gogrep.GrepF("/proc/self/cgroup", "docker") || gogrep.GrepF("/proc/self/cgroup", "lxc")
}

func IsPortOpen(addr string, port int, proto string) bool {
	if len(proto) == 0 {
		proto = "tcp"
	}
	conn, err := net.DialTimeout(proto, fmt.Sprintf("%s:%d", addr, port), time.Microsecond*500)
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

func createSha1(data []byte) string {
	h := sha1.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func str2Sha1(data string) string {
	h := sha1.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func TokenCreate(key int) string {
	if key == 0 {
		key = 1985
	}
	nowtimestam := time.Now().Unix() + int64(key)
	return createSha1([]byte(string(nowtimestam)))
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

func FileInsertStringAtLine(filePath, str string, index int) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	str = str + "\n" //add newline
	scanner := bufio.NewScanner(f)
	lines := ""
	linenum := 0

	for scanner.Scan() {
		linenum = linenum + 1
		lines = lines + scanner.Text() + "\n"
		if linenum == index {
			lines = lines + str
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if linenum == 0 && index == 1 {
		return ioutil.WriteFile(filePath, []byte(str), 0644)
	}

	return ioutil.WriteFile(filePath, []byte(lines), 0644)
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

	for i, _ := range files {
		regx := regexputil.New("^" + pathR)
		regx.Regexp()
		//		files[i] = pathS + strings.TrimLeft(files[i], pathR)
		files[i] = regx.ReplaceAllString(files[i], pathS)
		//		files[i] = strings.Replace(files[i], pathR, pathS, 1)
	}
	return files
}

func FindFileWithExt1(root, pattern string) []string {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
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

	filepath.Walk(pathS, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func HTTPDownLoadUrl(urlpath, httpmethod, username, password string, insecure_flag bool) (byterets []byte, err error) {
	byterets = make([]byte, 0, 0)
	// Generated reqby curl-to-Go: https://mholt.github.io/curl-to-go

	// TODO: This is insecure; use only in dev environments.
	client := &http.Client{}
	if insecure_flag {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
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

func NetGetMacs() ([]string, error) {
	ifas, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var as []string
	for _, ifa := range ifas {
		a := ifa.HardwareAddr.String()
		if a != "" {
			as = append(as, a)
		}
	}
	return as, nil
}

func GetOutboundIP() string {
	conn, err := net.DialTimeout("udp", "1.1.1.1:80", time.Second)
	if err != nil {
		log.Println(err)
		return ""
	} else {
		defer conn.Close()
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func NetGetCIDR() (string, error) {
	ret := NetGetDefaultInterface(2)
	if ret != "" {
		return ret, nil
	} else {
		return "", nil
	}
}

//infotype 0 interface name, 1 macaddr, 2 cird, >2 lanip]
func NetGetDefaultInterface(infotype int) (info string) {
	// get all the system's or local machine's network interfaces
	LanIP := GetOutboundIP()
	interfaces, _ := net.Interfaces()
	for _, interf := range interfaces {

		if addrs, err := interf.Addrs(); err == nil {
			for _, addr := range addrs {
				if strings.Contains(addr.String(), LanIP) {
					if infotype == 0 {
						return interf.Name
					} else if infotype == 1 {
						return interf.HardwareAddr.String()
					} else if infotype == 2 {
						return addr.String()
					} else {
						return LanIP
					}
				}
			}
		}
	}
	return ""
}

func NetGetMac() (macadd string) {
	return NetGetDefaultInterface(1)
}

func NetIsIpv4(ip net.IP) bool {
	if strings.Contains(ip.String(), ".") { //firtst ip4
		return true
	}
	return false
}

func NetTCPClientSend(servAddr string, dataSend []byte) (retbytes []byte, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		println("ResolveTCPAddr failed:", err.Error())
		return retbytes, err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		println("Dial failed:", err.Error())
		return retbytes, err
	}

	defer conn.Close()

	_, err = conn.Write(dataSend)

	if err != nil {
		println("Write to server failed:", err.Error())
		return retbytes, err
	}
	//	conn.Write(io.EOF)
	reply := make([]byte, 1024)

	n, err := conn.Read(reply)
	if err != nil {
		println("Write to server failed:", err.Error())
	}
	return reply[:n], err
}

func Unique(intSlice []interface{}) interface{} {
	var list []interface{}
	keys := make(map[interface{}]bool)
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
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

func GmailSend(usermail, password, from, subject string, to []string, msg []byte, attach interface{}, filename string) (err error) {
	return EmailSend("smtp.gmail.com:587", usermail, password, from, subject, to, msg, attach, filename)
}

func EmailSend(smtphost, usermail, password, from, subject string, to []string, msg []byte, attach interface{}, filename string) (err error) {
	if len(usermail) == 0 && len(password) == 0 {
		usermail = "iotecloud@gmail.com"
		password = "iot123cloud"
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

type SmartSleepMs struct {
	nextSleep int64
	Step, Max int64
}

func NewSmartSleep(Step, Max int64) *SmartSleepMs {
	return &SmartSleepMs{
		Max:       Max,
		Step:      Step,
		nextSleep: 0,
	}
}

func (ss *SmartSleepMs) NextSleep() {
	defer func() {
		ss.nextSleep += ss.Step
	}()

	if ss.nextSleep >= ss.Max {
		ss.nextSleep = 0
	}
	time.Sleep((time.Duration(ss.nextSleep) * time.Millisecond))
}

type SmartTimerMs struct {
	nextSleep int64
	Step, Max int64
	timer     *time.Timer
}

func NewSmartTimerMs(Step, Max int64) *SmartTimerMs {
	return &SmartTimerMs{
		Max:       Max,
		Step:      Step,
		nextSleep: 0,
		timer:     time.NewTimer(0),
	}
}

func (ss *SmartTimerMs) GetChannel() <-chan time.Time {
	return ss.timer.C
}

func (ss *SmartTimerMs) NextDuration() {
	defer func() {
		ss.nextSleep += ss.Step
	}()

	if ss.nextSleep >= ss.Max {
		ss.nextSleep = 0
	}
	ss.timer.Reset((time.Duration(ss.nextSleep) * time.Millisecond))
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
		log.Println("Cannot  get binary")
		return "", err
	}
	pathexe, err = filepath.EvalSymlinks(pathexe)
	if err != nil {
		log.Println("Cannot  get binary")
		return "", err
	}
	return
}

////progressbar
const DEFAULT_FORMAT = "\r%s   %3d %%  %d kb %0.2f kb/s %v      "

//const DEFAULT_FORMAT = "\r%s\t\t%3d %%\t%d kb\t%0.2f kb/s\t%v      "

type ProgressBar struct {
	Out       io.Writer
	Format    string
	Subject   string
	StartTime time.Time
	Size      int64
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

// Reflect if an interface is either a struct or a pointer to a struct
// and has the defined member method. If error is nil, it means
// the MethodName is accessible with reflect.
func ReflectStructMethod(Iface interface{}, MethodName string) error {
	ValueIface := reflect.ValueOf(Iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(Iface))
	}

	// Get the method by name
	Method := ValueIface.MethodByName(MethodName)
	if !Method.IsValid() {
		return fmt.Errorf("Couldn't find method `%s` in interface `%s`, is it Exported?", MethodName, ValueIface.Type())
	}
	return nil
}

// Reflect if an interface is either a struct or a pointer to a struct
// and has the defined member field, if error is nil, the given
// FieldName exists and is accessible with reflect.
func ReflectStructField(Iface interface{}, FieldName string) error {
	ValueIface := reflect.ValueOf(Iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface's Type, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(Iface))
	}

	// 'dereference' with Elem() and get the field by name
	Field := ValueIface.Elem().FieldByName(FieldName)
	if !Field.IsValid() {
		return fmt.Errorf("Interface `%s` does not have the field `%s`", ValueIface.Type(), FieldName)
	}
	return nil
}

//func CPULoad() {
//	sexec.ExecCommand()
//}
