# gosutils

A large collection of Go utility sub-packages covering file operations, command execution, SSH, networking, cryptography, logging, and more.

## Installation

```bash
go get github.com/sonnt85/gosutils
```

## Features

- **sutils** — file I/O, path manipulation, JSON/XML parsing, HTTP download, email sending, AES encryption/decryption, process management, temp files, ID generation
- **sexec** — shell/command execution with timeouts, env vars, context support, in-memory binary execution, elevated (sudo) execution, process listing (`Pgrep`)
- **sconv** — shell-word splitting (POSIX quoting rules)
- **sregexp** — cached/lazy-compiled `*regexp.Regexp` wrapper
- **endec** — AES-GCM encryption, random byte/string generation, base64 helpers
- **cron** — simple in-process cron scheduler (minute/hour/day/weekday granularity)
- **progressbar** — terminal progress bar with percentage and transfer rate
- **sshclient** — SSH client with SCP/SFTP support and terminal forwarding
- **sshserver** — SSH server with SCP handler
- **simplessh** — high-level SSH client with local port forwarding
- **sexec** — context-aware command execution helpers
- **scanport** — TCP port scanner
- **sched** — goroutine pool scheduler
- **pty** — PTY (pseudo-terminal) allocation
- **structs** — reflection-based struct field manipulation
- **sreflect** — Go reflection utilities
- **sintersect** — slice intersection/union helpers
- **smerge** — deep merge of structs/maps
- **slogrus** — logrus formatter and file-rotation hooks
- **vnc/vncproxy/websockify** — VNC client, proxy, and WebSocket bridge
- **gcurl** — HTTP client with digest auth, cookies, basic auth
- **goacl** — file ACL management (cross-platform)
- **gogrep** — in-memory grep utilities
- **gosed** — sed-like stream editor engine
- **goexpect** — expect-style process interaction (spawn/match/send)
- **bufcopy** — buffered I/O copy helpers
- **lockedfile** — file locking with timeout
- **sembed** — `embed.FS` sub-filesystem helper
- **ppjson** — pretty-print JSON
- **funcmap** — dynamic function registry (string → func)
- **hashmap** — concurrent int64 and empty hash maps
- **malloc** — low-level memory allocation helpers
- **osext** — executable path resolution (cross-platform)
- **runonce** — run-once / singleton helpers
- **sendxmpp** — XMPP message and file sender
- **service** — system service (daemon) management

## Usage

```go
import (
    "github.com/sonnt85/gosutils/sutils"
    "github.com/sonnt85/gosutils/sexec"
    "github.com/sonnt85/gosutils/endec"
)

// File operations
data, _ := sutils.File2lines("/etc/hosts")
sutils.FileCopy("/src/file", "/dst/file")

// Execute a shell command with timeout
stdout, stderr, err := sexec.ExecCommandShellTimeout("ls -la /tmp", 5*time.Second)

// AES-GCM encryption
key := []byte("0123456789abcdef")
cipher, _ := endec.AESGCMEncrypt(key, []byte("secret data"))
plain, _ := endec.AESGCMDecrypt(key, cipher)

// JSON path query
result, _ := sutils.JsonStringFindElement(&jsonStr, "user.name")
sutils.JsonSet(jsonStr, "user.age", 30)
```

## API

### sutils
- `File2lines(path) ([]string, error)` — read file into string slice
- `FileCopy(src, dst) (int64, error)` — copy file
- `FileHashMd5(path) (string, error)` — MD5 hash of file
- `FileUpdateOrAdd(path, contents, grep, pattern)` — update or append lines in file
- `PathIsExist/PathIsFile/PathIsDir(path)` — path existence checks
- `TempFileCreate/TempFileCreateWithContent(...)` — create temp files
- `HTTPDownLoadUrl(url, method, user, pass, insecure)` — HTTP download to bytes
- `JsonStringFindElement/JsonSet/JsonDelete(...)` — JSON path get/set/delete
- `XmlStringFindElement(...)` — XPath query on XML strings
- `StringEncrypt/StringDecrypt(str, key)` — AES string encryption
- `EmailSend(smtphost, user, pass, from, subject, to, msg, attach)` — send email
- `IDGenerate()` — generate unique ID (xid)
- `IsContainer()` — detect if running inside a container

### sexec
- `ExecCommand(name, args...)` — run command, return stdout/stderr
- `ExecCommandShell(script, args...)` — run via shell
- `ExecCommandShellTimeout(script, timeout)` — run with timeout
- `ExecCommandEnvTimeout(name, envs, timeout)` — run with custom env
- `ExecBytesEnvTimeout(binary, name, envs, timeout)` — run in-memory binary
- `Pgrep(names...)` — find processes by name
- `ExecCommandCtx*(...)` — context-aware variants of all exec functions

### endec
- `AESGCMEncrypt/AESGCMDecrypt(key, data)` — AES-GCM encrypt/decrypt
- `EncrypBytesToString/DecryptBytesFromString(data, password)` — password-based encryption
- `GenerateRandomBytes/GenerateRandomString(n)` — cryptographically random data

### sregexp
- `New(pattern) *Regexp` — cached regexp compilation
- Methods: `FindString`, `FindAllString`, `MatchString`, `ReplaceAllString`, etc.

## Author

**sonnt85** — [thanhson.rf@gmail.com](mailto:thanhson.rf@gmail.com)

## License

MIT License - see [LICENSE](LICENSE) for details.
