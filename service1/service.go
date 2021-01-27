// +build !openbsd, !netbsd, !windows

package service1

import (
	//	"fmt"

	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/sutils"
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
		return service.Install(args...)
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

func (service *Service) ManageOsArgs() (string, error) {

	//	usage := "Usage: myservice install | remove | start | stop | status"

	// if received any kind of command, do it
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install(os.Args[2:]...)
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return "", nil
		}
	}
	return "", nil
}

func NewService(name string, autoinstall bool, description string, dependencies []string, args ...string) *Service {
	if sutils.IsContainer() {
		return nil
	}

	srv, err := daemon.New(name, description, dependencies...)
	//	srv, err := daemon.New(name, description, dependencies...)
	if err != nil {
		return nil
	}

	psrv := &Service{srv}
	if retstr, err := psrv.ManageOsArgs(); retstr != "" || err != nil {
		os.Exit(0)
	} else {
		if os.Getenv("PLOG") == "yes" {
			log.Infof("CommandOutput/err: [%s] [%v] \n", retstr, err)
		}
	}

	if _, err := exec.LookPath("systemctl"); err == nil {
		//Restart=on-failure
		//RemainAfterExit=no {{.Path}}
		slinkpath := "/sbin/" + name
		efulpath, err := sutils.GetExecPath()
		if err != nil {
			return nil
		}

		if epath, err := sutils.GetExecPath(); err == nil {
			if _, err := os.Lstat(slinkpath); err == nil {
				os.Remove(slinkpath)
			}

			if err := os.Symlink(epath, slinkpath); err != nil {
				slinkpath = efulpath
				//				log.Printf("Cannot symlink [use default]: %s %v", efulpath, err)
			}
		}

		psrv.SetTemplate(`[Unit]
Description={{.Description}}
Requires={{.Dependencies}}
After={{.Dependencies}}

[Service]
Environment=INSD=yes
Type=simple
PIDFile=/run/{{.Name}}.pid
ExecStartPre=/bin/rm -f /run/{{.Name}}.pid
ExecStart=` + slinkpath + ` {{.Args}}

[Install]
WantedBy=multi-user.target
`)
	}
	if autoinstall {
		status, err := psrv.Status()

		if os.Getenv("PLOG") == "yes" {
			log.Infof("Status/err: [%s] [%v] \n", status, err)
		}

		needinstall := false
		if err != nil {
			needinstall = true
		}

		if needinstall {
			if k, err := psrv.Install(args...); err == nil { //alway install
				if os.Getenv("PLOG") == "yes" {
					log.Infof("Successful install [%v]", k)
				}
			} else {
				if os.Getenv("PLOG") == "yes" {
					log.Errorf("Cannot install service: [%v]\n", err)
				}
			}
		}
	}
	os.Unsetenv("PLOG")
	return psrv
}
