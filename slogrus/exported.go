package slogrus

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/mattn/go-colorable"
	"github.com/sonnt85/gosystem"

	// "github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
)

var defaultTimestampFormat = "2006-01-02T15:04:05.000Z07:00"

// Log Level 0->panic; 1->fatal, 2->error, 3->warn, 4->info, 5->debug, 6->trace
// type Level uint32
type Hook = logrus.Hook
type Slog struct {
	*logrus.Logger
	// rh      *RotateFileHook
	initted bool
}

type Entry struct {
	*logrus.Entry
}

func (slog Slog) WriteStd(v ...any) {
	slog.Out.Write([]byte(fmt.Sprint(v...)))
}

func (slog Slog) WritefStd(format string, v ...any) {
	slog.Out.Write([]byte(fmt.Sprintf(format, v...)))
}

var stdSlog = &Slog{
	Logger: logrus.StandardLogger(),
}

// var std = logrus.StandardLogger()

func New(writer io.Writer) *Slog {
	slog := &Slog{
		Logger: logrus.New(),
	}
	slog.Out = writer
	return slog
}

// new slog file with default stdout is os.Stderr
func NewLogFile(logPath string, log_level Level, pretty bool, diableStdout bool, logpath ...any) *Slog {
	slog := &Slog{
		Logger: logrus.New(),
	}
	initDefaultLog(slog, log_level, pretty, diableStdout, logpath...)
	return slog
}

func Stack(skipn int) (stackResult string) {
	// stack := make([]byte, 2048)
	// length := runtime.Stack(stack, false)
	stackString := string(debug.Stack())
	stackLines := strings.Split(stackString, "\n")
	if len(stackLines) >= skipn {
		stackLines = stackLines[skipn:]
	} else {
		stackLines = []string{}
	}

	stackResult = strings.Join(stackLines, "\n")
	return
}

// Number 11 may change
func traceStackSkip(msg ...any) (retmsg []any) {
	stackString := string(debug.Stack())
	stackLines := strings.Split(stackString, "\n")
	skipStackLines := make([]string, 0)
	found := false
	for i, l := range stackLines {
		if strings.Contains(l, "gosutils/slogrus") || strings.Contains(l, "runtime/debug") {
			found = true
			continue
		}
		if found {
			skipStackLines = append(skipStackLines, stackLines[i:]...)
			break
		} else {
			skipStackLines = append(skipStackLines, l)
		}
	}
	return append(msg, strings.Join(skipStackLines, "\n"))
}

func (slog *Slog) TraceStack(msg ...any) {
	nmsg := traceStackSkip(msg...)
	slog.Trace(nmsg...)
}

func (slog *Slog) DisableStd(msg ...any) {
	slog.Out = io.Discard
}

func (slog *Slog) TracefStack(format string, args ...any) {
	args = traceStackSkip(args...)
	format = format + "[%s]"
	slog.Tracef(format, args...)
}

func TracefStack(format string, args ...any) {
	args = traceStackSkip(args...)
	format = format + "[%s]"
	stdSlog.Tracef(format, args...)
}

var TracefStackS = TracefStack

func TraceStack(msg ...any) {
	nmsg := traceStackSkip(msg...)
	stdSlog.Trace(nmsg...)
}

var TraceStackS = TraceStack

func (slog *Slog) GetOldLogFiles() (retpaths []string) {
	if !slog.initted {
		return
	}
	for _, hooks := range slog.Hooks {
		for _, hook := range hooks {
			if lgr, ok := hook.(*RotateFileHook); ok {
				if lg, ok := lgr.logWriter.(*LoggerRotate); ok {
					retpaths = append(retpaths, lg.GetOldLogFiles()...)
				}
			}
		}
	}
	// if lgr, ok := slog.rh.logWriter.(*LoggerRotate); ok {
	// 	return lgr.GetOldLogFiles()
	// }
	return
}

func (slog *Slog) Flush() {
	// if !slog.initted || slog.rh == nil {
	if !slog.initted {
		return
	}
	for _, hooks := range slog.Hooks {
		for _, hook := range hooks {
			if lgr, ok := hook.(*RotateFileHook); ok {
				if lg, ok := lgr.logWriter.(*LoggerRotate); ok {
					lg.buff.WaitUntilEmpty()
					lg.close()
				}
			}
		}
	}
	// if lgr, ok := slog.rh.logWriter.(*LoggerRotate); ok {
	// 	lgr.buff.WaitUntilEmpty()
	// 	lgr.close()
	// }
}

