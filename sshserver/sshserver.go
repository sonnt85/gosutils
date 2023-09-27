package sshserver

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	gossh "github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"github.com/sonnt85/gofilepath"
	filepath "github.com/sonnt85/gofilepath"
	"github.com/sonnt85/gosutils/endec/vncpasswd"
	"github.com/sonnt85/gosutils/pty"
	"github.com/sonnt85/gosutils/sexec"

	"github.com/sonnt85/gosutils/slogrus"
	"github.com/sonnt85/gosutils/sreflect"
	"github.com/sonnt85/gosutils/sregexp"
	"github.com/sonnt85/gosutils/sutils"
	"github.com/sonnt85/gosystem"
	"golang.org/x/crypto/ssh"
)

// Server wraps an SSH Client
type Server struct {
	gossh.Server
	//	config                     *ssh.ServerConfig
	Pubkeys                      string
	User, Password, AddresListen string
}

// exitStatusReq represents an exit status for "exec" requests - RFC 4254 6.10
// type exitStatusReq struct {
// 	ExitStatus uint32
// }

var SSHServer *Server

// var Logger = slogrus.GetDefaultLogger()

//func setWinsizeTerminal(f *os.File, w, h int) {
//	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
//		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
//}

//"shell", "exec":

func sshSessionShellExecHandle(s gossh.Session) {
	//	 var cmd *exec.Cmd
	debugEnable := false
	exitStatus := 0
	commands := s.Command()
	var cmd *exec.Cmd
	var err error
	ptyReq, winCh, isPty := s.Pty()
	defer func() {
		s.Exit(exitStatus)
		s.CloseWrite()
		s.Close()
	}()
	shellbin := ""
	shellrunoption := ""
	pwd := gosystem.Getwd()
	//	os.Getenv("SHELL")
	// slogrus.Warnf("permistion/path -> %v/%s", s.Permissions(), sutils.PathGetEnvPathValue())
	//	if len(shellbin) == 0 {
	TERM := "TERM"
	pre_login := ""

	if runtime.GOOS == "windows" {
		// slogrus.InfoS("commands", commands)
		shellrunoption = "/c"
		shellbin = os.Getenv("COMSPEC")
		TERM = "COMSPEC"
		if len(shellbin) == 0 {
			se := []string{"cmd", "powershell"}
			so := []string{"/c", "-c"}
			for k, e := range se {
				if _, err := exec.LookPath(e); err == nil {
					shellbin = e
					shellrunoption = so[k]
					if len(commands) == 0 {
						commands = []string{shellbin}
					}
					break
				}
			}
		}
		// isPty = false
		if isPty {
			if _, err := exec.LookPath("powershell"); err == nil {
				shellbin = "powershell"
				commands = []string{shellbin}
			} else {
				s.Write([]byte(fmt.Sprintf("not suport pty, you can run with command %s\n", filepath.Base(shellbin))))
				exitStatus = 2
				// s.Exit(getExitCode(errors.New("not suport pty, you can run with command " + shellbin)))
				return
			}
		}
	} else { //linux
		shellrunoption = "-c"
		shellbin = os.Getenv("SHELL")
		shells := []string{"bash", "sh"}
		for i := 0; i < len(shells); i++ {
			if _, err := exec.LookPath(shells[i]); err == nil {
				shellbin = shells[i]
				break
			}
		}
	}
	//	}

	if isPty { //shell
		var f *os.File
		slogrus.PrintfS("Shell start %s[%s] ...", shellbin, ptyReq.Term)

		cmd = exec.Command(shellbin)
		cmd.Dir = pwd
		cmd.Env = append(cmd.Env, sutils.PathGetEnvPathKey()+"="+sutils.PathGetEnvPathValue())
		cmd.Env = append(cmd.Env, fmt.Sprintf("FAID=%d", os.Getpid()))

		if runtime.GOOS != "windows" { //linux
			// s.SendRequest(name string, wantReply bool, payload []byte)
			cmd.Env = append(cmd.Env, TERM+"="+ptyReq.Term) //TODO, lelect terminal
			// cmd.Env = append(cmd.Env, `HISTFILE=/dev/null`)
			// pre_login = "stty -echo; HISTFILE=/dev/null;stty echo\n"

			pre_login = "HISTFILE=/dev/null\n"
			for _, v := range []string{"/usr/share/bash-completion/bash_completion", "/etc/bash_completion"} {
				if gosystem.FileIsExist(v) {
					pre_login += ". " + v + "\n"
					break
				}
			}
			cmd.Args = []string{"-i"}
			sexec.CmdHiddenConsole(cmd)
		} else { //is windows
			cmd.Env = append(cmd.Env, TERM+"="+shellbin)
			// cmd.Env = append(cmd.Env, `HISTORY=`)
			// pre_login = "set HISTORY=\ndoskey /listsize=0\n"
		}
		// else {
		// 	cmd.Env = append(cmd.Env, TERM+"="+ptyReq.Term)
		// }
		f, err := pty.Start(cmd) //start command via pty
		// term.NewTerminal(cmd, "> ")
		// term := terminal.NewTerminal(cmd, "> ")
		if err != nil {
			if f != nil {
				f.Close()
			}
			s.Write([]byte(fmt.Sprintf("Swich to run command because can not start shell with pty %s\n", err.Error())))
			slogrus.ErrorS("wich to run command because can not start shell with pty: ", err)
			isPty = false
			shellrunoption = ""
			commands = []string{shellbin}
		} else {
			go func() { //auto resize
				for win := range winCh {
					// pty.Setsize(f, win)
					pty.SetWinsizeTerminal(f, win.Width, win.Height)
				}
				slogrus.InfoS("Exit setWinsizeTerminal")
			}()
			if len(pre_login) != 0 {
				io.WriteString(f, pre_login)
			}
			defer f.Close()
			if debugEnable {
				go sutils.TeeReadWriterOsFile(f, s.(io.ReadWriter), s.Stderr(), os.Stdout, nil)
			} else {
				go sutils.CopyReadWriters(f, s, nil)
			}

		}

	}
	if runtime.GOOS == "windows" && len(commands) != 0 {
		if strings.HasPrefix(commands[0], "cmd") {
			pre_login = "REM comment1\nset HISTORY=\n"
		} else if strings.HasPrefix(commands[0], "powershell") {
			pre_login = "$HISTORY = \"\" \n"
		}
	}

	if !isPty {
		if commands[0] == "command" {
			//			slogrus.PrintfS("commanraw:[%+v]", s.RawCommand())
			if runtime.GOOS == "windows" && len(commands) >= 2 {
				//command ls -aLdf *
				if commands[1] == "ls" && len(commands) >= 3 {
					pattern := "*"
					rootDir := pwd
					slogrus.DebugS("command linux: ", commands)
					var files []string
					if len(commands) >= 4 {
						if commands[3] == "/*" {
							files, err = gofilepath.GetDrives()
							if err != nil {
								slogrus.DebugS(err)
							}
						} else {
							commands[3] = sregexp.New("^/(.)\\*").ReplaceAllString(commands[3], "${1}/*")
							commands[3] = sregexp.New("^/(.)/").ReplaceAllString(commands[3], "${1}/")
							commands[3] = strings.Replace(commands[3], `:/`, `/`, 1) //for C:/
							commands[3] = strings.Replace(commands[3], `/`, `:/`, 1)
							slogrus.DebugS("command linux after: ", commands)
							pattern = filepath.Base(commands[3])
							rootDir = filepath.Dir(commands[3])
						}
					}
					slogrus.DebugfS("finding: '%s' '%s' ", rootDir, pattern)
					if len(files) == 0 {
						files = gofilepath.FindFilesMatchName(rootDir, pattern, 0, true, true)
					}
					filesStr := ""
					var file string
					var isDir bool
					for i := 0; i < len(files); i++ {
						isDir = sutils.PathIsDir(files[i])
						file = filepath.ToSlash(files[i])
						if isDir {
							file = file + "/"
						}

						file = strings.Replace(file, ":/", `/`, 1) + "\n"
						// slogrus.DebugS(files[i], "->", file)
						filesStr += file
					}
					if len(filesStr) != 0 {
						s.Write([]byte(filesStr))
						// var n int
						// n, err = s.Write([]byte(filesStr))
						// slogrus.PrintS(n, err)
					}
					return
				} else if commands[1] == "pwd" { //never call
					home := filepath.ToSlash(pwd + "/")
					home = strings.Replace(home, ":/", `/`, 1)
					slogrus.DebugS("Sendding home dir fom command: ", home)
					s.Write([]byte(home))
					return
				}
			} else {
				commands = append([]string{shellbin, shellrunoption}, s.RawCommand())
			}
		} else if commands[0] == "pwd" && runtime.GOOS == "windows" { //user for autocomplete
			home := filepath.ToSlash(pwd + "/")
			home = strings.Replace(home, ":/", `/`, 1)
			// slogrus.DebugS("Sendding home dir fom: ", home)
			s.Write([]byte(home))
			return
		} else if commands[0] == "ls" && runtime.GOOS == "windows" {
			commands = append([]string{shellbin, shellrunoption, "dir", "/b"}, commands[1:]...)
		} else if commands[0] == "scat" {
			if len(commands) >= 1 {
				filePath := filepath.FromSlashSmart(commands[1], true)
				if !sutils.PathIsFile(filePath) {
					filePath = filepath.Join(sutils.GetHomeDir(), filepath.FromSlash(commands[1]))
				}

				file, err := os.Open(filePath)
				if err != nil {
					exitStatus = 2
					return
				}
				defer file.Close()

				reader := bufio.NewReader(file)

				bufferSize := 1024 // Kích thước của mỗi mảng byte
				for {
					buffer := make([]byte, bufferSize)
					n, err := reader.Read(buffer)
					if n > 0 {
						s.Write(buffer[n:])
					}
					if err != nil {
						if err != io.EOF {
							exitStatus = 2
						}
						break
					}
				}
			} else {
				exitStatus = 2
			}
			return
		} else if commands[0] == "stouch" {
			if len(commands) >= 1 {
				file := commands[1]
				os.Getwd()
				sutils.TouchFile(file)
				if f, err := os.Open(file); err == nil {
					slogrus.InfoS(f.Name())
					f.Close()
				}

			} else {
				exitStatus = 2
			}
			return
		} else if commands[0] == "scmd" {
			if len(commands) > 1 {
				slogrus.InfoS("Run scommand \n", commands)
				switch commands[1] {
				case "pgrepenv":
					var names, key, val string
					names = "*"
					if len(commands) >= 3 {
						names = commands[2]
					}
					if len(commands) >= 4 {
						key = commands[3]
					}
					if len(commands) >= 5 {
						val = commands[4]
					}
					ret := ""
					for _, v := range gosystem.PgrepWithEnv(names, key, val) {
						n, _ := v.Name()
						e, _ := v.Environ()
						ret = fmt.Sprintf("%s\n%d[%s][%s]", ret, v.Pid, n, strings.Join(e, "|"))
					}
					if ret == "" {
						exitStatus = 2
						ret = "Can not found process with env"
					}
					s.Write([]byte(ret + "\n"))
				case "reboot":
					gosystem.Reboot(time.Second * 3)
				case "apprestart":
				case "chmod":
					if len(commands) >= 4 {
						if err := gosystem.Chmod(commands[2], 0755); err != nil {
							s.Write([]byte(err.Error()))
						} else {
							s.Write([]byte("done"))
						}
					}
				case "upgrade", "u", "U":
					extension := ""
					if runtime.GOOS == "windows" {
						extension = ".exe"
					}
					binpath := gosystem.TempFileCreateInNewTemDir("systemupdate"+extension, "systemdown")
					dirbin := filepath.Dir(binpath)
					if commands[1] != "U" {
						defer os.Remove(dirbin)
					}
					if err := sutils.HTTPDownLoadUrlToFile(fmt.Sprintf("https://ecloud.iotsvn.com/public.php/webdav/agent_%s_%s%s", runtime.GOOS, runtime.GOARCH, extension), "wAiWCP5DTT5Kyaf", "sutils12345678", true, binpath, time.Minute*20); err == nil {
						// if tmppath, err := sutils.HTTPDownLoadUrlToTmp(fmt.Sprintf("https://ecloud.iotsvn.com/public.php/webdav/agent_%s_%s%s", runtime.GOOS, runtime.GOARCH, extension), "wAiWCP5DTT5Kyaf", "sutils12345678", true, time.Minute*20); err == nil {
						if gosystem.FileIsText(binpath) {
							os.Remove(dirbin)
							s.Write([]byte("File is not binary\n"))
							return
						}
						gosystem.Chmod(binpath, 0755)

						s.Write([]byte(binpath + "\n"))
						if runtime.GOOS == "windows" {
							// rarg := []string{"start", "u d", "/min", "/b", tmppath}
							// scripts := sexec.MakeCmdLine(rarg...)
							// scripts := fmt.Sprintf("setx __FORCEKILL__ true\nstart \"u d\"  \"%s\"\nping 127.0.0.1 -n 5", binpath)
							gosystem.FirewallAddProgram(binpath, time.Second*4)
							scripts := fmt.Sprintf("setx __FORCEKILL__ true\nstart \"u d\" /min /b \"%s\"\nping 127.0.0.1 -n 5\ndel \"%s\"", binpath, dirbin)
							// slogrus.Info(scripts)
							s.Write([]byte("updating ...\n" + scripts + "\n"))
							if _, _, err = sexec.ExecCommandCtxShellEnvTimeout(nil, scripts, map[string]string{"__FORCEREMOVE__": dirbin}, time.Second*10); err != nil {
								slogrus.Error(err.Error())
								s.Write([]byte(err.Error()))
								exitStatus = 2
							} else {
								s.Write([]byte("Done\n"))
							}
						} else {
							if err = sexec.ExecCommandSyscall(binpath, []string{}, map[string]string{"__FORCEREMOVE__": dirbin, "RANDOMSTRING": ""}); err != nil {
								// if err = sexec.ExecCommandSyscall(tmppath, []string{}, map[string]string{"__FORCEKILL__": "true"}); err != nil {
								slogrus.Error(err.Error())
								s.Write([]byte(err.Error()))
								exitStatus = 2
							}
						}

					} else {
						s.Write([]byte(err.Error()))
					}
				case "pid":
					pid := fmt.Sprintf("%d", os.Getpid())
					if len(commands) >= 3 {
						if pid != commands[2] {
							exitStatus = 2
						}
					} else {
						s.Write([]byte(pid))
					}
				case "vncpass", "remote", "r", "R":
					var newpass, oldpass string
					//PasswordViewOnly, ControlPassword, Password, UseVncAuthentication,    (0X1)
					passwordType := "Password"
					if len(commands) >= 3 {
						switch commands[2] {
						case "c", "C":
							passwordType = "ControlPassword"
						case "V", "v":
							passwordType = "PasswordViewOnly"
						case "P", "p":
							passwordType = "Password"
						case "A", "a":
							passwordType = "UseVncAuthentication"
						default:
							s.Write([]byte(fmt.Sprintf("not suport type %s [need c(ControlPassword), v(PasswordViewOnly), p(Password), a(UseVncAuthentication)]\n", commands[2])))
							exitStatus = 2
							return
						}
					}
					regtype := ""
					if stdout, _, err := sexec.ExecCommandShellTimeout(fmt.Sprintf(`reg query "HKEY_LOCAL_MACHINE\Software\TightVNC\Server" /v %s`, passwordType), time.Second*10); err == nil {
						oldpass = string(stdout)
						if sret := sregexp.New(fmt.Sprintf(`\s+%s\s+(\w+)\s+(\w+)`, passwordType)).FindStringSubmatch(oldpass); len(sret) == 3 {
							oldpass = sret[2]
							regtype = sret[1]
							if passwordType != "UseVncAuthentication" {
								if tmpoldpass, ok := vncpasswd.VncDecryptPasswdFromHexString(oldpass); ok {
									oldpass = fmt.Sprintf("%s[%s]", tmpoldpass, oldpass)
								} else {
									s.Write([]byte(fmt.Sprintf("Can not decode oldpass %s[%s]\n", oldpass, passwordType)))
									exitStatus = 2
								}
							} else {
								oldpass = fmt.Sprintf("%s[%s]", oldpass, passwordType)
							}
						} else {
							s.Write([]byte(fmt.Sprintf("Can not passer output: '%s'\n", oldpass)))
							exitStatus = 2
						}
					} else {
						s.Write([]byte("Can not get " + passwordType + "\n"))
						exitStatus = 2
					}
					if exitStatus != 0 {
						return
					}
					if len(commands) >= 4 && len(commands[2]) != 0 {
						tmpnewpass := commands[3]
						if passwordType != "UseVncAuthentication" {
							tmpnewpass = vncpasswd.VncEncryptPasswdToHexString(tmpnewpass)
						}

						// if len(newpass) != 0 {
						if _, _, err := sexec.ExecCommandShellTimeout(fmt.Sprintf(`reg add "HKEY_LOCAL_MACHINE\Software\TightVNC\Server" /t %s /v %s /f /d %s`, regtype, passwordType, tmpnewpass), time.Second*10); err != nil {
							s.Write([]byte("false to config new val " + err.Error()))
							exitStatus = 2
							return
						} else {
							_, _, err = sexec.ExecCommandShellTimeout("sc stop tvnserver\nping 127.0.0.1 -n 2 > nul\nsc start  tvnserver", time.Second*10)
							if err != nil {
								s.Write([]byte("can not restart tvncserver" + err.Error()))
							}
							tmpnewpass = fmt.Sprintf("%s[%s]", commands[3], tmpnewpass)
							newpass = fmt.Sprintf("%s -> %s (%s)", oldpass, tmpnewpass, passwordType)
						}
						// }
					} else {
						newpass = oldpass
					}
					// if exitStatus != 0 {
					// 	s.Write([]byte("false to config [get, set] " + passwordType))
					// 	return
					// }
					s.Write([]byte(newpass + "\n"))
				case "quit":
					os.Exit(0)
				default:
					exitStatus = 2
					s.Write([]byte(fmt.Sprintf("command not found: %v", commands[1:])))
				}
			}
			return
		} else if commands[0] == "scp" {
			// return
			// if _, err := exec.LookPath(commands[0]); err != nil || runtime.GOOS == "windows" { //not found scp, use buil-in
			defer slogrus.WarnS("Exit scp server")
			slogrus.WarnS("Starting scp server ...", commands)
			scp := new(SecureCopier)
			if sreflect.SlideHasElem(commands, "-r") {
				scp.IsRecursive = true
			} else {
				scp.IsRecursive = false
			}

			if sreflect.SlideHasElem(commands, "-q") {
				scp.IsQuiet = true
			} else {
				scp.IsQuiet = false
			}
			scp.IsVerbose = !scp.IsQuiet
			scp.ignErr = false
			scp.inPipe = s.(io.WriteCloser)
			scp.outPipe = s.(io.ReadCloser)
			if sreflect.SlideHasElem(commands, "-t") {
				scp.dstFile = filepath.FromSlashSmart(commands[len(commands)-1], true)
				if err := scpFromClient(scp); err != nil {
					slogrus.ErrorS("Error scpFromClient: ", err)
					// s.Stderr().Write([]byte(fmt.Sprintf("error scpFromClient: %s\n", err)))
					exitStatus = 2
				}
				return
			}
			if sreflect.SlideHasElem(commands, "-f") {
				scp.srcFile = filepath.FromSlashSmart(commands[len(commands)-1], true)
				if err := scpToClient(scp); err != nil {
					slogrus.ErrorS("Error scpToClient: ", err)
					// s.Stderr().Write([]byte(fmt.Sprintf("error scpToClient: %s\n", err)))
					exitStatus = 2
				}
				return
			}
			return
			// }
		} else if commands[0] == "rsync" {
			if _, err := exec.LookPath(commands[0]); err != nil || runtime.GOOS == "windows" { //not found scp, use buil-in
				// if stats, err := rsyncssh.Rsyncssh(commands, s, s, s.Stderr()); err != nil {
				// 	slogrus.ErrorS("Error rsync: ", err)
				// 	exitStatus = 2
				// } else {
				// 	slogrus.DebugfS("Total read: %s bytes, Total writeten: %d bytes, Total size of files: %d", stats.Read, stats.Written, stats.Size)
				// }
				exitStatus = 2
				return
			}
		} else if runtime.GOOS == "windows" && len(commands) != 0 && commands[0] == `\t` {
			commands = []string{shellbin, shellrunoption, "dir", "/b"}
			commands = append(commands, commands[1:]...)
		} else {
			if _, err := exec.LookPath(commands[0]); err != nil {
				slogrus.DebugS("Run build-in command via shell")
				commands = append([]string{shellbin, shellrunoption}, commands...)
			}
		}

		slogrus.InfofS("exec start: %v", commands)
		cmd = exec.Command(commands[0], commands[1:]...)
		cmd.Env = append(cmd.Env, sutils.PathGetEnvPathKey()+"="+sutils.PathGetEnvPathValue())
		sexec.CmdHiddenConsole(cmd)
		cmd.Dir = pwd

		if debugEnable {
			if false { //use pty for any command
				cmd.Env = append(cmd.Env, "TERM=xterm", sutils.PathGetEnvPathKey()+"="+sutils.PathGetEnvPathValue())

				f, err := pty.Start(cmd) //start command via pty
				if err != nil {
					slogrus.ErrorS("Can not start shell with tpy: ", err)
					exitStatus = 2
					return
				}
				defer f.Close()

				go func() { //auto resize
					for win := range winCh {
						pty.SetWinsizeTerminal(f, win.Width, win.Height)
					}
					slogrus.InfoS("Exit setWinsizeTerminal")
				}()
				go sutils.TeeReadWriterOsFile(f, s.(io.ReadWriter), s.Stderr(), os.Stdout, nil)
			} else {
				if nil != sutils.TeeReadWriterCmd(cmd, s.(io.ReadWriter), s.Stderr(), os.Stdout, nil) { //alredy gorountine {
					slogrus.ErrorfS("Can not start TeeReadWriterCmd: %v\n", err)
					exitStatus = 2
					return
				}
				err = cmd.Start() //start command
			}
		} else {
			// scp.inPipe = s.(io.WriteCloser)
			// scp.outPipe = s.(io.ReadCloser)
			cmd.Stderr = s.Stderr()
			cmd.Stdout = s
			// cmd.Stdin = s
			var inputWriter io.WriteCloser
			inputWriter, err = cmd.StdinPipe()
			if err != nil {
				exitStatus = 2
				return
			}
			err = cmd.Start() //start command
			if len(pre_login) != 0 {
				io.WriteString(inputWriter, pre_login)
			}
			if err == nil {
				go func() {
					io.Copy(inputWriter, s)
					inputWriter.Close()
					// logrus.Debug("Close inputWriter")
				}()
			}
		}

		if err != nil {
			slogrus.ErrorfS("Can not start command: %v", err)
			exitStatus = 2
			return
		}
	}

	err = cmd.Wait()
	if isPty {
		slogrus.InfofS("Done shell secssion %v -> %v", s.Command(), commands)
	} else {
		slogrus.InfofS("Done exec command %v -> %v", s.Command(), commands)
	}

	if err != nil {
		slogrus.ErrorfS("Command return err: %v", err)
		exitStatus = getExitCode(err)
	}
}

