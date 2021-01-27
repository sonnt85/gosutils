package sshclient

import (
	log "github.com/sirupsen/logrus"

	"bufio"
	"bytes"
	//	"fmt"
	//	terminaldimensions "github.com/sonnt85/gosutils/terminaldimensions"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"io"
	"io/ioutil"
	"net"
	"os"
	//	"os/exec"
	"path/filepath"
	//	"regexp"
	//	"strconv"

	//	"context"
	"fmt"
	"strings"
	"sync"
	//	"time"
)

// Client wraps an SSH Client
type Client struct {
	*ssh.Client
	config *ssh.ClientConfig
	Addr   string
}

func CopyReadWriters(a, b io.ReadWriter, close func()) {
	var once sync.Once
	go func() {
		io.Copy(a, b)
		once.Do(close)
	}()

	go func() {
		io.Copy(b, a)
		once.Do(close)
	}()
}

func ClientAuthMethod(file string) (ssh.AuthMethod, error) {
	var buffer []byte
	if _, err := os.Stat(file); err == nil {
		buffer, err = ioutil.ReadFile(file) //private key
		if err != nil {
			//			logger.Println(fmt.Sprintf("Cannot read SSH public key file %s, use password", file))
			return nil, err
		}
	} else {
		if len(file) > 50 { //private key
			buffer = []byte(file)
		} else { //password
			return ssh.Password(file), nil
		}
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		//		logger.Println(fmt.Sprintf("Cannot parse SSH public key file %s", file))
		return nil, err
	}
	return ssh.PublicKeys(key), nil
}

func GetHostKey(host string) ssh.PublicKey {
	// parse OpenSSH known_hosts file
	// ssh or use ssh-keyscan to get initial key
	file, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hostKey ssh.PublicKey
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			continue
		}
		if strings.Contains(fields[0], host) {
			var err error
			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				return nil
			}
			break
		}
	}

	if hostKey == nil {
		return nil
	}

	return hostKey
}

// NewClient returns a new SSH Client. , c ssh.Conn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request
func NewClient(user, addr, keypass string) *Client {
	auMethod, err := ClientAuthMethod(keypass)
	if err != nil {
		log.Println(fmt.Sprintf("Cannot parse keypass: %s", keypass))
		return nil
	}
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			auMethod,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	//	client := ssh.NewClient(c, chans, reqs)

	return &Client{nil, config, addr}
}

// Dial starts an ssh connection to the provided server
func Dial(network, addr string, config *ssh.ClientConfig) (*Client, error) {
	c, err := ssh.Dial(network, addr, config)
	return &Client{c, config, addr}, err
}

func (c *Client) Dial() error {
	var err error
	c.Client, err = ssh.Dial("tcp", c.Addr, c.config)
	if err == nil {
		go func() {
			c.Wait()
			c.Close()
		}()
	}
	return err
}

func (c *Client) Run(cmd string) (stdout, stderr []byte, err error) {
	//	modes := ssh.TerminalModes{
	//		ssh.ECHO:          0,     // disable echoing
	//		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
	//		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	//	}
	session, err := c.NewSession()
	if err != nil {
		return stdout, stderr, err
	}
	defer session.Close()
	//	session.RequestPty("term", 80, 40, modes)
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf
	err = session.Run(cmd)
	stdout = stdoutBuf.Bytes()
	stderr = stderrBuf.Bytes()
	return stdout, stderr, err
}

func (c *Client) Shell() (err error) {
	return c.RunCommand("")
}

func (c *Client) SCPFromRemote(sourcePath, destinationPath string, ignErr, IsQuiet bool) (err error) {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer func() {
		//		done <- true
		session.Close()
	}()
	//	type SecureCopier struct {
	//	IsRecursive bool
	//	IsQuiet   bool
	//	IsVerbose bool
	//	inPipe  io.Reader
	//	outPipe io.Writer
	//	errPipe io.Writer
	//	srcFile string
	//	dstFile string
	//}

	scp := &SecureCopier{
		srcFile:     sourcePath,
		dstFile:     destinationPath,
		IsQuiet:     IsQuiet,
		ignErr:      ignErr,
		IsRecursive: true,
		IsVerbose:   !IsQuiet,

		//		inPipe:      os.Stdin,
		//		outPipe:     os.Stdout,
		//		errPipe:     os.Stdout,
	}

	return scpFromRemote(scp, session)
}

