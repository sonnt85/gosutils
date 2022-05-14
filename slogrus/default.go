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

import "io"

var (
	_ Logger = (*Slog)(nil)
)

var _defaultLogger Logger

// SetDefaultLogger sets the default logger.
// This is not concurrency safe, which means it should only be called during init.
func SetDefaultLogger(l Logger) {
	if l == nil {
		panic("logger must not be nil")
	}
	_defaultLogger = l
}

func SetDefaultLoggerIsDiscard() {
	SetDefaultLogger(New(io.Discard))
}

func GetDefaultLogger() Logger {
	return _defaultLogger
}

func init() {
	_defaultLogger = GetStandardLogger()
	// _defaultLogger = New(os.Stderr)
}
