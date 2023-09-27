package sexec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/process"

	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/sutils"
)

func ExecCommandShellEnvTimeout(script string, moreenvs map[string]string, timeout time.Duration, scriptrunoption ...string) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxShellEnvTimeout(nil, script, moreenvs, timeout, scriptrunoption...)
}

func ExecCommandScriptEnvTimeout(scriptbin, script string, moreenvs map[string]string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxScriptEnvTimeout(nil, scriptbin, script, moreenvs, timeout, arg...)
}

// func ExecCommandShellEnvTimeoutAs(script string, moreenvs map[string]string, timeout time.Duration) (stdOut, stdErr []byte, err error) {
// }

func ExecCommandShellTimeout(script string, timeout time.Duration) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxShell(nil, script, timeout)
}

func ExecCommandShell(script string, timeouts ...time.Duration) (stdOut, stdErr []byte, err error) {
	var timeout time.Duration
	timeout = 0
	if len(timeouts) != 0 {
		timeout = timeouts[0]
	}
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxShell(nil, script, timeout)
}

func ExecCommandEnvTimeout(name string, newenvs map[string]string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxEnvTimeout(nil, name, newenvs, timeout, arg...)
}

// run command without timeout
func ExecCommand(name string, arg ...string) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtx(nil, name, arg...)
}

// run command with timeout
func ExecCommandTimeout(name string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxTimeout(nil, name, timeout, arg...)
}

func LookPath(efile string) string {
	if ret, err := exec.LookPath(efile); err == nil {
		return ret
	}
	return ""
}

func CheckExecutablePermission(efile string) bool {
	if _, err := exec.LookPath(efile); err == nil {
		return true
	}
	return false
}

func ExecCommandEnv(name string, moreenvs map[string]string, arg ...string) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
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
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxSyscall(nil, executablePath, executableArgs, executableEnvs)
}

func ExecBytesToFileEnvTimeout(byteprog []byte, progname, workdir string, newenvs map[string]string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	var filePath string
	if len(workdir) == 0 {
		workdir, err = os.MkdirTemp("", "system")
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
	err = os.WriteFile(filePath, byteprog, 0755)
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
	//lint:ignore SA1012 ignore this!
	return ExecBytesCtxEnvTimeout(nil, byteprog, name, moreenvs, timeout, args...)
}

func ExecBytesEnv(byteprog []byte, name string, moreenvs map[string]string, args ...string) (retstdout, retstderr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecBytesCtxEnv(nil, byteprog, name, moreenvs, args...)
	// TODO: Validate user input
	// return ExecBytesCtxEnv(context.TODO(), byteprog, name, moreenvs, args...)

}

func ExecBytesTimeout(byteprog []byte, name string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecBytesCtxTimeout(nil, byteprog, name, timeout, args...)
}

func ExecBytes(byteprog []byte, name string, args ...string) (retstdout, retstderr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecBytesCtx(nil, byteprog, name, args...)
}

// exe is empty will run current program
func ExecCommandShellElevated(exe string, showCmd int32, args ...string) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxShellElevated(nil, exe, showCmd, args...)
}

// exe is empty will run current program
func ExecCommandShellElevatedEnvTimeout(exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxShellElevatedEnvTimeout(nil, exe, showCmd, moreenvs, timeout, args...)
}

func MakeCmdLine(args ...string) string {
	return makeCmdLine(args)
}

func CmdHiddenConsole(cmd *exec.Cmd) {
	cmdHiddenConsole(cmd)
}

func EnrovimentMergeWithCurrentEnv(envMap map[string]string) (senv []string) {
	var currentEnvMap = make(map[string]string, 0)
	for _, rawEnvLine := range os.Environ() {
		k, v, ok := strings.Cut(rawEnvLine, "=")
		if !ok {
			continue
		}
		currentEnvMap[k] = v
	}

	for key, value := range envMap {
		currentEnvMap[key] = value
	}
	for key, value := range currentEnvMap {
		senv = append(senv, fmt.Sprintf("%s=%s", key, value))
	}
	return
}

func IsConsoleExecutable(path string) bool {
	consoleBasenames := []string{
		"setx",
		"cmd",        // Windows cmd
		"powershell", // Windows PowerShell
		"bash",       // Bash on Linux/Unix
		"sh",         // Sh on Linux/Unix
		"zsh",        // Zsh on Linux/Unix
		"fish",       // Fish Shell on Linux/Unix
		"python",     // Python interpreter
		"ruby",       // Ruby interpreter
		"perl",       // Perl interpreter
		"node",       // Node.js interpreter
		"php",        // PHP interpreter
		"lua",        // Lua interpreter
		"jshell",     // Java Shell (JShell)
		"openvpn-gui",
	}
	if runtime.GOOS != "darwin" {
		return true
	}
	base := strings.ToLower(filepath.Base(path))
	baseWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))

	for _, consoleBasename := range consoleBasenames {
		if baseWithoutExt == consoleBasename {
			return true
		}
	}

	return false
}

func Open(b []byte, progname string) (f *os.File, err error) {
	return open(b, progname)
}

func OpenMemFd(b []byte, name string) (*os.File, error) {
	return openMemFd(b, name)
}

func Processes(names ...string) (ps []*process.Process, err error) {
	// func Pgrep(name string, isFullname ...bool) error {
	ps, err = process.Processes()
	// if len(names) == 0 {
	// 	return
	// }

	if err != nil {
		return
	}
	var pname string
	// found := false
	retps := make([]*process.Process, 0)
	for _, p := range ps {
		pname, err = p.Name()

		if len(names) == 0 || (len(names) == 1 && (names[0] == "*" || names[0] == "")) || (err == nil && sutils.SlideHasElementInStrings(names, pname)) {
			retps = append(retps, p)
		}
	}
	return retps, nil
}

func PgrepWithEnv(names string, key, val string) (ps []*process.Process) {
	if pst, err := Processes(names); err == nil {
		ps = make([]*process.Process, 0)
		env := fmt.Sprintf("%s=%s", key, val)
		for _, p := range pst {
			if nvs, e := p.Environ(); e == nil {
				for _, v := range nvs {
					if v == env || key == "*" || (val == "*" && strings.HasPrefix(v, fmt.Sprintf("%s=", key))) {
						ps = append(ps, p)
						break
					}
				}
			}
		}
	}
	return
}

func Pgrep(names ...string) (ps []*process.Process) {
	ps, _ = Processes(names...)
	return
}

func Command(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	CmdHiddenConsole(cmd)
	return cmd
}
