package sexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"

	"github.com/sonnt85/gosutils/sutils"
)

var prefixDirName = "system_p"
var prefixarg0 = "arg0:"
var prefixScriptName = "scriptname:"

type MemFile struct {
	*os.File
	tmpdir string
}

func (f *MemFile) Close() error {
	return f.close()
}

func ExecCommandShellEnvTimeout(script string, moreenvs map[string]string, timeout time.Duration, scriptrunoption ...interface{}) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxShellEnvTimeout(nil, script, moreenvs, timeout, scriptrunoption...)
}

func ExecCommandScriptEnvTimeout(scriptbin, script string, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxScriptEnvTimeout(nil, scriptbin, script, moreenvs, timeout, args...)
}

// func ExecCommandShellEnvTimeoutAs(script string, moreenvs map[string]string, timeout time.Duration) (stdOut, stdErr []byte, err error) {
// }

func ExecCommandShellTimeout(script string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxShell(nil, script, timeout, args...)
}

// args time.Durration, string, io.Writer
func ExecCommandShell(script string, args ...interface{}) (stdOut, stdErr []byte, err error) {
	var timeout time.Duration = -1
	// timeouts ...time.Duration,
	arg := make([]interface{}, 0)
	for _, a := range args {
		switch v := a.(type) {
		case time.Duration:
			timeout = v
		case int:
			timeout = time.Duration(v)
		default:
			arg = append(arg, v)
		}
	}
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxShell(nil, script, timeout, arg...)
}

func ExecCommandEnvTimeout(name string, newenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxEnvTimeout(nil, name, newenvs, timeout, args...)
}

// run command without timeout
// func ExecCommand(name string, args ...interface{}) (stdOut, stdErr []byte, err error) {
// 	//lint:ignore SA1012 ignore this!
// 	return ExecCommandCtx(nil, name, args...)
// }

// func ExecCommandCtxEnvTimeout(ctxc context.Context, name string, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
func ExecCommand(name string, args ...interface{}) (stdOut, stdErr []byte, err error) {
	var ctxc context.Context
	var timeout time.Duration
	var arg []interface{}
	var moreenvs map[string]string
	for _, a := range args {
		switch v := a.(type) {
		case time.Duration:
			timeout = v
		case map[string]string:
			moreenvs = v
		default:
			arg = append(arg, v)
			// err = fmt.Errorf("unsupported type: %v", v)
			// return
		}
	}
	return ExecCommandCtxEnvTimeout(ctxc, name, moreenvs, timeout, arg...)
}

// run command with timeout
func ExecCommandTimeout(name string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxTimeout(nil, name, timeout, args...)
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

func ExecCommandEnv(name string, moreenvs map[string]string, args ...interface{}) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxEnv(nil, name, moreenvs, args...)
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

func ExecBytesToFileEnvTimeout(byteprog interface{}, progname, workdir string, newenvs map[string]string, timeout time.Duration, args ...interface{}) (retstdout, retstderr []byte, err error) {
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
	var r io.Reader
	switch v := byteprog.(type) {
	case []byte:
		r = bytes.NewBuffer(v)
	case io.Reader:
		r = v
	}
	var w *os.File
	w, err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return
	}
	defer w.Close()
	_, err = io.Copy(w, r)
	if err != nil {
		return
	}
	// err = os.WriteFile(filePath, byteprog, 0755)
	// if err != nil {
	// 	log.Errorf("Can not create new file to run: %v", err)
	// 	return retstdout, retstderr, err
	// }

	//sutils.PathHasFile(filepath, PATH)
	os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathJointList(sutils.PathGetEnvPathValue(), filepath.Dir(filePath)))
	defer os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathRemove(sutils.PathGetEnvPathValue(), filepath.Dir(filePath)))
	return ExecCommandEnvTimeout(progname, newenvs, timeout, args...)
}

