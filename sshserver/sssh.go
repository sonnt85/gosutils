// A small SSH daemon providing bash sessions
//
// Server:
// cd my/new/dir/
// #generate server keypair
// ssh-keygen -t rsa
// go get -v .
// go run sshd.go
//
// Client:
// ssh foo@localhost -p 2200 #pass=bar

package main

import (
	"encoding/binary"
	"fmt"
	"io"
//	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"github.com/kr/pty"
	"golang.org/x/crypto/ssh"
//	"os"
)

func main() {

	// In the latest version of crypto/ssh (after Go 1.3), the SSH server type has been removed
	// in favour of an SSH connection type. A ssh.ServerConn is created by passing an existing
	// net.Conn and a ssh.ServerConfig to ssh.NewServerConn, in effect, upgrading the net.Conn
	// into an ssh.ServerConn

	config := &ssh.ServerConfig{
		//Define a function to run when a client attempts a password login
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Should use constant-time compare (or better, salt+hash) in a production setting.
			if c.User() == "foo" && string(pass) == "bar" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
		// You may also explicitly allow anonymous client authentication, though anon bash
		// sessions may not be a wise idea
		// NoClientAuth: true,
	}

	// You can generate a keypair with 'ssh-keygen -t rsa'
//	home, _ := os.UserHomeDir()
//	privateBytes, err := ioutil.ReadFile(home  + "/.ssh/id_rsa")
	privateBytes := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpgIBAAKCAQEAsJeD1Y1It9S7eO3S3PpVT0ZN88zeej3eCaHLiY8h2H664KPb
tnWhnkdijYYC30SgLGhPHIrMzLSsFWE0tncvqmbV0FMkUv5mJB22KxxG+MFsLQhC
hHNobgZQpz3xo3Y/HvBk6NWACh2mnA89D0Xh9yP33gonLAMdnEUF1gXgaB+iqsnc
naEAslDwR/tLnidvFzn5PF1LymYAETNJ0nDcdzF8CCt34sHOlQmaq/9h9JOh1TOu
xCjjULuMWp0nNAPoyA4LpAdlrxMJbgFLNUXqgqdnoZHQRI5XrdClyv6qm6kspuyx
j7mYG9v6n6Nwk22hIlfFuqWjXMIJyTFAzG8YWQIDAQABAoIBAQCiIrr8e7fkcQGf
ylvsYDuriZVQ3yz1d5BBr7e9GRmuOM1EK64zHFXDiS9HWV+RtuSJYUwhnJ7k5I2L
I7DORygQgFKX735OZR1K06zKcDAJfS3hOtA34+5h9pJeu1T9DDhwI6/CxyPEJe0v
JB6fwz3xN6kAyLmmg0XQkN8G3mZnsf1FjbuUWSuYkfQeCtzX7IU9g0mjfiqUtcje
kfwgmbGFuj7iabwWA+e2UiEAeKG9MSjg8kXcZZApbDkfjJwqmmPwtwb5r6n4Rydd
pCMCPD9o1SXeyljGRHvd2aMEDIfpkVZJAHryPX5/dY/gX25U9gNDV6YTiHX3lqxe
O4pOak3BAoGBAN6BzaBsPHfpk+C5T24n2WX4Zd+lC6MaihaEMGNqYTz1aieXioad
yPRTpUg7l7+njpZovC9fh1JVltxOnRtFuxl22EynCX67MfjwMpeGwSvuV5zjhIoA
eRdw+dqat6lQFmacjiWxqmFsvRFs3U40xuVbzYJSa0XqL57QWsommtHdAoGBAMss
ZKKQTFol0dqL75EkX/wKwx6mjQIr+AlA7eNzjq/C8Os/52LHnk9hSuZzFjGrM4uy
Dnfk1EtUxl0epPfWBebW+ZJ3aD+U/+ta44vIHdjrHhaXYuSuy/rYh3yXw9kxMLeZ
caQBFJIzwgNdFVdLLm7iC62i4Lm34g/Nqezljv6tAoGBAIuGyfK27JQlHF3m1jA1
PNX8laVQUaPNmJnV+qHcq20WV6LMHEmd182eRh6tf9LmtzsKIjdyp+CxWxB7G3lm
mJS3OZuXgxS9PfDkblUmYyuxIa933DzNXyGb7pFuQ40gc2uU8G4iorzE+ypaIcxQ
vAhHMO9vz2TgHUxxSv1Ih/zhAoGBALx5xjF4IxxNkUtoHSlL0S8C3NcGMjEdkM8k
yIoDnQ43jT7u3TupapbA7raxdJlG9F5XI0zdnoLzdcDUuLygcoEeVA8nbjHtiytN
+WCml+mu0w6qCTeTX+6oB6fxMeG93C+1zNITnn2yPfzY0P9V4xFB6Qt+2XHvv2ph
o4z7t5dRAoGBAJ9M1SkT7/0NazGmfV1qqkuJATAGMaqKRpIYmI1mTPaLATG1WU99
5v4uxdGlSE0/fQiDNFUwYn7oQpQ60sB7jXRaW564KEUF/z4QFaOnwPT6cYNoQP5V
0Rs9NUU0KfJbejL+YA/irToklJcRsuPi5DAlAaDtnd9kkK+aXmtRgpgR
-----END RSA PRIVATE KEY-----`)
//	if err != nil {
//		log.Fatal("Failed to load private key (./id_rsa)")
//	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key")
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:2200")
	if err != nil {
		log.Fatalf("Failed to listen on 2200 (%s)", err)
	}

	// Accept all connections
	log.Print("Listening on 2200...")
	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection (%s)", err)
			continue
		}
		// Before use, a handshake must be performed on the incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			log.Printf("Failed to handshake (%s)", err)
			continue
		}

		log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
		// Discard all global out-of-band Requests
		go ssh.DiscardRequests(reqs)
		// Accept all channels
		go handleChannels(chans)
	}
}

func handleChannels(chans <-chan ssh.NewChannel) {
	// Service the incoming Channel channel in go routine
	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}

func handleChannel(newChannel ssh.NewChannel) {
	// Since we're handling a shell, we expect a
	// channel type of "session". The also describes
	// "x11", "direct-tcpip" and "forwarded-tcpip"
	// channel types.
	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	// At this point, we have the opportunity to reject the client's
	// request for another logical connection
	connection, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Could not accept channel (%s)", err)
		return
	}

	// Fire up bash for this session
	bash := exec.Command("bash")

	// Prepare teardown function
	close := func() {
		connection.Close()
		_, err := bash.Process.Wait()
		if err != nil {
			log.Printf("Failed to exit bash (%s)", err)
		}
		log.Printf("Session closed")
	}

	// Allocate a terminal for this channel
	log.Print("Creating pty...")
	bashf, err := pty.Start(bash)
	if err != nil {
		log.Printf("Could not start pty (%s)", err)
		close()
		return
	}

	//pipe session to bash and visa-versa
	var once sync.Once
	go func() {
		io.Copy(connection, bashf)
		once.Do(close)
	}()
	go func() {
		io.Copy(bashf, connection)
		once.Do(close)
	}()

	// Sessions have out-of-band requests such as "shell", "pty-req" and "env"
	go func() {
		for req := range requests {
			switch req.Type {
			case "shell":
				// We only accept the default shell
				// (i.e. no command in the Payload)
				if len(req.Payload) == 0 {
					req.Reply(true, nil)
				}
			case "pty-req":
				termLen := req.Payload[3]
				w, h := parseDims(req.Payload[termLen+4:])
				SetWinsize(bashf.Fd(), w, h)
				// Responding true (OK) here will let the client
				// know we have a pty ready for input
				req.Reply(true, nil)
			case "window-change":
				w, h := parseDims(req.Payload)
				SetWinsize(bashf.Fd(), w, h)
			}
		}
	}()
}

// =======================

// parseDims extracts terminal dimensions (width x height) from the provided buffer.
func parseDims(b []byte) (uint32, uint32) {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return w, h
}

// ======================

// Winsize stores the Height and Width of a terminal.
type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16 // unused
	y      uint16 // unused
}

// SetWinsize sets the size of the given pty.
func SetWinsize(fd uintptr, w, h uint32) {
	ws := &Winsize{Width: uint16(w), Height: uint16(h)}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}

// Borrowed from https://github.com/creack/termios/blob/master/win/win.go
