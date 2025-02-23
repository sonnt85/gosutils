package simplessh

import (
	"bufio"
	"bytes"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	//	"os/exec"
	//	"strconv"
	//	"fmt"
)

// Client wraps an SSH Client
type Client struct {
	*ssh.Client
	config *ssh.ClientConfig
	addr   string
	//	Client *ssh.Client
}

func ClientAuthMethod(file string) (ssh.AuthMethod, error) {
	var buffer []byte
	if _, err := os.Stat(file); err == nil {
		buffer, err = os.ReadFile(file) //private key
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
		//		logger.Println(fmt.Sprintf("Cannot parse keypass: %s", keypass))
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
	c.Client, err = ssh.Dial("tcp", c.addr, c.config)
	if err == nil {
		go func() {
			c.Wait()
			c.Close()
		}()
	}
	return err
}

func (c *Client) Run(cmd string) (stdout, stderr string, err error) {
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
	session.Run(cmd)
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()
	return stdout, stderr, err
}

func (c *Client) Shell(cmd string) (err error) {
	//	modes := ssh.TerminalModes{
	//		ssh.ECHO:          0,     // disable echoing
	//		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
	//		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	//	}
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	//	out, err := exec.Command("stty", "size").Output()
	//	if err != nil {
	//		return err
	//	}
	//	s := strings.Split(string(out), " ")
	//	w, _ := strconv.Atoi(s[0])
	//	h, _ := strconv.Atoi(s[1])
	//	session.RequestPty("term", w, h, modes)
	//	var stdoutBuf,stderrBuf  bytes.Buffer
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin
	err = session.Run(cmd)
	//	stdout = stdoutBuf.String()
	//	stderr = stderrBuf.String()
	return nil
}

// LocalForward performs a port forwarding over the ssh connection - ssh -L. Client will bind to the local address, and will tunnel those requests to host addr
func (c *Client) LocalForward(laddr, raddr *net.TCPAddr) error {

	ln, err := net.ListenTCP("tcp", laddr) //tie to the client connection
	if err != nil {
		println(err.Error())
		return err
	}
	//	logger.Println("Listening on address: ", ln.Addr().String())

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
func (c *Client) RemoteForward(remote, local string) error {
	ln, err := c.Listen("tcp", remote)
	if err != nil {
		return err
	}
	ln.Addr()

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

				close := func() {
					conn.Close()
					conn2.Close()

				}

				go CopyReadWriters(conn, conn2, close)

			}

		}
	}()

	c.Wait()
	ln.Close()
	quit <- true

	return nil
}

// HandleOpenChannel requests that the remote end accept a channel request and if accepted,
// passes the newly opened channel and requests to the provided handler
func (c *Client) HandleOpenChannel(channelName string, handler ChannelMultipleRequestsHandler, data ...byte) error {
	ch, reqs, err := c.OpenChannel(channelName, data)
	if err != nil {
		return err
	}
	handler.HandleMultipleRequests(reqs, c.Conn, channelName, ch)
	return nil
}

// HandleOpenChannelFunc requests that the remote end accept a channel request and if accepted,
// passes the newly opened channel and requests to the provided handler function
func (c *Client) HandleOpenChannelFunc(channelName string, handler ChannelMultipleRequestsHandlerFunc, data ...byte) error {

	return c.HandleOpenChannel(channelName, ChannelMultipleRequestsHandlerFunc(handler), data...)
}
