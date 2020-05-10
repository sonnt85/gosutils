package simplessh

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sync"

	"golang.org/x/crypto/ssh"
)

var logger *log.Logger

// this needs to be global (because TCP ports are global)
var fwdList = &forwardList{
	reverseCancellations: map[string]chan bool{},
}

func init() {
	logger = log.New(ioutil.Discard, "simplessh", 0)
}

// EnableLogging enables logging for the simplessh library
func EnableLogging(output io.Writer) {
	logger.SetOutput(output)
}

// A ConnHandler is a top level SSH Manager.  Objects implementing the ConnHandler are responsible for managing incoming Channels and Global Requests
type ConnHandler interface {
	HandleSSHConn(ssh.Conn, <-chan ssh.NewChannel, <-chan *ssh.Request)
}

// ConnHandlerFunc is an adapter that allows regular functions to act as SSH Connection Handlers
type ConnHandlerFunc func(ssh.Conn, <-chan ssh.NewChannel, <-chan *ssh.Request)

// HandleSSHConn calls f(sshConn, chans, reqs)
func (f ConnHandlerFunc) HandleSSHConn(sshConn ssh.Conn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request) {
	f(sshConn, chans, reqs)
}

// ChannelHandler handles  channel requests for a given channel type
type ChannelHandler interface {
	HandleChannel(newChannel ssh.NewChannel, channel ssh.Channel, reqs <-chan *ssh.Request, sshConn ssh.Conn)
}

// ChannelHandlerFunc is an adapter that allows regular functions to act as SSH Channel Handlers
type ChannelHandlerFunc func(newChannel ssh.NewChannel, channel ssh.Channel, reqs <-chan *ssh.Request, sshConn ssh.Conn)

// HandleChannel calls f(channelType, channel, reqs, sshConn)
func (f ChannelHandlerFunc) HandleChannel(newChannel ssh.NewChannel, channel ssh.Channel, reqs <-chan *ssh.Request, sshConn ssh.Conn) {
	f(newChannel, channel, reqs, sshConn)
}

// MultipleChannelsHandler handles  a chan of all SSH channel requests for a connection
type MultipleChannelsHandler interface {
	HandleChannels(chans <-chan ssh.NewChannel, sshConn ssh.Conn)
}

// MultipleChannelsHandlerFunc is an adapter that allows regular functions to act as SSH Multiple Channels Handlers
type MultipleChannelsHandlerFunc func(chans <-chan ssh.NewChannel, sshConn ssh.Conn)

// HandleChannels calls f(chans, sshConn)
func (f MultipleChannelsHandlerFunc) HandleChannels(chans <-chan ssh.NewChannel, sshConn ssh.Conn) {
	f(chans, sshConn)
}

// GlobalMultipleRequestsHandler handles global (not tied to a channel) out-of-band SSH Requests
type GlobalMultipleRequestsHandler interface {
	HandleRequests(<-chan *ssh.Request, ssh.Conn)
}

// GlobalMultipleRequestsHandlerFunc is an adaper to allow regular functions to act as a Global Requests Handler
type GlobalMultipleRequestsHandlerFunc func(<-chan *ssh.Request, ssh.Conn)

// HandleRequests calls f(reqs, sshConn)
func (f GlobalMultipleRequestsHandlerFunc) HandleRequests(reqs <-chan *ssh.Request, sshConn ssh.Conn) {
	f(reqs, sshConn)
}

// DiscardGlobalMultipleRequests is a wrapper around ssh.DiscardRequests. Ignores ssh ServerConn
func DiscardGlobalMultipleRequests(reqs <-chan *ssh.Request, sshConn ssh.Conn) {
	ssh.DiscardRequests(reqs)
}

// DiscardRequest appropriately discards SSH Requests, returning responses to those that expect it
func DiscardRequest(req *ssh.Request) {
	if req.WantReply {
		req.Reply(false, nil)
	}
}

// GlobalRequestHandler handles global (not tied to a channel) out-of-band SSH Requests
type GlobalRequestHandler interface {
	HandleRequest(*ssh.Request, ssh.Conn)
}

// GlobalRequestHandlerFunc is an adaper to allow regular functions to act as a Global Request Handler
type GlobalRequestHandlerFunc func(*ssh.Request, ssh.Conn)

// HandleRequest calls f(reqs, sshConn)
func (f GlobalRequestHandlerFunc) HandleRequest(req *ssh.Request, sshConn ssh.Conn) {
	f(req, sshConn)
}

// ChannelMultipleRequestsHandler handles tied to a channel out-of-band SSH Requests
type ChannelMultipleRequestsHandler interface {
	HandleMultipleRequests(reqs <-chan *ssh.Request, sshConn ssh.Conn, channelType string, channel ssh.Channel)
}

