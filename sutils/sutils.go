package sutils

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	//	"log"
	"net"
	"net/http"
	"net/smtp"

	"github.com/jordan-wright/email"
	"os"
	"path/filepath"
	//	"io"
	"io"
	//	"mime"
	"bufio"
	"bytes"
	//	"errors"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	//	"github.com/takama/daemon"
	//github.com/VividCortex/godaemon
	//		"github.com/sonnt85/gosutils/sexec"
	//	"github.com/sonnt85/gosutils/daemon"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rs/xid"
	. "github.com/sonnt85/gosutils/gogrep"
)

func GetHomeDir() (home string) {
	home, err := homedir.Dir()
	if err != nil {
		return home
	} else {
		return ""
	}
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
func IDGet() (id string) {
	var idname = ".idlock"
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
	data = data + string(os.PathSeparator)
	if len(path) == 0 {
		return data
	}
	return path + string(os.PathListSeparator) + data
	//	filepath.ListSeparator
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
	return Grepf("/proc/self/cgroup", "docker")
}

func IsPortAvailable(ip string, port int, timeout int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), time.Duration(timeout)*time.Second)

		if err, _ := err.(*net.OpError); err != nil {
			//ok && err.TimeOut()
			//			fmt.Printf("Timeout error: %s\n", err)
			return true
		}

		if err != nil {
			// Log or report the error here
			//			fmt.Printf("Error: %s\n", err)
			return true
		} else {
			defer conn.Close()
		}
		return false
	} else {
		conn.Close()
	}
	return true
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
		fmt.Println(err)
		return false
	}

	err = ioutil.WriteFile(destinationFile, input, 0644)
	if err != nil {
		fmt.Println("Error creating", destinationFile)
		fmt.Println(err)
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

func FileInsertStringAtLine(path, str string, index int) error {
	lines, err := File2lines(path)
	if err != nil {
		return err
	}

	fileContent := ""
	for i, line := range lines {
		if i == index {
			fileContent += str
		}
		fileContent += line
		fileContent += "\n"
	}

	return ioutil.WriteFile(path, []byte(fileContent), 0644)
}

func FindFileWithExt(pathS, ext string) (files []string) {
	ext = "." + ext
	if !PathIsExist(pathS) {
		return files
	}

	if PathIsFile(pathS) {
		if filepath.Ext(pathS) == ext {
			files = append(files, pathS)
		}
		return files
	}

	if !PathIsDir(pathS) {
		return files
	}

	filepath.Walk(pathS, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			if filepath.Ext(path) == ext {
				files = append(files, path)
			}
		}
		return nil
	})
	return files
}

func HTTPDownLoadUrl(urlpath, httpmethod, username, password string, insecure_flag bool) (byterets []byte) {
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
		return byterets
	}

	if len(username) != 0 {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
		// handle err
		return byterets
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	return bodyBytes
}

func GetMacAddr() ([]string, error) {
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
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err != nil {
		fmt.Println(err)
	} else {
		defer conn.Close()
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func NetGetMac(printFlag bool) (macadd string) {
	// get all the system's or local machine's network interfaces
	LanIP := GetOutboundIP()
	interfaces, _ := net.Interfaces()
	for _, interf := range interfaces {

		if addrs, err := interf.Addrs(); err == nil {
			for _, addr := range addrs {
				if strings.Contains(addr.String(), LanIP) {
					if printFlag {
						fmt.Printf("%s: %s\n", interf.Name, LanIP)
					}
					return interf.HardwareAddr.String()
				}
			}
		}
	}
	return ""
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
	//	fmt.Println(attachtype.Kind())
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
		//		fmt.Println(attachstr)
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

	if err := e.Send(smtphost, smtp.PlainAuth("", usermail, password, smtphostnoport)); err != nil {
		fmt.Printf("Can not send email: %s", err)
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

//func CPULoad() {
//	sexec.ExecCommand()
//}
