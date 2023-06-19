package sexec

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExecBytes(t *testing.T) {
	path, err := exec.LookPath("env")
	if err != nil {
		t.Fatal(err)
	}
	//lint:ignore SA4006 ignore this!
	b, err := os.ReadFile(path)
	b = []byte(`#!/bin/bash
`)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	out, _, err = ExecBytesEnvTimeout(b, "name", map[string]string{"TESTENV": "testenv"}, time.Second)
	require.Nil(t, err)
	fmt.Println(string(out))

}

func TestExecAsScript(t *testing.T) {
	if o, _, err := ExecCommandShell(`#/bin/bash;
	 ls -lhas /root`, 0); err == nil {
		t.Logf("%s", string(o))
	} else {
		t.Error(err)
	}

	t.Log("done")
}
func TestExecAsRoot(t *testing.T) {
	ExecCommandShellElevatedEnvTimeout("ls", 0, nil, time.Second*5, "-lhas", "/root")
	t.Log("done")
}
