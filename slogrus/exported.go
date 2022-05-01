package slogrus

import (
	"io"
	"io/ioutil"
	"os"
	"runtime/debug"

	"github.com/mattn/go-colorable"
	"github.com/sonnt85/gosystem"

	// "github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
)

var defaultTimestampFormat = "2006-01-02T15:04:05.000Z07:00"

//Log Level 0->panic; 1->fatal, 2->error, 3->warn, 4->info, 5->debug, 6->trace
// type Level uint32

type Slog struct {
	*logrus.Logger
	rh      *RotateFileHook
	initted bool
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

//new slog file with default stdout is os.Stderr
func NewLogFile(logPath string, log_level Level, pretty bool, diableStdout bool, logpath ...interface{}) *Slog {
	slog := &Slog{
		Logger: logrus.New(),
	}
	initDefaultLog(slog, log_level, pretty, diableStdout, logpath...)
	return slog
}

func (slog *Slog) TraceStack(msg ...any) {
	msg = append(msg, string(debug.Stack()))
	slog.Trace(msg...)
}

func (slog *Slog) TracefStack(format string, args ...interface{}) {
	args = append(args, string(debug.Stack()))
	format = format + "[%s]"
	slog.Tracef(format, args...)
}

func TracefStack(format string, args ...interface{}) {
	stdSlog.TracefStack(format, args...)
}

func TraceStack(msg ...any) {
	stdSlog.TraceStack(msg...)
}

func (slog *Slog) GetOldLogFiles() (retpaths []string) {
	if !slog.initted {
		return
	}
	if lgr, ok := slog.rh.logWriter.(*LoggerRotate); ok {
		return lgr.GetOldLogFiles()
	}
	return
}

//logPath string, log_level logrus.Level, pretty bool)
func initDefaultLog(slog *Slog, log_level Level, pretty bool, diableStdout bool, logpaths ...interface{}) {
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
	slog.initted = true
	slogOutFile, outputIsOsFile := slog.Out.(*os.File)
	if diableStdout {
		slog.SetOutput(ioutil.Discard)
		if stdSlog == slog { //auto disable os.Stdout if is standard log
			os.Stdout, _ = os.Open(os.DevNull)
			os.Stderr, _ = os.Open(os.DevNull)
		}
	} else {
		if outputIsOsFile {
			slog.SetOutput(colorable.NewColorable(slogOutFile))
			// colorable.EnableColorsStdout(nil)()
		}
	}

	// timeFormat := time.RFC3339 //"2006-01-02T15:04:05Z07:00"
	timeFormat := defaultTimestampFormat //milisecond
	logJsonFormatter := &JSONFormatter{
		// logJsonFormatter := &logrus.JSONFormatter{
		TimestampFormat:      timeFormat,
		PrettyPrint:          pretty,
		DisableHTMLEscape:    true,
		DisableMsgJsonOpject: disableMsgJsonOpject,
	}

	logRuntimeFormatter := &FormatterRuntime{
		ChildFormatter: logJsonFormatter,
		File:           true,
		Line:           true,
		Package:        false,
		// TextToSearchFun: "gosutils.slogrus.",
	}
	slog.SetLevel(log_level.Level)
	if stdSlog == slog { //print to stdout standard, auto disable output if not is terminal
		if !diableStdout {
			if outputIsOsFile && gosystem.IsTerminal(slogOutFile.Fd()) {
				// fmt.Println("Is Terminal")

				// if gosystem.IsTerminalWriter(stdSlog.Out) || (os.Getenv(DEBUGENVNAME) == "yes" && runtime.GOOS == "windows" || gosystem.IsTerminal(os.Stderr.Fd())) {
				// if gosystem.IsTerminalWriter(stdSlog.Out) {
				// if (ok && (isatty.IsTerminal(fileprr.Fd()) || isatty.IsCygwinTerminal(fileprr.Fd()))) || (isatty.IsTerminal(stdoutFD) || isatty.IsCygwinTerminal(stdoutFD)) {
				// fmt.Println("Is Terminal")
				logStdStandardRuntimeFormatter := *logRuntimeFormatter
				logTextFormatter := &logrus.TextFormatter{
					TimestampFormat: timeFormat,
					FullTimestamp:   true,
					ForceColors:     true,
					DisableColors:   false}
				logStdStandardRuntimeFormatter.ChildFormatter = logTextFormatter
				slog.SetFormatter(&logStdStandardRuntimeFormatter)
			} else { //disable output if is not terminal
				// fmt.Println("Not is Terminal")
				slog.SetOutput(ioutil.Discard)
			}
		}
	} else {
		slog.SetFormatter(logRuntimeFormatter)
	}

	if len(logpath) != 0 { //hook rotation
		if false { //for test only
			pathMap := PathMap{}
			for _, level := range logrus.AllLevels {
				if level < (log_level.Level + 1) {
					pathMap[level] = logpath
				}
			}
			localFileHook := NewLocalFileHook(
				pathMap,
				logJsonFormatter,
			)
			logrus.AddHook(localFileHook)
		}

		rotateFileHook := NewRotateFileHook(RotateFileConfig{
			Filename:   logpath,
			MaxSize:    1024, // kbytes
			MaxBackups: 32,
			MaxAge:     31,              //days
			Level:      log_level.Level, //for file
			Formatter:  logRuntimeFormatter,
			Compress:   true,
			BuffSize:   1024 * 10,
		})
		slog.rh = rotateFileHook.(*RotateFileHook)
		slog.AddHook(rotateFileHook)
	}
}

func GetStandardLogger() *Slog {
	return stdSlog
}

//log for stdout and logfile,
func InitStandardLogger(log_level Level, pretty bool, diableStdout bool, logpaths ...interface{}) *Slog {
	if stdSlog.initted {
		return stdSlog
	}
	// stdSlog = &Slog{
	// 	Logger: logrus.StandardLogger(),
	// }
	initDefaultLog(stdSlog, log_level, pretty, diableStdout, logpaths...)
	return stdSlog
}

//log for stdout and logfile,
func GetOldLogFiles() (filesPath []string) {
	return stdSlog.GetOldLogFiles()
}
