// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

// simple does nothing except block while running the service.
package service

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	goservice "github.com/kardianos/service"
	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/sutils"
)

type Service struct {
	goservice.Service
}

type RunFunc = func() error

type program struct {
	Excute RunFunc
}

func (p *program) Start(s goservice.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	os.Exit(0)
	return nil
}
func (p *program) run() {
	if p.Excute == nil {
		return
	}
	p.Excute()
	// Do work here
}
func (p *program) SetRun(rfunc RunFunc) {
	p.Excute = rfunc
	// Do work here
}

func (p *program) Stop(s goservice.Service) error {
	// Stop should not block. Return with a few seconds.
	os.Exit(0)
	return nil
}

func (service *Service) Manage(command string) (byte, error) {
	var ncbyte = byte(128)
	switch command {
	case "install":
		return byte(127), service.Install()
	case "remove":
		return byte(127), service.Uninstall()
	case "start":
		return byte(127), service.Start()
	case "stop":
		return byte(127), service.Stop()
	case "status":
		retbyte, err := service.Status()
		return byte(retbyte), err
	default:
		return ncbyte, nil
	}
	// return ncbyte, nil
}

func NewService(runcunf RunFunc, name, fakename string, autoinstall bool, description string, dependencies []string, args ...string) *Service {
	if sutils.IsContainer() {
		return nil
	}

	exeorgpath, err := sutils.GetExecPath()
	if err != nil {
		return nil
	}

	exepath := exeorgpath

	if fakename != "" {
		for _, exedir := range []string{path.Dir(exeorgpath), sutils.GetHomeDir()} {
			//			fmt.Println("checking: ", exedir)
			if !sutils.PathIsDir(exedir) {
				continue
			}
			fakenamepath := path.Join(exedir, fakename)
			if runtime.GOOS == "windows" { //not use symlink for window
				fakenamepath = fakenamepath + ".exe"
				exepath = fakenamepath
				if _, err := sutils.FileCopy(exeorgpath, fakenamepath); err == nil {
					break
				}
			}

			if _, err := os.Lstat(fakenamepath); err == nil { //not windows
				if tpath, err := os.Readlink(fakenamepath); err == nil {
					//					fmt.Println("tpath, fakenamepath: ", tpath, fakenamepath)
					if tpath != exeorgpath {
						os.Remove(fakenamepath)
					} else {
						break
					}
				}
			}

			orgwdir, err1 := os.Getwd()
			if relpath, err := filepath.Rel(path.Dir(fakenamepath), exeorgpath); err == nil && err1 == nil && os.Chdir(path.Dir(fakenamepath)) == nil {
				breakflag := false
				if err := os.Symlink(relpath, fakename); err == nil {
					exepath = fakenamepath
					breakflag = true
				} else {
					//					fmt.Println("Can not Symlink:", err)
				}
				os.Chdir(orgwdir) //restore work dir
				//				fmt.Println("relpath: ", relpath)
				if breakflag {
					break
				}
			} else {
				if err := os.Symlink(exeorgpath, fakenamepath); err == nil {
					exepath = fakenamepath
					break
				} else {
					//					fmt.Println(err)
				}
			}

		}
	}

	//	fmt.Println(exepath)
	svcConfig := &goservice.Config{
		Name:         name,
		Description:  description,
		DisplayName:  "System service",
		Arguments:    args,
		Dependencies: dependencies,
		Executable:   exepath,
	}

	if runtime.GOOS == "windows" {
		svcConfig.Executable = "nssm"
		svcConfig.Arguments = append([]string{exepath}, svcConfig.Arguments...)
	}

	prg := &program{}
	prg.SetRun(runcunf)

	psrvorg, err := goservice.New(prg, svcConfig)
	if err != nil {
		if os.Getenv("PLOG") == "yes" {
			log.Infof("Can not create new service: [%v] \n", err)
		}
		return nil
	}
	psrv := &Service{psrvorg}
	if len(os.Args) > 1 {
		if err := goservice.Control(psrv, os.Args[1]); err != nil && strings.Contains(err.Error(), "Unknown action") {
			os.Exit(0)
		}

		//		if retbyte, _ := psrv.Manage(os.Args[1]); retbyte != 128 {
		//			if os.Getenv("PLOG") == "yes" {
		//				log.Infof("CommandOutput: retbyte (%v): [%v] \n", retbyte, err)
		//			}
		//			os.Exit(0)
		//		}
	}

	//	logger, err = psrv.Logger(nil)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	if autoinstall {
		retbyte, err := psrv.Status()
		if goservice.StatusUnknown == retbyte || err != nil {
			err = psrv.Install()
			if err != nil {
				log.Error("Cannot install: ", err)
			}
		}
	}

	//	fmt.Print("End NewService")
	// && goservice.Interactive()
	//	if runtime.GOOS == "windows" {
	//		time.Sleep(time.Second * 2)
	//		psrv.Run()
	//		os.Exit(0)
	//	}
	return psrv
}
