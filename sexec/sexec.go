package sexec

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/sutils"
)

func ExecCommandShellEnvTimeout(script string, moreenvs map[string]string, timeout time.Duration, scriptrunoption ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxShellEnvTimeout(nil, script, moreenvs, timeout, scriptrunoption...)
}

func ExecCommandScriptEnvTimeout(scriptbin, script string, moreenvs map[string]string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxScriptEnvTimeout(nil, scriptbin, script, moreenvs, timeout, arg...)
}

// func ExecCommandShellEnvTimeoutAs(script string, moreenvs map[string]string, timeout time.Duration) (stdOut, stdErr []byte, err error) {
// }

func ExecCommandShell(script string, timeout time.Duration, dummy ...bool) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxShell(nil, script, timeout)
}

func ExecCommandEnvTimeout(name string, newenvs map[string]string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxEnvTimeout(nil, name, newenvs, timeout, arg...)
}

// run command without timeout
func ExecCommand(name string, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtx(nil, name, arg...)
}

// run command with timeout
func ExecCommandTimeout(name string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxTimeout(nil, name, timeout, arg...)
}

func LookPath(efile string) string {
	if ret, err := exec.LookPath(efile); err == nil {
		return ret
	}
	return ""
}

func ExecCommandEnv(name string, moreenvs map[string]string, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxEnv(nil, name, moreenvs, arg...)
}

func GetExecPath() (pathexe string, err error) {
	pathexe, err = os.Executable()
	if err != nil {
		// log.Println("Cannot  get binary")
		// return os.Args[0], nil
		return "", err
	}
	// os.Readlink(pathexe)
	var tmppathexe string
	if tmppathexe, err = filepath.EvalSymlinks(pathexe); err == nil {
		pathexe = tmppathexe
	} else {
		err = nil
	}
	return
}

// spaw father to  child via syscall, merge executablePath to executableArgs if first executableArgs[0] is diffirence executablePath
func ExecCommandSyscall(executablePath string, executableArgs []string, executableEnvs map[string]string) error {
	return ExecCommandCtxSyscall(nil, executablePath, executableArgs, executableEnvs)
}

func ExecBytesToFileEnvTimeout(byteprog []byte, progname, workdir string, newenvs map[string]string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	var filePath string
	if len(workdir) == 0 {
		workdir, err = ioutil.TempDir("", "system")
		if err != nil {
			return retstdout, retstderr, err
		} else {
			defer os.RemoveAll(workdir)
		}
	} else {
		if err = os.MkdirAll(workdir, 0700); err != nil {
			return retstdout, retstderr, err
		} else {
			defer os.Remove(filePath)
		}
	}

	filePath = filepath.Join(workdir, progname)
	err = ioutil.WriteFile(filePath, byteprog, 0755)
	if err != nil {
		log.Errorf("Can not create new file to run: %v", err)
		return retstdout, retstderr, err
	}

	//sutils.PathHasFile(filepath, PATH)
	os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathJointList(sutils.PathGetEnvPathValue(), filepath.Dir(filePath)))
	defer os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathRemove(sutils.PathGetEnvPathValue(), filepath.Dir(filePath)))
	return ExecCommandEnvTimeout(progname, newenvs, timeout, args...)
}

func ExecBytesEnvTimeout(byteprog []byte, name string, moreenvs map[string]string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	return ExecBytesCtxEnvTimeout(nil, byteprog, name, moreenvs, timeout, args...)
}

func ExecBytesEnv(byteprog []byte, name string, moreenvs map[string]string, args ...string) (retstdout, retstderr []byte, err error) {
	return ExecBytesCtxEnv(nil, byteprog, name, moreenvs, args...)
	// TODO: Validate user input
	// return ExecBytesCtxEnv(context.TODO(), byteprog, name, moreenvs, args...)

}

func ExecBytesTimeout(byteprog []byte, name string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	return ExecBytesCtxTimeout(nil, byteprog, name, timeout, args...)
}

func ExecBytes(byteprog []byte, name string, args ...string) (retstdout, retstderr []byte, err error) {
	return ExecBytesCtx(nil, byteprog, name, args...)
}

// exe is empty will run current program
func ExecCommandShellElevated(exe string, showCmd int32, args ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxShellElevated(nil, exe, showCmd, args...)
}

// exe is empty will run current program
func ExecCommandShellElevatedEnvTimeout(exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxShellElevatedEnvTimeout(nil, exe, showCmd, moreenvs, timeout, args...)
}

func MakeCmdLine(args ...string) string {
	return makeCmdLine(args)
}

func CmdHiddenConsole(cmd *exec.Cmd) {
	cmdHiddenConsole(cmd)
}
