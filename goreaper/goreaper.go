package goreaper

type Config struct {
	Pid              int
	Options          int
	DisablePid1Check bool
}

var debug bool = false
var calledReap bool = false

func EnableDebug() {
	debug = true
}

func DisableDebug() {
	debug = false
}

func Reap(disableCheckPid1s ...bool) {
	if !calledReap {
		calledReap = true
		reap(disableCheckPid1s...)
	}
}

func Start(config Config) {
	start(config)
}
