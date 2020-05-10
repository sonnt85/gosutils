package simplessh_test

import (
	"fmt"
	"io/ioutil"

	"github.com/sonnt85/gosutils/simplessh"

	"golang.org/x/crypto/ssh"
)

func ExampleDirectPortForwardChannel() {
	s := simplessh.Server{Addr: ":2022"}

	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		// Failed to load private key (./id_rsa)
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
	}
	config.AddHostKey(private)

	s.Config = config

	handler := simplessh.NewStandardSSHServerHandler()
	channelHandler := simplessh.NewChannelsMux()

	channelHandler.HandleChannel(simplessh.DirectForwardRequest, simplessh.DirectPortForwardHandler())
	handler.MultipleChannelsHandler = channelHandler

	s.Handler = handler

	s.ListenAndServe()
}