func Flush() {
	stdSlog.Flush()
}

func ColorStd() {
	// colorable.NewColorableStderr().Write(p []byte)
	colorable.EnableColorsStdout(nil)()
}

func DisableStd() {
	os.Stdout, _ = os.Open(os.DevNull)
	os.Stderr, _ = os.Open(os.DevNull)
}

func DisableOutput() {
	stdSlog.SetOutput(io.Discard)
}

// logPath string, log_level logrus, pretty bool)
func initDefaultLog(slog *Slog, log_level Level, pretty bool, diableStdout bool, logpaths ...any) {
	if slog.initted {
		return
	}
	slog.initted = true
	logpath := ""
	disableMsgJsonOpject := false
	for _, x := range logpaths {
		switch v := x.(type) {
		case string:
			logpath = v
		case bool:
			disableMsgJsonOpject = v
		}
	}
	slogOutFile, outputIsOsFile := slog.Out.(*os.File)
	if diableStdout {
		slog.SetOutput(io.Discard)
		if stdSlog == slog { //auto disable os.Stdout if is standard log
			DisableStd()
		}
	} else {
		if outputIsOsFile {
			slog.SetOutput(colorable.NewColorable(slogOutFile))
			// colorable.EnableColorsStdout(nil)()
		}
	}

	// timeFormat := time.RFC3339 //"2006-01-02T15:04:05Z07:00"
	timeFormat := defaultTimestampFormat //milisecond
	orderKeys := os.Getenv("SLOGRUS_ORDER_KEYS")
	var orderArrayKeys []string
	if orderKeys != "" {
		orderArrayKeys = strings.Split(orderKeys, ",")
		if len(orderArrayKeys) != 0 {
			for i := range orderArrayKeys {
				orderArrayKeys[i] = strings.TrimSpace(orderArrayKeys[i])
			}
		} else {
			orderArrayKeys = []string{
				FieldKeyTime,
				FieldKeyLevel,
				FieldKeyMsg,
				FieldKeyFunc,
				FieldKeyFile,
			}
		}
	}
	rootDir := os.Getenv("SLOGRUS_ORDER_ROOT_DIR")
	baseNameOnly := false
	if os.Getenv("SLOGRUS_BASENAMEONLY") != "" {
		if b, e := strconv.ParseBool(os.Getenv("SLOGRUS_BASENAMEONLY")); e == nil {
			baseNameOnly = b
		}
	}
	logJsonFormatter := &JSONFormatter{
		// logJsonFormatter := &logrus.JSONFormatter{
		TimestampFormat:      timeFormat,
		PrettyPrint:          pretty,
		DisableHTMLEscape:    true,
		DisableMsgJsonOpject: disableMsgJsonOpject,
		ReorderArrayKeys:     orderArrayKeys,
	}

	logRuntimeFormatter := &FormatterRuntime{
		ChildFormatter: logJsonFormatter,
		File:           true,
		Line:           true,
		Package:        false,
		BaseNameOnly:   baseNameOnly,
		RootDir:        rootDir,
		// TextToSearchFun: "gosutils.slogrus.",
	}
	slog.SetLevel(log_level)
	if stdSlog == slog { //print to stdout standard, auto disable output if not is terminal
		if !diableStdout {
			if outputIsOsFile && gosystem.IsTerminal(slogOutFile.Fd()) {
				// fmt.Println("Is Terminal")

				// if gosystem.IsTerminalWriter(stdSlog.Out) || (os.Getenv(DEBUGENVNAME) == "true" && runtime.GOOS == "windows" || gosystem.IsTerminal(os.Stderr.Fd())) {
				// if gosystem.IsTerminalWriter(stdSlog.Out) {
				// if (ok && (isatty.IsTerminal(fileprr.Fd()) || isatty.IsCygwinTerminal(fileprr.Fd()))) || (isatty.IsTerminal(stdoutFD) || isatty.IsCygwinTerminal(stdoutFD)) {
				// fmt.Println("Is Terminal")
				// logStdStandardRuntimeFormatter := *logRuntimeFormatter
				logTextFormatter := &logrus.TextFormatter{
					TimestampFormat: timeFormat,
					FullTimestamp:   true,
					ForceColors:     true,
					DisableColors:   false,
				}
				logRuntimeFormatter.ChildFormatter = logTextFormatter
				slog.SetFormatter(logRuntimeFormatter)
			} else { //disable output if is not terminal
				// fmt.Println("Not is Terminal")
				slog.SetOutput(io.Discard)
			}
		}
	} else {
		slog.SetFormatter(logRuntimeFormatter)
	}

	if len(logpath) != 0 { //hook rotation
		if false { //for test only
			pathMap := PathMap{}
			for _, level := range logrus.AllLevels {
				if level < (log_level + 1) {
					pathMap[level] = logpath
				}
			}
			localFileHook := NewLocalFileHook(
				pathMap,
				logJsonFormatter,
			)
			logrus.AddHook(localFileHook)
		}
		maxSizeKb := 1024
		if m := os.Getenv("SLOGRUS_MAXSIZE"); m != "" {
			if maxSizeKbTmp, e := strconv.Atoi(m); e == nil {
				maxSizeKb = maxSizeKbTmp
			}
		}
		maxAge := 31

		if m := os.Getenv("SLOGRUS_MAXAGEDAYS"); m != "" {
			if maxAgeTmp, e := strconv.Atoi(m); e == nil {
				maxAge = maxAgeTmp
			}
		}

		maxBackups := 32
		if m := os.Getenv("SLOGRUS_MAXBACKUPS"); m != "" {
			if maxBackupsTmp, e := strconv.Atoi(m); e == nil {
				maxBackups = maxBackupsTmp
			}
		}

		enableCompress := true
		if m := os.Getenv("SLOGRUS_ENABLECOMPRESS"); m != "" {
			if enableCompressTmp, e := strconv.ParseBool(m); e == nil {
				enableCompress = enableCompressTmp
			}
		}

		rotateFileHook := NewRotateFileHook(RotateFileConfig{
			Filename:   logpath,
			MaxSize:    maxSizeKb, // kbytes
			MaxBackups: maxBackups,
			MaxAgeDays: maxAge,    //days
			Level:      log_level, //for file
			Formatter:  logRuntimeFormatter,
			Compress:   enableCompress,
			BuffSize:   1024 * 10,
		})
		// slog.rh = rotateFileHook.(*RotateFileHook)
		slog.AddHook(rotateFileHook)
	}
}

