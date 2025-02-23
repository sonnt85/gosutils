package gosed

import (
	//"fmt"
	"bytes"
	"io"
	"io/fs"
	"os"
	"regexp"
	"strings"
)

func Sed(sedscript, filepath string) (changed bool, retstring string, err error) {
	return SedFunc(sedscript, filepath, true)
}

func FileReplaceRegex(pat, tostring, filepath string, literalFlags ...bool) (err error) {
	literalFlag := false
	if len(literalFlags) != 0 && literalFlags[0] {
		literalFlag = true
	}

	allbytes, err := os.ReadFile(filepath)
	contenStr := string(allbytes)
	if err != nil {
		return err
	}

	r, err := regexp.Compile(string(pat))
	if err != nil {
		return err
	}
	newstring := ""
	if literalFlag {
		newstring = r.ReplaceAllLiteralString(contenStr, tostring)
	} else {
		newstring = r.ReplaceAllString(contenStr, tostring)
	}

	if bytes.Equal([]byte(newstring), allbytes) {
		return nil
	} else {
		return os.WriteFile(filepath, []byte(newstring), fs.FileMode(0644))
	}
}

func StringToPattern(str string) (retstring string) {
	return regexp.QuoteMeta(str)
}

func SedString(sedscript, data string) (changed bool, retstring string, err error) {
	return SedFunc(sedscript, data, false)
}

func SedFunc(sedscript, filepath_or_string string, isFile bool) (changed bool, retstring string, err error) {
	var expressions io.Reader
	changed = false
	expressions = strings.NewReader(sedscript)

	//	engine, err := NewQuiet(expressions)
	engine, err := New(expressions)

	if err != nil {
		return
	}

	var buf bytes.Buffer
	if isFile {
		var fl *os.File
		var orgcontents []byte

		orgcontents, err = os.ReadFile(filepath_or_string)
		if err != nil {
			return
		}

		fl, err = os.Open(filepath_or_string)
		if err != nil {
			return
		}

		_, err = io.Copy(&buf, engine.Wrap(fl))
		fl.Close()
		if err != nil {
			return
		}
		if !bytes.Equal(buf.Bytes(), orgcontents) {
			info, err := os.Stat(filepath_or_string)
			if err != nil {
				return false, "", err
			}

			err = os.WriteFile(filepath_or_string, buf.Bytes(), info.Mode().Perm())
			if err != nil {
				return false, "", err
			}

			return true, "", nil
		} else {
			return false, "", nil
		}
	} else {
		orgcontents := filepath_or_string
		if retstring, err = engine.RunString(filepath_or_string); err != nil {
			if !bytes.Equal([]byte(retstring), []byte(orgcontents)) {
				return true, retstring, nil
			} else {
				return false, retstring, nil
			}
		} else {
			return
		}

	}

}
