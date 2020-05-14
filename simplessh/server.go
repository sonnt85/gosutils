package simplessh

import (
	"net"

	"golang.org/x/crypto/ssh"
)

// Server represents an SSH Server. The SSH ServerConfig must be provided
type Server struct {
	Addr    string
	Config  *ssh.ServerConfig
	Handler ConnHandler
	*ssh.ServerConn
}

// ListenAndServe listens on the TCP address s.Addr and then calls Serve to handle requests on incoming connections. If s.Addr is blank, ":ssh" is used
func (s *Server) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":ssh"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.Serve(ln.(*net.TCPListener))
}

// Serve accepts incoming connections on the provided listener.and reads global SSH Channel and Out-of-band requests and calls s,ConnHandler to handle them
func (s *Server) Serve(l net.Listener) error {
	defer l.Close()
	logger.Print("SSH Server started listening on: ", l.Addr())
	for {
		tcpConn, err := l.Accept()
		if err != nil {
			return err
		}

		c, err := s.newConn(tcpConn)
		if err != nil {
			continue
		}
		go c.serve()
	}
}

// HandleOpenChannel requests that the remote end accept a channel request and if accepted,
// passes the newly opened channel and requests to the provided handler
func (s *Server) HandleOpenChannel(channelName string, handler ChannelMultipleRequestsHandler, data ...byte) error {
	ch, reqs, err := s.OpenChannel(channelName, data)
	if err != nil {
		return err
	}
	handler.HandleMultipleRequests(reqs, s.ServerConn, channelName, ch)
	return nil
}

// HandleOpenChannelFunc requests that the remote end accept a channel request and if accepted,
// passes the newly opened channel and requests to the provided handler function
func (s *Server) HandleOpenChannelFunc(channelName string, handler func(reqs <-chan *ssh.Request, sshConn ssh.Conn, channelType string, channel ssh.Channel), data ...byte) error {

	return s.HandleOpenChannel(channelName, ChannelMultipleRequestsHandlerFunc(handler), data...)
}

type conn struct {
	server     *Server
	remoteAddr string
	conn       net.Conn
}

func (c *conn) serve() {
	sshConn, chans, reqs, err := ssh.NewServerConn(c.conn, c.server.Config)
	if err != nil {
		return
	}
	c.server.ServerConn = sshConn
	logger.Print("New ssh connection from: ", c.conn.RemoteAddr())

	go func() {
		sshConn.Wait()
		logger.Print("Closing ssh connection from: ", c.conn.RemoteAddr())
		if c.conn != nil {
			c.conn.Close()
			//c.conn = nil
		}
	}()

	// Use default ConnHandler if one isn't provided
	serverHandler{c.server}.HandleSSHConn(sshConn, chans, reqs)
}

func (s *Server) newConn(netConn net.Conn) (*conn, error) {
	c := new(conn)
	c.remoteAddr = netConn.RemoteAddr().String()
	c.server = s
	c.conn = netConn
	return c, nil

}

// NewSessionServerHandler creates a ConnHandler to provide a more standard SSH server providing sessions
func NewSessionServerHandler() *SSHConnHandler {
	s := SSHConnHandler{}
	channelHandler := NewChannelsMux()

	channelHandler.HandleChannel(SessionRequest, SessionHandler())
	s.MultipleChannelsHandler = channelHandler
	return &s
}

// NewStandardSSHServerHandler returns a server handler that can deal with ssh sessions and both local and remote port forwarding
func NewStandardSSHServerHandler() *SSHConnHandler {
	s := NewSSHConnHandler()

	chHandler := NewChannelsMux()
	chHandler.HandleChannel(SessionRequest, SessionHandler())
	chHandler.HandleChannel(DirectForwardRequest, DirectPortForwardHandler())

	s.MultipleChannelsHandler = chHandler

	globalHandler := NewGlobalMultipleRequestsMux()
	globalHandler.HandleRequest(RemoteForwardRequest, TCPIPForwardRequestHandler())

	//	globalHandler.HandleRequest(CancelRemoteForwardRequest, TCPIPForwardCancelRequestHandler())

	s.GlobalMultipleRequestsHandler = globalHandler
	return s

}

// ListenAndServe listens on the given tcp address addr and then calls Serve with handler.
// If handler is nil, the DefaultServerHandler is used.
func ListenAndServe(addr string, conf *ssh.ServerConfig, handler ConnHandler) error {
	s := &Server{addr, conf, handler, nil}
	return s.ListenAndServe()

}

// Serve accepts incoming SSH connections on the listener l.
// If handler is nil, the DefaultServerHandler is used.
func Serve(l net.Listener, conf *ssh.ServerConfig, handler ConnHandler) error {
	s := &Server{Config: conf, Handler: handler}
	return s.Serve(l)
}

type serverHandler struct {
	s *Server
}

// ServeSSH is a wrapper, tests if the server has a ServerHandler, and if not uses the default one
func (s serverHandler) HandleSSHConn(conn *ssh.ServerConn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request) {
	handler := s.s.Handler
	if handler == nil {
		handler = DefaultSSHConnHandler
	}
	handler.HandleSSHConn(conn, chans, reqs)
}
