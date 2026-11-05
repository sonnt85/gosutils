package sexec

import "os/exec"

// PreStartHook is called with the prepared *exec.Cmd right before cmd.Start().
// Use it to adjust the command, for example to set SysProcAttr.Setpgid or Dir.
// It is a distinct type so the variadic options can tell it apart from the
// end hook (plain func(*exec.Cmd)), which runs after cmd.Wait().
type PreStartHook func(*exec.Cmd)

// StartedHook is called synchronously, in the calling goroutine, right after
// cmd.Start() succeeded and before cmd.Wait(). cmd.Process is set, so the hook
// can read the pid without racing with another goroutine.
type StartedHook func(*exec.Cmd)
