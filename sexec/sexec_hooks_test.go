//go:build linux

package sexec

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreStartHookRunsBeforeStartAndSetpgidApplies(t *testing.T) {
	var out bytes.Buffer
	var called atomic.Bool
	pre := PreStartHook(func(c *exec.Cmd) {
		require.Nil(t, c.Process, "pre-start hook must run before the process exists")
		called.Store(true)
		c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	})
	var pid int
	started := StartedHook(func(c *exec.Cmd) { pid = c.Process.Pid })

	// The shell prints its own process group id; with Setpgid it equals its pid.
	_, _, err := ExecCommandCtxShellEnvTimeout(context.Background(), `ps -o pgid= -p $$`, nil, 0, pre, started, &out, &out)
	require.NoError(t, err)
	require.True(t, called.Load())
	pgid, convErr := strconv.Atoi(strings.TrimSpace(out.String()))
	require.NoError(t, convErr, "output: %q", out.String())
	require.Equal(t, pid, pgid, "the shell must lead its own process group")
}

func TestStartedHookReceivesThePid(t *testing.T) {
	var out bytes.Buffer
	var hookPid int
	var cmdPtr *exec.Cmd
	started := StartedHook(func(c *exec.Cmd) {
		require.NotNil(t, c.Process)
		hookPid = c.Process.Pid
	})
	_, _, err := ExecCommandCtxShellEnvTimeout(context.Background(), `echo $$`, nil, 0, started, &cmdPtr, &out, &out)
	require.NoError(t, err)
	require.NotZero(t, hookPid)
	require.Equal(t, strconv.Itoa(hookPid), strings.TrimSpace(out.String()))
	require.Equal(t, hookPid, cmdPtr.Process.Pid)
}

func TestEndHookStillRunsAfterWait(t *testing.T) {
	var order []string
	pre := PreStartHook(func(*exec.Cmd) { order = append(order, "pre") })
	started := StartedHook(func(*exec.Cmd) { order = append(order, "started") })
	end := func(c *exec.Cmd) {
		require.NotNil(t, c.ProcessState, "end hook runs after Wait")
		order = append(order, "end")
	}
	_, _, err := ExecCommandCtxEnvTimeout(context.Background(), "true", nil, 0, end, pre, started)
	require.NoError(t, err)
	require.Equal(t, []string{"pre", "started", "end"}, order)
}

func TestStartedHookNotCalledWhenStartFails(t *testing.T) {
	var startedCalled, preCalled bool
	pre := PreStartHook(func(*exec.Cmd) { preCalled = true })
	started := StartedHook(func(*exec.Cmd) { startedCalled = true })
	_, _, err := ExecCommandCtxEnvTimeout(context.Background(), "/nonexistent/binary-for-sexec-test", nil, 0, pre, started)
	require.Error(t, err)
	require.True(t, preCalled)
	require.False(t, startedCalled)
}
