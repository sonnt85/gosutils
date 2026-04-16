# gosutils

[![Go Reference](https://pkg.go.dev/badge/github.com/sonnt85/gosutils.svg)](https://pkg.go.dev/github.com/sonnt85/gosutils)

A large collection of Go utility sub-packages covering file operations, command execution, SSH, networking, cryptography, logging, and more.

## Installation

```bash
go get github.com/sonnt85/gosutils
```

## Features

- **sutils** — file I/O, path manipulation, JSON/XML parsing, HTTP download, email sending, AES encryption/decryption, process management, temp files, ID generation
- **sexec** — shell/command execution with timeouts, env vars, context support, in-memory binary execution, elevated (sudo) execution, process listing (`Pgrep`)
- **sconv** — string conversion: split strings into shell-word argument lists and join argument lists back into shell-quoted strings (POSIX quoting rules)
- **cmdshellwords** — shell-word splitting in the style of UNIX Bourne shell (`Split`, `SplitPosix`, `Join`, `Escape`)
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
- **vnc/vncproxy** — VNC client and proxy
- **gcurl** — HTTP client with digest auth, cookies, basic auth
- **goacl** — file ACL management (cross-platform)
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
- **service1** — alternative system service/daemon management (Linux/macOS)
- **goat** — AT command modem/GSM interface (APN configuration, SMS, TTY management)
- **gcauto** — automatic GC tuning: adjusts GOGC based on available memory at runtime
- **goxml** — XML document parsing and manipulation via element-tree abstraction
- **gogrep** — in-file and in-string grep utilities with regex support
- **gosed** — sed-like file/stream editor: regex substitution, line replacement
- **ethernet** — marshal/unmarshal IEEE 802.3 Ethernet II frames and 802.1Q VLAN tags
- **raw** — read/write raw network packets at the device driver level
- **arp** — ARP packet construction and resolution helpers
- **ssh_config** — SSH config file parser and writer (preserves comments)
- **websockify** — WebSocket-to-TCP bridge (WebSocket proxy for VNC/TCP services)
- **terminaldimensions** — get terminal width and height

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

### sconv
- `StringToArgs(input string) ([]string, error)` — split a shell-quoted string into argument tokens (supports single/double quotes and backslash escapes)
- `ArgsToString(args ...string) string` — join argument tokens into a properly shell-quoted string

### cmdshellwords
- `Split(line string) ([]string, error)` — split a line into tokens in the Bourne shell style
- `SplitPosix(line string) ([]string, error)` — POSIX-mode split
- `Join(words ...string) string` — join words into a shell-safe command line
- `JoinPosix(words ...string) string` — POSIX-mode join
- `Escape(str string) string` — escape a single string for safe unquoted shell use

### goat
- `ConfigAutoPort(b bool)` — enable/disable automatic AT port detection
- `ConfigApn(dev, apn, username, password string) error` — configure APN on a modem device
- `GetTTyAt(dev string) string` — resolve the AT command TTY path from a data port

### gcauto
- `Init()` — initialize automatic GC tuning based on total available memory
- `Tuning(threshold uint64)` — manually set the GC target heap size threshold

### goxml
- XML element-tree API: `Document`, `Element`, `Attr` types
- `ReadSettings` / `WriteSettings` for controlling parse/write behavior
- XPath-style element selection and manipulation

### gogrep
- `GrepFileLine(file, pattern string, n int, literal ...bool) ([]string, error)` — grep matching lines from file

### gosed
- `Sed(sedscript, filepath string) (bool, string, error)` — apply sed-like script to a file
- `FileReplaceRegex(pat, tostring, filepath string, literal ...bool) error` — regex substitution in file

### ethernet
- `Frame` type: marshal/unmarshal IEEE 802.3 Ethernet II frames
- `VLAN` type: marshal/unmarshal 802.1Q VLAN tags
- `EtherType` constants for common protocol types
- `Broadcast` hardware address constant

### raw
- `Conn` type implementing `net.PacketConn` for raw Ethernet frames
- `ListenPacket(ifi *net.Interface, proto uint16, cfg *Config) (*Conn, error)` — open raw socket on interface
- `Addr` type (`HardwareAddr`-based network address)

### arp
- `Client` type — ARP client bound to a network interface
- `Dial(ifi *net.Interface) (*Client, error)` — create ARP client on interface
- `New(ifi *net.Interface, p net.PacketConn) (*Client, error)` — create ARP client from existing connection
- `(*Client).Resolve(ip net.IP) (net.HardwareAddr, error)` — ARP resolve IP to MAC
- `(*Client).Request(ip net.IP) error` — send ARP request
- `(*Client).Reply(req *Packet, hwAddr net.HardwareAddr, ip net.IP) error` — send ARP reply
- `(*Client).Read() (*Packet, *ethernet.Frame, error)` — read incoming ARP packet
- `Packet` type with `NewPacket(op, srcHW, srcIP, dstHW, dstIP)` constructor

### ssh_config
- `Decode(r io.Reader) (*Config, error)` — parse SSH config from reader
- `Get(alias, key string) string` — get config value for host alias
- `GetStrict(alias, key string) (string, error)` — strict get with error
- `(*Config).Get/GetStrict(alias, key)` — instance methods

### websockify
- `New()` — start WebSocket-to-TCP proxy (reads `--source` and `--target` flags)

### terminaldimensions
- `Width() (int, error)` — terminal column count
- `Height() (int, error)` — terminal row count

## Author

**sonnt85** — [thanhson.rf@gmail.com](mailto:thanhson.rf@gmail.com)

## License

MIT License - see [LICENSE](LICENSE) for details.
