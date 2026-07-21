# slogrus

[![Go Reference](https://pkg.go.dev/badge/github.com/sonnt85/gosutils/slogrus.svg)](https://pkg.go.dev/github.com/sonnt85/gosutils/slogrus)

`slogrus` is an enhanced Logrus wrapper for Go that provides automatic runtime stack decoration (caller function, line number, file, and package tags), log file rotation, custom formatting, and global logger management.

## Features

- **Runtime Stack Decoration**: Automatically tag log messages with `func`, `file`, `line`, and `package` caller information without modifying application code.
- **Log File Rotation**: Built-in support for log rotation with file size limits, max age, max backup counts, and compression.
- **Structured JSON & Text Formatting**: Customizable JSON and Text formatters supporting custom timestamp formats, field ordering, and colored console output.
- **Global & Instance Logger**: Simple package-level functions (`slogrus.Info`, `slogrus.Errorf`, etc.) and configurable `Slog` instances.

## Installation

```bash
go get github.com/sonnt85/gosutils/slogrus
```

## Quick Start

### Basic Global Logging

```go
package main

import (
	"github.com/sonnt85/gosutils/slogrus"
)

func main() {
	slogrus.SetLevel(slogrus.LevelDebug)

	slogrus.Info("Application started")
	slogrus.Infof("Listening on port %d", 8080)
	slogrus.Debug("Debug info message")
}
```

### Log File Creation with Rotation

```go
package main

import (
	"github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/slogrus"
)

func main() {
	// Create a logger that writes to a file with rotation
	log := slogrus.NewLogFile("logs/app.log", logrus.InfoLevel, true, false)

	log.Info("Log message written to file and stderr")
}
```

### Runtime Formatter Usage

Wrap any Logrus formatter with `FormatterRuntime` to automatically inject caller details:

```go
package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/slogrus"
)

func main() {
	logger := slogrus.New(os.Stdout)

	runtimeFormatter := &slogrus.FormatterRuntime{
		ChildFormatter: &logrus.TextFormatter{
			FullTimestamp: true,
		},
		Line:         true,
		Package:      true,
		File:         true,
		BaseNameOnly: true,
	}

	logger.Formatter = runtimeFormatter
	logger.Info("Message with caller info tagged")
}
```

## API Summary

- `New(writer io.Writer) *Slog` — Create a new logger writing to specified output.
- `NewLogFile(path, level, pretty, disableStdout)` — Create a rotating file logger.
- `SetDefaultLogger(logger)` / `GetDefaultLogger()` — Manage the global default logger instance.
- `SetLevel(level)` / `ParseLevel(str)` — Configure log verbosity level.
- `FormatterRuntime` — Decorate log entries with `func`, `file`, `line`, and `package`.
- Package-level logging methods: `Trace`, `Debug`, `Info`, `Warn`, `Error`, `Fatal`, `Printf` and their formatted variants (`Tracef`, `Debugf`, `Infof`, `Warnf`, `Errorf`, `Fatalf`).

## License

Apache License 2.0 / MIT
