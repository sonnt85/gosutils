package gosed

import (
	"bytes"
	//	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func Sed(evalProg, fileedit string) (edited bool, err error) {
	var expressions io.Reader
	expressions = strings.NewReader(evalProg)

	//	engine, err := NewQuiet(expressions)
	engine, err := New(expressions)

	if err != nil {
		return
	}

	orgcontents, err := ioutil.ReadFile(fileedit)
	if err != nil {
		return
	}

	fl, err := os.Open(fileedit)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, engine.Wrap(fl))
	fl.Close()
	if err != nil {
		return
	}

	if bytes.Compare(buf.Bytes(), orgcontents) != 0 {
		info, err := os.Stat(fileedit)
		if err != nil {
			return false, err
		}

		err = ioutil.WriteFile(fileedit, buf.Bytes(), info.Mode().Perm())
		return true, nil
	} else {
		return false, nil
	}
}
