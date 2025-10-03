// Copyright 2021 ByteDance Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package slogrus

import (
	"github.com/sirupsen/logrus"
)

// Logger is a logger interface that provides logging function with levels.
type Logger interface {
	Trace(v ...any)
	Debug(v ...any)
	Print(v ...any)
	// WriteStd(format string, v ...any)
	// WritefStd(v ...any)
	Info(v ...any)
	Warn(v ...any)
	Error(v ...any)
	Fatal(v ...any)

	Tracef(format string, v ...any)
	Debugf(format string, v ...any)
	Printf(format string, v ...any)
	Infof(format string, v ...any)
	Warnf(format string, v ...any)
	Errorf(format string, v ...any)
	Fatalf(format string, v ...any)

	// CtxTracef(ctx context.Context, format string, v ...any)
	// CtxDebugf(ctx context.Context, format string, v ...any)
	// CtxInfof(ctx context.Context, format string, v ...any)
	// CtxWarnf(ctx context.Context, format string, v ...any)
	// CtxErrorf(ctx context.Context, format string, v ...any)
	// CtxFatalf(ctx context.Context, format string, v ...any)
}

// Level defines the priority of a log message.
// When a logger is configured with a level, any log message with a lower
// log level (smaller by integer comparison) will not be output.
type Level = logrus.Level

// The levels of logs.
const (
	LevelPanic = logrus.PanicLevel
	LevelTrace
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanicOnly = logrus.PanicLevel + 100
	LevelTraceOnly
	LevelDebugOnly
	LevelInfoOnly
	LevelWarnOnly
	LevelErrorOnly
	LevelFatalOnly
)

const levelOfDefaultLogger = LevelPanic // level for default logger

// SetLevel sets the level of logs below which logs will not be output.
// The default log level is LevelTrace.

var __parselevel func(string) (Level, error)

// ParseLevel takes a string level and returns the Logrus log level constant.
func ParseLevel(lvl string) (Level, error) {
	if level, err := logrus.ParseLevel(lvl); err == nil {
		return level, nil
	} else if __parselevel != nil {
		return __parselevel(lvl)
	} else {
		return levelOfDefaultLogger, err
	}
}

func SetLevel(lv Level) {
	_defaultLogger.(*Slog).SetLevel(lv)
}

// Fatal calls the default logger's Fatal method and then os.Exit(1).
func Fatal(v ...any) {
	_defaultLogger.Fatal(v...)
}

var FatalS = Fatal

// Error calls the default logger's Error method.
func Error(v ...any) {
	_defaultLogger.Error(v...)
}

var ErrorS = Error

// Warn calls the default logger's Warn method.
func Warn(v ...any) {
	_defaultLogger.Warn(v...)
}

var WarnS = Warn

func Print(v ...any) {
	_defaultLogger.Print(v...)
}

func WriteStd(v ...any) {
	if slog, ok := _defaultLogger.(*Slog); ok {
		slog.WriteStd(v...)
	} else {
		_defaultLogger.Print(v...)
	}
}

var WriteStdS = WriteStd

func WritefStd(format string, v ...any) {
	if slog, ok := _defaultLogger.(*Slog); ok {
		slog.WritefStd(format, v...)
	} else {
		_defaultLogger.Printf(format, v...)
	}
}

var WritefStdS = WritefStd
var PrintS = Print

// Info calls the default logger's Info method.
func Info(v ...any) {
	_defaultLogger.Info(v...)
}

var InfoS = Info

// Debug calls the default logger's Debug method.
func Debug(v ...any) {
	_defaultLogger.Debug(v...)
}

var DebugS = Debug

// Trace calls the default logger's Trace method.
func Trace(v ...any) {
	_defaultLogger.Trace(v...)
}

var TraceS = Trace

// Fatalf calls the default logger's Fatalf method and then os.Exit(1).
func Fatalf(format string, v ...any) {
	_defaultLogger.Fatalf(format, v...)
}

var FatalfS = Fatalf

// Errorf calls the default logger's Errorf method.
func Errorf(format string, v ...any) {
	_defaultLogger.Errorf(format, v...)
}

var ErrorfS = Errorf

// Warnf calls the default logger's Warnf method.
func Warnf(format string, v ...any) {
	_defaultLogger.Warnf(format, v...)
}

var WarnfS = Warnf

var Warning = Warn
var WarningS = Warning

// Infof calls the default logger's Infof method.
func Printf(format string, v ...any) {
	_defaultLogger.Printf(format, v...)
}

var PrintfS = Printf

// Infof calls the default logger's Infof method.
func Infof(format string, v ...any) {
	_defaultLogger.Infof(format, v...)
}

var InfofS = Infof

// Debugf calls the default logger's Debugf method.
func Debugf(format string, v ...any) {
	_defaultLogger.Debugf(format, v...)
}

var DebugfS = Debugf

// Tracef calls the default logger's Tracef method.
func Tracef(format string, v ...any) {
	_defaultLogger.Tracef(format, v...)
}

var TracefS = Tracef