func (c *Client) SCPToRemote(sourcePath, destinationPath string, ignErr, IsQuiet bool) (err error) {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer func() {
		//		done <- true
		session.Close()
	}()
	//	type SecureCopier struct {
	//	IsRecursive bool
	//	IsQuiet   bool
	//	IsVerbose bool
	//	inPipe  io.Reader
	//	outPipe io.Writer
	//	errPipe io.Writer
	//	srcFile string
	//	dstFile string
	//}

	scp := &SecureCopier{
		srcFile:     sourcePath,
		dstFile:     destinationPath,
		IsQuiet:     IsQuiet,
		ignErr:      ignErr,
		IsRecursive: true,
		IsVerbose:   !IsQuiet,
		//		inPipe:      os.Stdin,
		//		outPipe:     os.Stdout,
		//		errPipe:     os.Stdout,
	}
	return scpToRemote(scp, session)
}

//func (c *Client) Run1(cmd string) (stdout, stderr string, err error) {
//	c.RunCommand(cmd)
//}

func (c *Client) RunCommand(cmd string) (err error) {
	done := make(chan bool)
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer func() {
		done <- true
		session.Close()
	}()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	fileDescriptor := int(os.Stdin.Fd())
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echoing
		ssh.ECHOCTL:       0,     //Ignore CR on input.
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	//	//vt100  xterm "xterm-256color"
	tertype := os.Getenv("TERM")
	if tertype == "" {
		tertype = "xterm-256color"
	}
	if terminal.IsTerminal(fileDescriptor) {

		originalState, err := terminal.MakeRaw(fileDescriptor)
		if err != nil {
			return err
		}

		defer terminal.Restore(fileDescriptor, originalState)

		termWidth, termHeight, err := terminal.GetSize(fileDescriptor)
		if err != nil {
			return err
		}

		err = session.RequestPty(tertype, termHeight, termWidth, modes)
		//		log.Printf("%dx%d\n", termWidth, termHeight)
		if err != nil {
			return err
		}
	}
	if cmd == "" {
		if err = session.Shell(); err != nil {
			return err
		}
		return session.Wait()
	} else {
		err = session.Run(cmd)
		if err != nil {
			return err
		}
	}

	return err
}

// LocalForward performs a port forwarding over the ssh connection - ssh -L. Client will bind to the local address, and will tunnel those requests to host addr
func (c *Client) LocalForward(retport chan int, laddrstr, raddrstr string) error {
	doneRetPort := false

	defer func() {
		if !doneRetPort {
			retport <- 0
		}
	}()

	if laddrstr == "0" {
		laddrstr = "localhost:0"
	}

	laddr, err := net.ResolveTCPAddr("tcp", laddrstr)
	if err != nil {
		println(err.Error())
		return err
	}

	raddr, err := net.ResolveTCPAddr("tcp", raddrstr)
	if err != nil {
		println(err.Error())
		return err
	}
	ln, err := net.ListenTCP("tcp", laddr) //tie to the client connection
	if err != nil {
		println(err.Error())
		return err
	}

	go func() {
		select {
		case retport <- ln.Addr().(*net.TCPAddr).Port:
			doneRetPort = true
			return
		}
	}()
	//	log.Println("[LocalForward] Listening on address: ", ln.Addr().String())

	quit := make(chan bool)

	go func() { // Handle incoming connections on this new listener
		for {
			select {
			case <-quit:

				return
			default:
				conn, err := ln.Accept()
				if err != nil { // Unable to accept new connection - listener likely closed
					continue
				}
				go func(conn net.Conn) {
					conn2, err := c.DialTCP("tcp", laddr, raddr)

					if err != nil {
						return
					}
					go func(conn, conn2 net.Conn) {

						close := func() {
							conn.Close()
							conn2.Close()
						}

						go CopyReadWriters(conn, conn2, close)

					}(conn, conn2)

				}(conn)
			}

		}
	}()

	c.Wait()

	ln.Close()
	quit <- true

	return nil
}

// RemoteForward forwards a remote port - ssh -R
func (c *Client) RemoteForward(retport chan int, remote, local string) error {
	doneRetPort := false

	defer func() {
		if !doneRetPort {
			retport <- 0
		}
	}()

	if remote == "0" {
		remote = "localhost:0"
	}
	ln, err := c.Listen("tcp", remote)
	if err != nil {
		println(err.Error())
		return err
	}

	go func() {
		select {
		case retport <- ln.Addr().(*net.TCPAddr).Port:
			// strings.Split(ln.Addr().String(), ":")[1]:
			doneRetPort = true
			return
		}
	}()
	//	log.Println("[Remote forward] Listening on address: ", ln.Addr().String())

	quit := make(chan bool)

	go func() { // Handle incoming connections on this new listener
		for {
			select {
			case <-quit:

				return
			default:
				conn, err := ln.Accept()
				if err != nil { // Unable to accept new connection - listener likely closed
					continue
				}

				conn2, err := net.Dial("tcp", local)
				if err != nil {
					continue
				}

				closefunc := func() {
					conn.Close()
					conn2.Close()

				}

				go CopyReadWriters(conn, conn2, closefunc)

			}

		}
	}()

	c.Wait()
	ln.Close()
	quit <- true

	return nil
}