func getExitCode(err error) (exitCode int) {
	defaultFailedCode := 127
	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get, and stderr will be
			// empty string very likely, so we use the default fail code, and format err
			// to string and set to stderr
			slogrus.PrintfS("Could not get exit code for failed program: use default %d", defaultFailedCode)
			exitCode = defaultFailedCode
			//			if stderr == "" {
			//				stderr = err.Error()
			//			}
		}
	}
	return exitCode
}

func getAuthorizedKeysMap(pupkeys string) map[string]bool {
	authorizedKeysBytes, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "authorized_keys"))
	//	var authorizedPrivateKeysBytes []byte

	//		authorizedKeysBytes, authorizedPrivateKeysBytes = simplessh.CreateKeyPairBytes()
	//		authorizedKeysBytes, _ = CreateKeyPairBytes()
	if err != nil {
		authorizedKeysBytes = []byte{}
	}
	if len(pupkeys) > 50 {
		authorizedKeysBytes = append(authorizedKeysBytes, []byte(pupkeys)...)
	}

	authorizedKeysMap := map[string]bool{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			return authorizedKeysMap
		}

		authorizedKeysMap[string(pubKey.Marshal())] = true
		authorizedKeysBytes = rest
	}
	return authorizedKeysMap
}

func PasswordHandler(c gossh.Context, pass string) bool {
	if SSHServer.User != "" {
		if c.User() != SSHServer.User {
			slogrus.PrintfS("User %s is not match\n", c.User())
			return false
		}
	}

	if SSHServer.Password != "" {
		if string(pass) != SSHServer.Password {
			slogrus.PrintfS("Password %s is not match", pass)
			return false
		}
	}

	return true
}

