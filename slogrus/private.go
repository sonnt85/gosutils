package slogrus

var (
	DEBUGENVNAME = "DEBUGENVNAME"
	ON_PREMISE   = "ON_PREMISE"
	FORCELOG     = "FORCELOG"
)

var slog func(args ...interface{}) // function pointer as requested

func EnableSLog(enable bool) {
	// Implementation to enable or disable slog
}
