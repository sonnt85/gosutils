package slogrus

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// FieldKeyPakage holds the package field
const FieldKeyPakage = "package"
const FieldKeyLine = "line"
const FieldKeyFile = logrus.FieldKeyFile               // "file"
const FieldKeyMsg = logrus.FieldKeyMsg                 // "msg"
const FieldKeyLevel = logrus.FieldKeyLevel             // "level"
const FieldKeyTime = logrus.FieldKeyTime               // "time"
const FieldKeyLogrusError = logrus.FieldKeyLogrusError // "logrus_error"
const FieldKeyFunc = logrus.FieldKeyFunc               // "func"

const (
	logrusStackJump          = 4
	logrusFieldlessStackJump = 6
)

// Formatter decorates log entries with function name and package name (optional) and line number (optional)
type FormatterRuntime struct {
	// TextToSearchFun string
	ChildFormatter logrus.Formatter
	// When true, line number will be tagged to fields as well
	Line bool
	// When true, package name will be tagged to fields as well
	Package bool
	// When true, file name will be tagged to fields as well
	File bool
	// When true, only base name of the file will be tagged to fields
	BaseNameOnly bool

	RootDir string

	globalFields map[string]any
	// TraceFlag    bool
}

// Format the current log entry by adding the function name and line number of the caller.
func (f *FormatterRuntime) Format(entry *logrus.Entry) ([]byte, error) {
	function, file, line := f.getCurrentPosition(entry)

	packageEnd := strings.LastIndex(function, ".")
	functionName := function[packageEnd+1:]

	data := logrus.Fields{FieldKeyFunc: functionName}
	if f.Line {
		data[FieldKeyLine] = line
	}
	if f.Package {
		packageName := function[:packageEnd]
		data[FieldKeyPakage] = packageName
	}
	if f.File {
		if f.BaseNameOnly {
			flagNoErr := false
			if f.RootDir != "" {
				relFile, err := filepath.Rel(f.RootDir, file)
				if err == nil {
					file = relFile
					flagNoErr = true
				}
			}
			if !flagNoErr {
				data[FieldKeyFile] = filepath.Base(file)
			}
			// data[FieldKeyFile] = filepath.Base(file)
		} else {
			data[FieldKeyFile] = file
		}
	}
	for k, v := range f.globalFields {
		data[k] = v
	}
	for k, v := range entry.Data {
		data[k] = v
	}
	entry.Data = data

	return f.ChildFormatter.Format(entry)
}

func (f *FormatterRuntime) getCurrentPosition(entry *logrus.Entry) (string, string, string) {
	var function, lineNumber, file string
	var pc uintptr
	var line int
	skip := logrusStackJump
	if len(entry.Data) == 0 {
		skip = logrusFieldlessStackJump
	}
start:
	pc, file, line, _ = runtime.Caller(skip)
	lineNumber = ""
	if f.Line {
		lineNumber = fmt.Sprintf("%d", line)
	}
	function = runtime.FuncForPC(pc).Name()
	// if strings.LastIndex(function, "sirupsen/logrus.") != -1 {
	if (strings.LastIndex(function, "sirupsen/logrus.") != -1) || (strings.LastIndex(function, "gosutils/slogrus.") != -1) {
		skip++
		goto start
	}
	return function, file, lineNumber
}