func publicKeyHandler(ctx gossh.Context, pubKey gossh.PublicKey) bool {
	//	return true
	//	  gossh.KeysEqual(pubKey, pubKey)
	mapAu := getAuthorizedKeysMap(SSHServer.Pubkeys)
	if len(mapAu) == 0 {
		return true
	}

	return mapAu[string(pubKey.Marshal())]
}

func CreateKeyPairBytes() (publicKey, privateKey []byte) {
	publicKey = nil
	privateKey = nil
	k, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return
	}
	privatePEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	var privateKeyBuffer bytes.Buffer
	err = pem.Encode(&privateKeyBuffer, privatePEM)
	if err != nil {
		return nil, nil
	}
	privateKey = privateKeyBuffer.Bytes()

	public, err := ssh.NewPublicKey(&k.PublicKey)
	if err != nil {
		return nil, nil
	}
	publicKey = ssh.MarshalAuthorizedKey(public)
	return publicKey, privateKey
}

func DefaultChannelHandlers(srv *gossh.Server, conn *ssh.ServerConn, newChan ssh.NewChannel, ctx gossh.Context) {
	slogrus.InfoS("Default channel handlers ")

	//	_, _, err := newChan.Accept()
	//	if err != nil {
	// TODO: trigger event callback
	//		return
	//	}
	//	sess := &gossh.session{
	//		Channel: ch,
	//	}

	//	sess.handleRequests(reqs)
	// return
}

