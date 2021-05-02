// +build darwin dragonfly freebsd linux netbsd openbsd plan9 solaris

package daemon

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sonnt85/gosutils/sutils"

	//	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	//	"time"
)

// A Context describes daemon context.
type Context struct {
	// If PidFileName is non-empty, parent process will try to create and lock
	// pid file with given name. Child process writes process id to file.
	PidFileName string
	// Permissions for new pid file.
	PidFilePerm os.FileMode

	// If LogFileName is non-empty, parent process will create file with given name
	// and will link to fd 2 (stderr) for child process.
	LogFileName string
	// Permissions for new log file.
	LogFilePerm os.FileMode

	// If WorkDir is non-empty, the child changes into the directory before
	// creating the process.
	WorkDir string
	// If Chroot is non-empty, the child changes root directory
	Chroot string

	// If Env is non-nil, it gives the environment variables for the
	// daemon-process in the form returned by os.Environ.
	// If it is nil, the result of os.Environ will be used.
	Env []string
	// If Args is non-nil, it gives the command-line args for the
	// daemon-process. If it is nil, the result of os.Args will be used.
	Args []string

	// Credential holds user and group identities to be assumed by a daemon-process.
	Credential *syscall.Credential
	// If Umask is non-zero, the daemon-process call Umask() func with given value.
	Umask int

	// Struct contains only serializable public fields (!!!)
	abspath  string
	pidFile  *LockFile
	logFile  *os.File
	nullFile *os.File

	rpipe, wpipe    *os.File
	NameProg        string
	ExeRealPathFake string
	CloneFlag       bool
}

func (d *Context) reborn() (child *os.Process, err error) {
	if !WasReborn() {
		child, err = d.parent()
	} else {
		err = d.child()
	}
	return
}

func (d *Context) search() (daemon *os.Process, err error) {
	if len(d.PidFileName) > 0 {
		var pid int
		if pid, err = ReadPidFile(d.PidFileName); err != nil {
			return
		}
		daemon, err = os.FindProcess(pid)
	}
	return
}

func (d *Context) parent() (child *os.Process, err error) {
	if err = d.prepareEnv(); err != nil {
		return
	}

	defer d.closeFiles()
	if err = d.openFiles(); err != nil {
		return
	}

	attr := &os.ProcAttr{
		Dir:   d.WorkDir,
		Env:   d.Env,
		Files: d.files(),
		Sys: &syscall.SysProcAttr{
			//Chroot:     d.Chroot,
			Credential: d.Credential,
			Setsid:     true,
		},
	}

	if len(d.ExeRealPathFake) != 0 {
		defer func() {
			//			log.Println("\nRemove Exe symlink file\n")
			//			if !d.CloneFlag {
			os.RemoveAll(path.Dir(d.ExeRealPathFake))
			//			}
		}()
	}
	if child, err = os.StartProcess(d.abspath, d.Args, attr); err != nil {
		if d.pidFile != nil {
			d.pidFile.Remove()
		}
		return
	}

	d.rpipe.Close()
	encoder := json.NewEncoder(d.wpipe)
	if err = encoder.Encode(d); err != nil {
		return
	}
	_, err = fmt.Fprint(d.wpipe, "\n\n")
	return
}

func (d *Context) openFiles() (err error) {
	if d.PidFilePerm == 0 {
		d.PidFilePerm = FILE_PERM
	}
	if d.LogFilePerm == 0 {
		d.LogFilePerm = FILE_PERM
	}

	if d.nullFile, err = os.Open(os.DevNull); err != nil {
		return
	}

	if len(d.PidFileName) > 0 {
		if d.PidFileName, err = filepath.Abs(d.PidFileName); err != nil {
			return err
		}
		if d.pidFile, err = OpenLockFile(d.PidFileName, d.PidFilePerm); err != nil {
			return
		}
		if err = d.pidFile.Lock(); err != nil {
			return
		}
		if len(d.Chroot) > 0 {
			// Calculate PID-file absolute path in child's environment
			if d.PidFileName, err = filepath.Rel(d.Chroot, d.PidFileName); err != nil {
				return err
			}
			d.PidFileName = "/" + d.PidFileName
		}
	}

	if len(d.LogFileName) > 0 {
		if d.logFile, err = os.OpenFile(d.LogFileName,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND, d.LogFilePerm); err != nil {
			return
		}
	}

	d.rpipe, d.wpipe, err = os.Pipe()
	return
}