func ExecBytesEnvTimeout(byteprog interface{}, name string, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (retstdout, retstderr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecBytesCtxEnvTimeout(nil, byteprog, name, moreenvs, timeout, args...)
}

func ExecBytesEnv(byteprog interface{}, name string, moreenvs map[string]string, args ...interface{}) (retstdout, retstderr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecBytesCtxEnv(nil, byteprog, name, moreenvs, args...)
	// TODO: Validate user input
	// return ExecBytesCtxEnv(context.TODO(), byteprog, name, moreenvs, args...)

}

func ExecBytesTimeout(byteprog interface{}, name string, timeout time.Duration, args ...interface{}) (retstdout, retstderr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecBytesCtxTimeout(nil, byteprog, name, timeout, args...)
}

// args: io.Writer, io.Writer for stdout, stderr, strings for arg
// func ExecBytesCtxEnvTimeout(ctx context.Context, byteprog interface{}, name string, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (retstdout, retstderr []byte, err error) {
func ExecBytes(byteprog interface{}, name string, args ...interface{}) (retstdout, retstderr []byte, err error) {
	var ctxc context.Context
	var timeout time.Duration
	var arg []interface{}
	var moreenvs map[string]string
	for _, a := range args {
		switch v := a.(type) {
		case map[string]string:
			moreenvs = v
		case time.Duration:
			timeout = v
		default:
			arg = append(arg, v)
		}
	}
	return ExecBytesCtxEnvTimeout(ctxc, byteprog, name, moreenvs, timeout, arg...)
}

// exe is empty will run current program
func ExecCommandShellElevated(exe string, showCmd int32, args ...interface{}) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxShellElevated(nil, exe, showCmd, args...)
}

// exe is empty will run current program
func ExecCommandShellElevatedEnvTimeout(exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	//lint:ignore SA1012 ignore this!
	return ExecCommandCtxShellElevatedEnvTimeout(nil, exe, showCmd, moreenvs, timeout, args...)
}

func MakeCmdLine(args ...string) string {
	return makeCmdLine(args)
}

func CmdHiddenConsole(cmd *exec.Cmd) {
	cmdHiddenConsole(cmd)
}

func EnrovimentMapToStrings(envMap map[string]string) (senv []string) {
	for key, value := range envMap {
		senv = append(senv, fmt.Sprintf("%s=%s", key, value))
	}
	return
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

// b are []byte or io.Reader
func Open(b interface{}, progname string) (f *MemFile, err error) {
	return open(b, progname)
}

func OpenMemFd(b interface{}, name string) (*MemFile, error) {
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

// func ExecCommandCtxEnvTimeout(ctxc context.Context, name string, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	CmdHiddenConsole(cmd)
	return cmd
}

func CombineArgsAndStreams(args []string, stdoe ...io.Writer) []interface{} {
	interfaces := make([]interface{}, len(args)+len(stdoe))
	if len(args) != 0 {
		for i, arg := range args {
			interfaces[i] = arg
		}
	}
	if len(stdoe) != 0 {
		for i, std := range stdoe {
			interfaces[i] = std
		}
	}
	return interfaces
}

func CombineToSlideInterface(input ...interface{}) []interface{} {
	var result []interface{}
	for _, item := range input {
		if item == nil {
			continue
		}
		if reflect.TypeOf(item).Kind() == reflect.Slice {
			sliceValue := reflect.ValueOf(item)
			for i := 0; i < sliceValue.Len(); i++ {
				result = append(result, sliceValue.Index(i).Interface())
			}
		} else {
			result = append(result, item)
		}
	}
	return result
}

func CombineToSlideInterfaceNoFlat(input ...interface{}) []interface{} {
	var result []interface{}
	for _, item := range input {
		if item == nil {
			continue
		}
		result = append(result, item)
	}
	return result
}

type ExecWriter struct {
	io.Writer
}

func ToWriter(w interface{}) ExecWriter {
	if writer, ok := w.(io.Writer); ok {
		return ExecWriter{writer}
	}
	return ExecWriter{}
}