// ChannelMultipleRequestsHandlerFunc is an adaper to allow regular functions to act as a Channel Requests Handler
type ChannelMultipleRequestsHandlerFunc func(reqs <-chan *ssh.Request, sshConn ssh.Conn, channelType string, channel ssh.Channel)

// HandleMultipleRequests calls f(reqs, sshConn, channelType, channel)
func (f ChannelMultipleRequestsHandlerFunc) HandleMultipleRequests(reqs <-chan *ssh.Request, sshConn ssh.Conn, channelType string, channel ssh.Channel) {
	f(reqs, sshConn, channelType, channel)
}

// DiscardChannelMultipleRequests is a wrapper around ssh.DiscardRequests. Ignores ssh ServerConn
func DiscardChannelMultipleRequests(reqs <-chan *ssh.Request, sshConn ssh.Conn, channelType string, channel ssh.Channel) {
	ssh.DiscardRequests(reqs)
}

// ChannelRequestHandler handles  tied to a channel out-of-band SSH Requests
type ChannelRequestHandler interface {
	HandleRequest(req *ssh.Request, sshConn ssh.Conn, channelType string, channel ssh.Channel)
}

// ChannelRequestHandlerFunc is an adaper to allow regular functions to act as a Global Request Handler
type ChannelRequestHandlerFunc func(req *ssh.Request, sshConn ssh.Conn, channelType string, channel ssh.Channel)

// HandleRequest calls f(reqs, sshConn, channelType, channel)
func (f ChannelRequestHandlerFunc) HandleRequest(req *ssh.Request, sshConn ssh.Conn, channelType string, channel ssh.Channel) {
	f(req, sshConn, channelType, channel)
}

// SSHConnHandler is an SSH Channel multiplexer. It matches Channel types and calls the handler for the corresponding type
type SSHConnHandler struct {
	MultipleChannelsHandler
	GlobalMultipleRequestsHandler
}

// GlobalMultipleRequestsMux is an SSH Global Requests multiplexer. It matches Channel types and calls the handler for the corresponding type - can be used as GlobalMultipleRequestsHandler
type GlobalMultipleRequestsMux struct {
	requestMutex sync.RWMutex
	requests     map[string]GlobalRequestHandler
}

// NewGlobalMultipleRequestsMux creates and returns a GlobalMultipleRequestsHandler that performs multiplexing of request types with dispatching to GlobalRequestHandlers
func NewGlobalMultipleRequestsMux() *GlobalMultipleRequestsMux {
	return &GlobalMultipleRequestsMux{requests: map[string]GlobalRequestHandler{}}
}

// ChannelsMux is an SSH Channel multiplexer. It matches Channel types and calls the handler for the corresponding type - Can be used as ChannelsHandler
type ChannelsMux struct {
	channelMutex sync.RWMutex
	channels     map[string]ChannelHandler
}

// NewChannelsMux creates and returns a MultipleChannelsHandler that performs multiplexing of request types with dispatching to ChannelHandlers
func NewChannelsMux() *ChannelsMux {
	return &ChannelsMux{channels: map[string]ChannelHandler{}}
}

// NewSSHConnHandler creates and returns a basic working ConnHandler to provide a minimal "working" SSH server
func NewSSHConnHandler() *SSHConnHandler {
	return &SSHConnHandler{MultipleChannelsHandler: NewChannelsMux(), GlobalMultipleRequestsHandler: NewGlobalMultipleRequestsMux()}
}

// HandleSSHConn manages incoming channel and out-of-band requests. It discards out-of-band requests and dispatches channel requests if a ChannelHandler is registered for a given Channel Type
func (s *SSHConnHandler) HandleSSHConn(sshConn ssh.Conn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request) {
	go globalRequestsHandler{s}.HandleRequests(reqs, sshConn)
	go channelsHandler{s}.HandleChannels(chans, sshConn)

}

// HandleRequest registers the GlobalRequestHandler for the given Channel Type. If a GlobalRequestHandler was already registered for the type, HandleRequest panics
func (s *GlobalMultipleRequestsMux) HandleRequest(requestType string, handler GlobalRequestHandler) {
	s.requestMutex.Lock()
	defer s.requestMutex.Unlock()
	_, ok := s.requests[requestType]
	if ok {
		panic("simplessh: GlobalRequestHandler already registered for " + requestType)
	}
	s.requests[requestType] = handler
}

// HandleRequestFunc registers the Channel Handler function for the provided Request Type
func (s *GlobalMultipleRequestsMux) HandleRequestFunc(requestType string, f func(*ssh.Request, ssh.Conn)) {
	s.HandleRequest(requestType, GlobalRequestHandlerFunc(f))

}

