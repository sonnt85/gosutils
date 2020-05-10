# simplessh

simplessh provides a simple wrapper around the standard SSH library. Designed to be like net/http but for ssh.

## Install

    go get github.com/sonnt85/gosutils/simplessh

## Usage

### Server Example

```go
package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/sonnt85/gosutils/simplessh"

	"golang.org/x/crypto/ssh"
)

func main() {

	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key (./id_rsa)")
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

```

### Client Example

```go
package main

import (
	"log"

	"github.com/sonnt85/gosutils/simplessh"
	"golang.org/x/crypto/ssh"
)

func main() {
	config := &ssh.ClientConfig{
		User: "test",
		Auth: []ssh.AuthMethod{
			ssh.Password("test"),
		},
	}


  conn, err := simplessh.Dial("tcp", "localhost:2022", config)
  if err != nil {
  	log.Fatalf("unable to connect: %s", err)
  }
  defer conn.Close()

  err = conn.LocalForward("localhost:8000", "localhost:6060")
  if err != nil {
  	log.Fatalf("unable to forward local port: %s", err)
  }

}

```