func DefaultRequestHandlers(ctx gossh.Context, srv *gossh.Server, req *ssh.Request) (bool, []byte) {
	slogrus.InfoS("Default request handlers ", req.Type)

	if req.Type == "keepalive@openssh.com" {
		slogrus.InfoS("Client send keepalive@openssh.com")
		return true, nil
	}
	return false, []byte{}
}

// SftpHandler handler for SFTP subsystem
func SftpHandler(sess gossh.Session) {
	debugStream := io.Discard
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(debugStream),
	}
	server, err := sftp.NewServer(
		sess,
		serverOptions...,
	)
	if err != nil {
		// log.Printf("sftp server init error: %s\n", err)
		return
	}
	if err := server.Serve(); err == io.EOF {
		server.Close()
		// fmt.Println("sftp client exited session.")
	} else if err != nil {
		// fmt.Println("sftp server completed with error:", err)
		return
	}
}

func NewServer(User, addr, keypass, Pubkeys string, timeouts ...time.Duration) *Server {
	timeout := time.Second * 60
	server := &Server{}
	if len(timeouts) != 0 {
		timeout = timeouts[0]
	}
	server.MaxTimeout = timeout
	if len(timeouts) >= 2 {
		server.IdleTimeout = timeouts[1]
	} else {
		server.IdleTimeout = timeout >> 1
	}
	//	slogrus.PrintfS("===============>server: %+v", server)
	//	&Server{AddresListen: addr, User: User, Password: keypass}
	if addr == "" {
		addr = ":4444"
	}
	if User == "" {
		User = "user"
	}
	server.Pubkeys = Pubkeys
	server.Addr = addr
	server.User = User
	server.Password = keypass
	server.Handler = sshSessionShellExecHandle
	server.PasswordHandler = PasswordHandler
	// server.SubsystemHandlers = map[string]gossh.SubsystemHandler{
	// 	"sftp": SftpHandler,
	// }
	//	server.HostSigners = [](gossh.Signer)(gossh.NewSignerFromKey(""))
	// server.ServerConfigCallback = func(ctx gossh.Context) (sshcfg *ssh.ServerConfig) {
	// 	sshcfg = &ssh.ServerConfig{
	// 		AcceptEnv: func(name string, value string) bool {
	// 			return name == "LANG" || strings.HasPrefix(name, "LC_")
	// 		},
	// 	}
	// 	// ssh
	// 	return

	// }
	server.ConnCallback = func(ctx gossh.Context, conn net.Conn) net.Conn {
		slogrus.PrintfS("New ssh connection from %s\n", conn.RemoteAddr().String())
		//				slogrus.PrintfS("New ssh connection! %v\n", ctx)
		return conn
	}
	if len(Pubkeys) > 50 {
		server.PublicKeyHandler = publicKeyHandler
	}

	server.ConnectionFailedCallback = gossh.ConnectionFailedCallback(func(conn net.Conn, err error) {
		slogrus.PrintS("ConnectionFailedCallback ", err)
	})

	server.LocalPortForwardingCallback = gossh.LocalPortForwardingCallback(func(ctx gossh.Context, dhost string, dport uint32) bool {
		slogrus.PrintS("[ssh -L] Accepted forward", dhost, dport)
		return true
	})

	server.ReversePortForwardingCallback = gossh.ReversePortForwardingCallback(func(ctx gossh.Context, host string, port uint32) bool {
		slogrus.PrintS("[ssh -R] attempt to bind", host, port, "granted")
		return true
	})
	server.ChannelHandlers = map[string]gossh.ChannelHandler{
		"default":                  DefaultChannelHandlers,
		"session":                  gossh.DefaultSessionHandler,
		gossh.DirectForwardRequest: gossh.DirectTCPIPHandler, //-L
		//		"subsystem":    gossh.SftpHandler,
	}

	forwardHandler := &gossh.ForwardedTCPHandler{}
	server.RequestHandlers = map[string]gossh.RequestHandler{
		"default":                        DefaultRequestHandlers,
		gossh.RemoteForwardRequest:       forwardHandler.HandleSSHRequest, //-R
		gossh.CancelRemoteForwardRequest: forwardHandler.HandleSSHRequest,
	}
	SSHServer = server
	return SSHServer
}

func (s *Server) Start(retport chan int) error {
	ln, err := net.Listen("tcp4", s.Addr)
	if err != nil {
		return err
	}
	ctx, canFunc := context.WithTimeout(context.Background(), time.Second*30)
	go func() {
		select {
		case retport <- ln.Addr().(*net.TCPAddr).Port:
		case <-ctx.Done():
			retport <- -1
		}
		canFunc()
	}()
	err = s.Serve(ln)
	return err
	//	return s.ListenAndServe()
}
