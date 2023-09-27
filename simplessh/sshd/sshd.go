package main

import (
	"fmt"
	"log"
	"os"

	"github.com/sonnt85/gosutils/simplessh"

	"golang.org/x/crypto/ssh"
)

func main() {

	privateBytes, err := os.ReadFile("id_rsa")
	if err != nil {
		logl("Failed to load private key (./id_rsa)")
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key")
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "test" && string(pass) == "test" {
				log.Printf("User logged in: %s", c.User())
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %s", c.User())
		},
	}
	config.AddHostKey(private)

	simplessh.HandleChannel(simplessh.SessionRequest, simplessh.SessionHandler())
	simplessh.HandleChannel(simplessh.DirectForwardRequest, simplessh.DirectPortForwardHandler())
	simplessh.HandleRequestFunc(simplessh.RemoteForwardRequest, simplessh.TCPIPForwardRequest)

	simplessh.ListenAndServe(":2022", config, nil)
}
