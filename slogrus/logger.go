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
	Trace(v ...interface{})
	Debug(v ...interface{})
	Print(v ...interface{})
	Info(v ...interface{})
	Warn(v ...interface{})
	Error(v ...interface{})
	Fatal(v ...interface{})

	Tracef(format string, v ...interface{})
	Debugf(format string, v ...interface{})
	Printf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})

	// CtxTracef(ctx context.Context, format string, v ...interface{})
	// CtxDebugf(ctx context.Context, format string, v ...interface{})
	// CtxInfof(ctx context.Context, format string, v ...interface{})
	// CtxWarnf(ctx context.Context, format string, v ...interface{})
	// CtxErrorf(ctx context.Context, format string, v ...interface{})
	// CtxFatalf(ctx context.Context, format string, v ...interface{})
}

// Level defines the priority of a log message.
// When a logger is configured with a level, any log message with a lower
// log level (smaller by integer comparison) will not be output.
type Level struct {
	logrus.Level
}

// The levels of logs.
var (
	LevelPanic = Level{logrus.PanicLevel}
	LevelTrace = Level{logrus.TraceLevel}
	LevelDebug = Level{logrus.DebugLevel}
	LevelInfo  = Level{logrus.InfoLevel}
	LevelWarn  = Level{logrus.WarnLevel}
	LevelError = Level{logrus.ErrorLevel}
	LevelFatal = Level{logrus.FatalLevel}
)

var levelOfDefaultLogger Level //level for default logger

// SetLevel sets the level of logs below which logs will not be output.
// The default log level is LevelTrace.

// ParseLevel takes a string level and returns the Logrus log level constant.
func ParseLevel(lvl string) (Level, error) {
	if level, err := logrus.ParseLevel(lvl); err == nil {
		return Level{level}, nil
	} else {
		return LevelPanic, err
	}
}

func SetLevel(lv Level) {
	_defaultLogger.(*Slog).SetLevel(lv.Level)
}

// Fatal calls the default logger's Fatal method and then os.Exit(1).
func Fatal(v ...interface{}) {
	_defaultLogger.Fatal(v...)
}

// Error calls the default logger's Error method.
func Error(v ...interface{}) {
	_defaultLogger.Error(v...)
}

// Warn calls the default logger's Warn method.
func Warn(v ...interface{}) {
	_defaultLogger.Warn(v...)
}

func Print(v ...interface{}) {
	_defaultLogger.Print(v...)
}

// Info calls the default logger's Info method.
func Info(v ...interface{}) {
	_defaultLogger.Info(v...)
}

// Debug calls the default logger's Debug method.
func Debug(v ...interface{}) {
	_defaultLogger.Debug(v...)
}

// Trace calls the default logger's Trace method.
func Trace(v ...interface{}) {
	_defaultLogger.Trace(v...)
}

// Fatalf calls the default logger's Fatalf method and then os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	_defaultLogger.Fatalf(format, v...)
}

// Errorf calls the default logger's Errorf method.
func Errorf(format string, v ...interface{}) {
	_defaultLogger.Errorf(format, v...)
}

// Warnf calls the default logger's Warnf method.
func Warnf(format string, v ...interface{}) {
	_defaultLogger.Warnf(format, v...)
}

// Infof calls the default logger's Infof method.
func Printf(format string, v ...interface{}) {
	_defaultLogger.Printf(format, v...)
}

// Infof calls the default logger's Infof method.
func Infof(format string, v ...interface{}) {
	_defaultLogger.Infof(format, v...)
}

// Debugf calls the default logger's Debugf method.
func Debugf(format string, v ...interface{}) {
	_defaultLogger.Debugf(format, v...)
}

// Tracef calls the default logger's Tracef method.
func Tracef(format string, v ...interface{}) {
	_defaultLogger.Tracef(format, v...)
}
