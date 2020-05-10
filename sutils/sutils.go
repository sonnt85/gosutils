package sutils

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/smtp"

	"github.com/jordan-wright/email"
	"os"
	"strconv"
	"strings"
	"time"
	//		"github.com/sonnt85/gosutils/sexec"
)

func FileCreateTempFile(rootdir, filename string) {
	file, err := ioutil.TempFile(rootdir, filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.Remove(file.Name())

	fmt.Println(file.Name())
}

func IsPortAvailable(ip string, port int, timeout int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), time.Duration(timeout)*time.Second)
		defer conn.Close()

		if err, _ := err.(*net.OpError); err != nil {
			//ok && err.TimeOut()
			fmt.Printf("Timeout error: %s\n", err)
			return false
		}

		if err != nil {
			// Log or report the error here
			fmt.Printf("Error: %s\n", err)
			return false
		}
		return false
	}
	conn.Close()
	return true
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

func NetGetMac() (macadd string) {
	// get all the system's or local machine's network interfaces
	LanIP := GetOutboundIP()
	interfaces, _ := net.Interfaces()
	for _, interf := range interfaces {

		if addrs, err := interf.Addrs(); err == nil {
			for _, addr := range addrs {
				if strings.Contains(addr.String(), LanIP) {
					fmt.Printf("%s: %s\n", interf.Name, LanIP)
					return interf.HardwareAddr.String()
				}
			}
		}
	}
	return ""
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

func GmailSend(usermail, password, from, subject string, to []string, msg []byte) {
	if len(usermail) == 0 && len(password) == 0 {
		usermail = "iotecloud@gmail.com"
		password = "iot123cloud"
	}
	e := email.NewEmail()
	e.From = from //"Jordan Wright <test@gmail.com>"
	e.To = to     //[]string{"test@example.com"}
	//	e.Bcc = []string{"test_bcc@example.com"}
	//	e.Cc = []string{"test_cc@example.com"}
	e.Subject = subject //"Awesome Subject"
	e.Text = msg        //[]byte("Text Body is, of course, supported!")
	//	e.HTML = []byte("<h1>Fancy HTML is supported, too!</h1>")
	if err := e.Send("smtp.gmail.com:587", smtp.PlainAuth("", usermail, password, "smtp.gmail.com")); err != nil {
		fmt.Printf("Can not send email: %s", err)
	}
}

//func CPULoad() {
//	sexec.ExecCommand()
//}
