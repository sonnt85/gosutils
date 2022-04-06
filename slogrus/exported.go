package slogrus

import (
	"io"
	"io/ioutil"
	"os"
	"runtime/debug"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
)

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

func NewLogFile(logPath string, log_level logrus.Level, pretty bool, diableStdout bool, logpath ...string) *Slog {
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

func TraceStack(msg ...any) {
	stdSlog.TraceStack(msg...)
}

func (slog *Slog) GetOldLogFiles() (retpaths []string) {
	if !slog.initted {
		return
	}
	if lgr, ok := slog.rh.logWriter.(*LoggerRotate); ok {
		if filesinfo, err := lgr.GetOldLogFiles(); err == nil {
			for _, fp := range filesinfo {
				retpaths = append(retpaths, fp.Name())
			}
		}
	}
	return
}

//logPath string, log_level logrus.Level, pretty bool)
func initDefaultLog(slog *Slog, log_level logrus.Level, pretty bool, diableStdout bool, logpaths ...string) {

	slog.initted = true
	if diableStdout {
		slog.SetOutput(ioutil.Discard)
		if stdSlog == slog {
			os.Stdout, _ = os.Open(os.DevNull)
			os.Stderr, _ = os.Open(os.DevNull)
		}
	} else {
		slog.SetOutput(colorable.NewColorableStdout())
		colorable.EnableColorsStdout(nil)
	}

	timeFormat := time.RFC3339

	logJsonFormatter := &logrus.JSONFormatter{
		TimestampFormat: timeFormat,
		PrettyPrint:     pretty,
	}

	logRuntimeFormatter := &FormatterRuntime{
		ChildFormatter: logJsonFormatter,
		File:           true,
		Line:           true,
		Package:        false,
	}
	slog.SetLevel(log_level)
	if stdSlog == slog { //print to stdout standard
		if !diableStdout {
			// fmt.Println("Config standard Log Level: ", log_level)
			fileprr, ok := stdSlog.Out.(*os.File)
			if ok && (isatty.IsTerminal(fileprr.Fd()) || isatty.IsCygwinTerminal(fileprr.Fd())) {
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
				slog.SetOutput(ioutil.Discard)
			}
		}
	} else {
		slog.SetFormatter(logRuntimeFormatter)
	}

	if len(logpaths) != 0 && len(logpaths[0]) != 0 { //hook rotation
		if false { //for test only
			pathMap := PathMap{}
			for _, level := range logrus.AllLevels {
				if level < (log_level + 1) {
					pathMap[level] = logpaths[0]
				}
			}
			localFileHook := NewLocalFileHook(
				pathMap,
				logJsonFormatter,
			)
			logrus.AddHook(localFileHook)
		}

		rotateFileHook := NewRotateFileHook(RotateFileConfig{
			Filename:   logpaths[0],
			MaxSize:    1024, // kbytes
			MaxBackups: 32,
			MaxAge:     31,        //days
			Level:      log_level, //for file
			Formatter:  logRuntimeFormatter,
			Compress:   true,
			BuffSize:   1024 * 10,
		})
		slog.rh = rotateFileHook.(*RotateFileHook)
		slog.AddHook(rotateFileHook)
	}
}

//log for stdout and logfile,
func InitDefaultLog(log_level logrus.Level, pretty bool, diableStdout bool, logpaths ...string) *Slog {
	if stdSlog.initted {
		return stdSlog
	}
	stdSlog = &Slog{
		Logger: logrus.StandardLogger(),
	}
	initDefaultLog(stdSlog, log_level, pretty, diableStdout, logpaths...)
	return stdSlog
}

//log for stdout and logfile,
func GetOldLogFiles() (filesPath []string) {
	return stdSlog.GetOldLogFiles()
}

func StandardLogger() *logrus.Logger {
	return logrus.StandardLogger()
}
