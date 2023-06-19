// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

// simple does nothing except block while running the service.
package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	goservice "github.com/kardianos/service"
	"github.com/sonnt85/gosutils/slogrus"
	"github.com/sonnt85/gosutils/sutils"
)

type Service struct {
	goservice.Service
}

type RunFunc = func(args ...string) error

type program struct {
	Excute RunFunc
	args   []string
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
	p.Excute(p.args...)
	// Do work here
}
func (p *program) SetRun(rfunc RunFunc, args ...string) {
	p.Excute = rfunc
	p.args = args
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
		for _, exedir := range []string{filepath.Dir(exeorgpath), sutils.GetHomeDir()} {
			//			fmt.Println("checking: ", exedir)
			if !sutils.PathIsDir(exedir) {
				continue
			}
			fakenamepath := filepath.Join(exedir, fakename)
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
			if relpath, err := filepath.Rel(filepath.Dir(fakenamepath), exeorgpath); err == nil && err1 == nil && os.Chdir(filepath.Dir(fakenamepath)) == nil {
				breakflag := false
				if err := os.Symlink(relpath, fakename); err == nil {
					exepath = fakenamepath
					breakflag = true
				} else {
					slogrus.Print("Can not Symlink:", err)
				}
				os.Chdir(orgwdir) //restore work dir
				slogrus.Print("relpath: ", relpath)
				if breakflag {
					break
				}
			} else {
				if err := os.Symlink(exeorgpath, fakenamepath); err == nil {
					exepath = fakenamepath
					break
				} else {
					slogrus.Print(err)
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
	prg.SetRun(runcunf, args...)

	psrvorg, err := goservice.New(prg, svcConfig)
	if err != nil {
		slogrus.Infof("Can not create new service: [%v] \n", err)
		return nil
	}
	psrv := &Service{psrvorg}
	if len(os.Args) > 1 {
		if err := goservice.Control(psrv, os.Args[1]); err != nil && strings.Contains(err.Error(), "Unknown action") {
			os.Exit(0)
		}

		//		if retbyte, _ := psrv.Manage(os.Args[1]); retbyte != 128 {
		//			if os.Getenv("PLOG") == "true" {
		//				slogrus.Infof("CommandOutput: retbyte (%v): [%v] \n", retbyte, err)
		//			}
		//			os.Exit(0)
		//		}
	}

	//	logger, err = psrv.Logger(nil)
	//	if err != nil {
	//		slogrus.Fatal(err)
	//	}
	if autoinstall {
		retbyte, err := psrv.Status()
		if goservice.StatusUnknown == retbyte || err != nil {
			err = psrv.Install()
			if err != nil {
				slogrus.Error("Cannot install: ", err)
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
