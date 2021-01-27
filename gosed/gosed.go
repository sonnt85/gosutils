package gosed

import (
	//"fmt"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func Sed(sedscript, filepath string) (changed bool, retstring string, err error) {
	return SedFunc(sedscript, filepath, true)
}

func SedString(sedscript, data string) (changed bool, retstring string, err error) {
	return SedFunc(sedscript, data, false)
}

func SedFunc(sedscript, filepath_or_string string, isFile bool) (changed bool, retstring string, err error) {
	var expressions io.Reader
	expressions = strings.NewReader(sedscript)

	//	engine, err := NewQuiet(expressions)
	engine, err := New(expressions)

	if err != nil {
		return
	}

	orgcontents, err := ioutil.ReadFile(filepath_or_string)
	if err != nil {
		return
	}
	var buf bytes.Buffer

	if isFile {
		var fl *os.File
		fl, err = os.Open(filepath_or_string)
		if err != nil {
			return
		}

		_, err = io.Copy(&buf, engine.Wrap(fl))
		fl.Close()
		if err != nil {
			return
		}
		if bytes.Compare(buf.Bytes(), orgcontents) != 0 {
			info, err := os.Stat(filepath_or_string)
			if err != nil {
				return false, "", err
			}

			err = ioutil.WriteFile(filepath_or_string, buf.Bytes(), info.Mode().Perm())
			if err != nil {
				return false, "", err
			}

			return true, "", nil
		} else {
			return false, "", nil
		}
	} else {
		if retstring, err = engine.RunString(filepath_or_string); err != nil {
			if bytes.Compare([]byte(retstring), orgcontents) != 0 {
				return true, retstring, nil
			} else {
				return false, retstring, nil
			}
		} else {
			return
		}

	}

}
