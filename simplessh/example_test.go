package simplessh_test

import (
	"fmt"
	"io/ioutil"

	"github.com/sonnt85/gosutils/simplessh"

	"golang.org/x/crypto/ssh"
	"os"
	"filepath"
)

func ExampleServer_ListenAndServe() {
	// Public key authentication is done by comparing
	// the public key of a received connection
	// with the entries in the authorized_keys file.
	authorizedKeysBytes, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "authorized_keys"))
	var authorizedPrivateKeysBytes []byte
	if err != nil {
		log.Fatalf("Failed to load authorized_keys, err: %v", err)
		authorizedKeysBytes, authorizedPrivateKeysBytes = simplessh.CreateKeyPairBytes()
	}

	authorizedKeysMap := map[string]bool{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			log.Fatal(err)
		}

		authorizedKeysMap[string(pubKey.Marshal())] = true
		authorizedKeysBytes = rest
	}
	s := simplessh.Server{Addr: ":2022"}

	privateBytes, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"))
	if err != nil {
		// Failed to load private key (./id_rsa)
		privateBytes = authorizedPrivateKeysBytes
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		// Failed to parse private key
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "test" && string(pass) == "test" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %s", c.User())
		},
		// Remove to disable public key auth.
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					// Record the public key used for authentication.
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}
	config.AddHostKey(private)

	s.Config = config

	if 1 {
		handler := simplessh.NewStandardSSHServerHandler()
		channelHandler := simplessh.NewChannelsMux()

		channelHandler.HandleChannel(simplessh.SessionRequest, simplessh.SessionHandler())
		handler.MultipleChannelsHandler = channelHandler

		s.Handler = handler
	} else {
		//pty
		simplessh.HandleChannel(simplessh.SessionRequest, simplessh.SessionHandler())
		//ssh -L 
		simplessh.HandleChannel(simplessh.DirectForwardRequest, simplessh.DirectPortForwardHandler())
		//ssh -R
		simplessh.HandleRequestFunc(simplessh.RemoteForwardRequest, simplessh.TCPIPForwardRequest)

	}
	s.ListenAndServe()
}

func ExampleListenAndServe() {
	config := &ssh.ServerConfig{}
	simplessh.HandleChannel(simplessh.SessionRequest, simplessh.SessionHandler())
	simplessh.HandleChannel(simplessh.DirectForwardRequest, simplessh.DirectPortForwardHandler())
	simplessh.HandleRequestFunc(simplessh.RemoteForwardRequest, simplessh.TCPIPForwardRequest)

	simplessh.ListenAndServe(":2022", config, nil)
}

func ExampleChannelsMux_HandleChannelFunc() {
	handler := simplessh.NewChannelsMux()

	testHandler := func(newChannel ssh.NewChannel, channel ssh.Channel, reqs <-chan *ssh.Request, sshConn ssh.Conn) {
		defer channel.Close()
		ssh.DiscardRequests(reqs)
	}

	handler.HandleChannelFunc("test", testHandler)

}
