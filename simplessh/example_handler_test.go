package simplessh_test

import (
	"github.com/sonnt85/gosutils/simplessh"

	"golang.org/x/crypto/ssh"
)

type testHandler struct{}

func (testHandler) HandleChannel(nCh ssh.NewChannel, ch ssh.Channel, reqs <-chan *ssh.Request, conn ssh.Conn) {
	defer ch.Close()
	// Do something
}

func ExampleChannelsMux_HandleChannel() {
	handler := simplessh.NewChannelsMux()

	handler.HandleChannel("test", testHandler{})

	test2Handler := func(newChannel ssh.NewChannel, channel ssh.Channel, reqs <-chan *ssh.Request, sshConn ssh.Conn) {
		defer channel.Close()
		ssh.DiscardRequests(reqs)
	}

	handler.HandleChannelFunc("test2", test2Handler)

	handler.HandleChannel("anotherTest2", simplessh.ChannelHandlerFunc(test2Handler))
}
