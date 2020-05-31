// Example of a daemon with echo service
package service

import (
	. "github.com/sonnt85/gosutils/sutils"
	"github.com/takama/daemon"
)

// dependencies that are NOT required by the service, but might be used
//var dependencies = []string{"dummy.service"}

// Service has embedded daemon
type Service struct {
	daemon.Daemon
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage(command string) (string, error) {
	switch command {
	case "install":
		return service.Install()
	case "remove":
		return service.Remove()
	case "start":
		return service.Start()
	case "stop":
		return service.Stop()
	case "status":
		return service.Status()
	default:
		return command, nil
	}
}

func NewService(name, description string) *Service {
	if IsContainer() {
		return nil
	}

	srv, err := daemon.New(name, description)
	//	srv, err := daemon.New(name, description, dependencies...)

	if err != nil {
		return nil
	}
	psrv := &Service{srv}
	_, err = psrv.Status()

	if err != nil {
		return nil
	}
	return psrv
}