func (d *Context) closeFiles() (err error) {
	cl := func(file **os.File) {
		if *file != nil {
			(*file).Close()
			*file = nil
		}
	}
	cl(&d.rpipe)
	cl(&d.wpipe)
	cl(&d.logFile)
	cl(&d.nullFile)
	if d.pidFile != nil {
		d.pidFile.Close()
		d.pidFile = nil
	}
	return
}

func (d *Context) prepareEnv() (err error) {
	if d.abspath, err = osExecutable(); err != nil {
		//		fmt.Println("can not get exec from prepareEnv")
		return
	}

	if len(d.Args) == 0 {
		d.Args = os.Args
	}
	if len(d.NameProg) != 0 {
		tmppath := sutils.TempFileCreateInNewTemDir(d.NameProg)
		if len(tmppath) != 0 {
			if d.CloneFlag {
				_, err = sutils.FileCopy(d.abspath, tmppath)
				//				err = os.Rename(d.abspath, tmppath)
				err = os.Chmod(tmppath, 0755)
				if err != nil {
					//					log.Println("Can not clone file", err)
				}
			} else {
				err = os.Symlink(d.abspath, tmppath)
			}
			if err != nil {
				return err
			}
			//				log.Printf("\nUse fake name: %v\n", tmppath)
			os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathJointList(sutils.PathGetEnvPathValue(), path.Dir(tmppath)))
			d.Args[0] = d.NameProg
			d.abspath = tmppath
			d.ExeRealPathFake = tmppath
			//				log.Printf("\nUse fake name: %v\n%v:%v\n", tmppath, sutils.PathGetEnvPathKey(), sutils.PathGetEnvPathValue())
		}
	}
	mark := fmt.Sprintf("%s=%s", MARK_NAME, MARK_VALUE)
	if len(d.Env) == 0 {
		d.Env = os.Environ()
	}
	d.Env = append(d.Env, mark)
	d.Env = append(d.Env, fmt.Sprintf("%s=%s", ASBPATHORG_NAME, d.abspath))
	d.Env = append(d.Env, fmt.Sprintf("%s=%s", ASBPATHCLONE_NAME, d.ExeRealPathFake))

	return
}

func (d *Context) files() (f []*os.File) {
	log := d.nullFile
	if d.logFile != nil {
		log = d.logFile
	}

	f = []*os.File{
		d.rpipe,    // (0) stdin
		log,        // (1) stdout
		log,        // (2) stderr
		d.nullFile, // (3) dup on fd 0 after initialization
	}

	if d.pidFile != nil {
		f = append(f, d.pidFile.File) // (4) pid file
	}
	return
}

var initialized = false

func (d *Context) child() (err error) {
	if initialized {
		return os.ErrInvalid
	}
	initialized = true

	decoder := json.NewDecoder(os.Stdin)
	if err = decoder.Decode(d); err != nil {
		return
	}

	// create PID file after context decoding to know PID file full path.
	if len(d.PidFileName) > 0 {
		d.pidFile = NewLockFile(os.NewFile(4, d.PidFileName))
		if err = d.pidFile.WritePid(); err != nil {
			return
		}
		defer func() {
			if err != nil {
				d.pidFile.Remove()
			}
		}()
	}

	if err = syscallDup(3, 0); err != nil {
		return
	}

	if d.Umask != 0 {
		syscall.Umask(int(d.Umask))
	}
	if len(d.Chroot) > 0 {
		err = syscall.Chroot(d.Chroot)
		if err != nil {
			return
		}
	}

	return
}

func (d *Context) release() (err error) {
	if !initialized {
		return
	}
	if d.pidFile != nil {
		err = d.pidFile.Remove()
	}
	return
}
