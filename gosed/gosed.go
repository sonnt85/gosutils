package gosed

import (
	//"fmt"
	"bytes"
	"io"
	"io/fs"
	"io/ioutil"
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

	allbytes, err := ioutil.ReadFile(filepath)
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

	if bytes.Compare([]byte(newstring), allbytes) == 0 {
		return nil
	} else {
		return ioutil.WriteFile(filepath, []byte(newstring), fs.FileMode(0644))
	}
}

func StringToPattern(str string) (retstring string) {
	return regexp.QuoteMeta(str)

	for _, vrune := range string(`\.+*?()|[]{}^$`) {
		v := string(vrune)
		str = strings.ReplaceAll(str, v, `\`+v)
	}
	return str
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

		orgcontents, err = ioutil.ReadFile(filepath_or_string)
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
		orgcontents := filepath_or_string
		if retstring, err = engine.RunString(filepath_or_string); err != nil {
			if bytes.Compare([]byte(retstring), []byte(orgcontents)) != 0 {
				return true, retstring, nil
			} else {
				return false, retstring, nil
			}
		} else {
			return
		}

	}

}
