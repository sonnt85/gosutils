package runonce

func RebornNewProgram(port int, newname string, cloneflags ...bool) bool {
	return false
}

/*
  Set close-on-exec state for all fds >= 3
  The idea comes from
    https://github.com/golang/gofrontend/commit/651e71a729e5dcbd9dc14c1b59b6eff05bfe3d26
*/
func closeOnExec(state bool) {
	return
}