// HandleChannel registers the ChannelHandler for the given Channel Type. If a ChannelHandler was already registered for the type, HandleChannel panics
func (s *ChannelsMux) HandleChannel(channelType string, handler ChannelHandler) {
	s.channelMutex.Lock()
	defer s.channelMutex.Unlock()
	_, ok := s.channels[channelType]
	if ok {
		panic("simplessh: ChannelHandler already registered for " + channelType)
	}
	s.channels[channelType] = handler

}

// HandleChannelFunc registers the Channel Handler function for the provided Channel Type
func (s *ChannelsMux) HandleChannelFunc(channelType string, f func(newChannel ssh.NewChannel, channel ssh.Channel, reqs <-chan *ssh.Request, sshConn ssh.Conn)) {
	s.HandleChannel(channelType, ChannelHandlerFunc(f))

}

// HandleRequests handles global out-of-band SSH Requests -
func (s *GlobalMultipleRequestsMux) HandleRequests(reqs <-chan *ssh.Request, sshConn ssh.Conn) {
	for req := range reqs {
		t := req.Type
		s.requestMutex.RLock()
		handler, ok := s.requests[t]
		if !ok {
			DiscardRequest(req)
		}

		s.requestMutex.RUnlock()

		go handler.HandleRequest(req, sshConn)
	}
}

// HandleChannels acts a a mux for incoming channel requests
func (s *ChannelsMux) HandleChannels(chans <-chan ssh.NewChannel, sshConn ssh.Conn) {
	for newChannel := range chans {

		logger.Printf("Received channel: %v", newChannel.ChannelType())

		// Check the type of channel
		t := newChannel.ChannelType()
		s.channelMutex.RLock()
		handler, ok := s.channels[t]

		s.channelMutex.RUnlock()
		if !ok {
			logger.Printf("Unknown channel type: %s", t)

			newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			logger.Printf("could not accept channel (%s)", err)
			continue
		}
		go handler.HandleChannel(newChannel, channel, requests, sshConn)

	}

}

// DefaultSSHConnHandler is an SSH Server Handler
var DefaultSSHConnHandler = &SSHConnHandler{}

// DefaultMultipleChannelsHandler is the ChannelMux and is used to handle all  incoming channel requests
var DefaultMultipleChannelsHandler = NewChannelsMux()

// DefaultGlobalMultipleRequestsHandler is a GlobalMultipleRequestsHandler that by default discards all incoming global requests
var DefaultGlobalMultipleRequestsHandler = NewGlobalMultipleRequestsMux()

// HandleRequest registers the given handler with the DefaultGlobalMultipleRequestsHandler
func HandleRequest(requestType string, handler GlobalRequestHandler) {
	DefaultGlobalMultipleRequestsHandler.HandleRequest(requestType, handler)
}

// HandleRequestFunc registers the given handler function with the DefaultGlobalMultipleRequestsHandler
func HandleRequestFunc(requestType string, handler func(*ssh.Request, ssh.Conn)) {
	DefaultGlobalMultipleRequestsHandler.HandleRequestFunc(requestType, GlobalRequestHandlerFunc(handler))
}

// HandleChannel registers the given handler under the channelType with the DefaultMultipleChannelsHandler
func HandleChannel(channelType string, handler ChannelHandler) {
	DefaultMultipleChannelsHandler.HandleChannel(channelType, handler)
}

// HandleChannelFunc registers the given handler function under the channelType with the DefaultMultipleChannelsHandler
func HandleChannelFunc(channelType string, handler func(newChannel ssh.NewChannel, channel ssh.Channel, reqs <-chan *ssh.Request, sshConn ssh.Conn)) {
	DefaultMultipleChannelsHandler.HandleChannelFunc(channelType, ChannelHandlerFunc(handler))
}

type channelsHandler struct {
	s *SSHConnHandler
}

// HandleChannels is a wrapper, tests if the SSHConnHandler has a MultipleChannelsHandler, and if not uses the default one
func (s channelsHandler) HandleChannels(chans <-chan ssh.NewChannel, sshConn ssh.Conn) {
	handler := s.s.MultipleChannelsHandler
	if handler == nil {
		handler = DefaultMultipleChannelsHandler
	}
	handler.HandleChannels(chans, sshConn)
}

type globalRequestsHandler struct {
	s *SSHConnHandler
}

// HandleRequests is a wrapper, tests if the SSHConnHandler has a GlobalMultipleRequestsHandler, and if not uses the default one
func (s globalRequestsHandler) HandleRequests(reqs <-chan *ssh.Request, sshConn ssh.Conn) {
	handler := s.s.GlobalMultipleRequestsHandler
	if handler == nil {
		handler = DefaultGlobalMultipleRequestsHandler
	}
	handler.HandleRequests(reqs, sshConn)
}