func GetStandardLogger() *Slog {
	return stdSlog
}

// log for stdout and logfile,
// Logpaths are logpath and disable parser json msg
func InitStandardLogger(log_level Level, pretty bool, diableStdout bool, logpaths ...any) *Slog {
	// stdSlog = &Slog{
	// 	Logger: logrus.StandardLogger(),
	// }
	initDefaultLog(stdSlog, log_level, pretty, diableStdout, logpaths...)
	return stdSlog
}

// log for stdout without logfile, prety json
func InitStandardLoggerWithDefault(log_level Level) *Slog {
	return InitStandardLogger(log_level, true, false)
}

// log for stdout and logfile,
func GetOldLogFiles() (filesPath []string) {
	return stdSlog.GetOldLogFiles()
}

func RotateSetHookCompress(h func(zipPath string) error) {
	zipPostHook = h
}

func WithFields(fields map[string]any) *Entry {
	return &Entry{
		stdSlog.Logger.WithFields(logrus.Fields(fields)),
	}
}

func (slog *Slog) RemoveFields(fields ...string) {
	// if !slog.initted || slog.rh == nil {
	if !slog.initted {
		return
	}
	if slog.Formatter != nil {
		if frt, ok := slog.Formatter.(*FormatterRuntime); ok {
			for _, k := range fields {
				if k == FieldKeyFunc || k == FieldKeyPakage || k == FieldKeyLine || k == FieldKeyFile {
					continue
				}
				delete(frt.globalFields, k)
			}
		}
	}
}

func (slog *Slog) UpdateFields(fields map[string]any) {
	if slog.Formatter != nil {
		if frt, ok := slog.Formatter.(*FormatterRuntime); ok {
			for k, v := range fields {
				if k == FieldKeyFunc || k == FieldKeyPakage || k == FieldKeyLine || k == FieldKeyFile {
					continue
				}
				if frt.globalFields == nil {
					frt.globalFields = map[string]any{}
				}
				frt.globalFields[k] = v
			}
		}
	}
}

func UpdateFields(fields map[string]any) {
	stdSlog.UpdateFields(fields)
}

func RemoveFields(fields ...string) {
	stdSlog.RemoveFields(fields...)
}

func (slog *Slog) ResetFields() {
	if slog.Formatter != nil {
		if frt, ok := slog.Formatter.(*FormatterRuntime); ok {
			frt.globalFields = map[string]any{}
		}
	}
}

func ResetFields() {
	stdSlog.ResetFields()
}
