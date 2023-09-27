package gogrep

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func GrepFileLine(file, pat string, numberlineMatch int, literalFlags ...bool) (retarray []string, err error) {
	//	var retarray []string
	if len(literalFlags) != 0 && literalFlags[0] {
		pat = regexp.QuoteMeta(pat)
	}

	f, err := os.Open(file)
	if err != nil {
		return retarray, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	nmatch := 0
	r, err := regexp.Compile(pat)
	if err != nil {
		return retarray, err
	}

	for scanner.Scan() {
		datas := r.FindAllString(string(scanner.Bytes()), -1)
		if len(datas) != 0 {
			nmatch++
			retarray = append(retarray, datas...)
			if numberlineMatch != -1 {
				if nmatch == numberlineMatch {
					break
				}
			}
			//		matched, _ := regexp.Match(, scanner.Bytes())
		}
	}
	return retarray, err
}

func GrepFileLines(file, pat string, numberlineMatch int, literalFlags ...bool) (retarray []string, err error) {

	if len(literalFlags) != 0 && literalFlags[0] {
		pat = regexp.QuoteMeta(pat)
	}

	allbytes, err := os.ReadFile(file)
	if err != nil {
		return retarray, err
	}

	r, err := regexp.Compile(pat)
	if err != nil {
		return retarray, err
	}

	retarray = r.FindAllString(string(allbytes), numberlineMatch)

	return retarray, err
}

func FileIsMatchLine(filepath, pattern string, literalFlags ...bool) bool {
	ret, err := GrepFileLine(filepath, pattern, 1, literalFlags...)
	if len(ret) != 0 && err == nil {
		return true
	} else {
		return false
	}
}

func FileIsMatchLines(filepath, pattern string, literalFlags ...bool) bool {
	ret, err := GrepFileLines(filepath, pattern, 1, literalFlags...)
	if len(ret) != 0 && err == nil {
		return true
	} else {
		return false
	}
}

func FileIsMatchLiteralLine(filepath, pattern string) bool {
	ret, err := GrepFileLine(filepath, pattern, 1, true)
	if len(ret) != 0 && err == nil {
		return true
	} else {
		return false
	}
}

func FileIsMatchLiteralLines(filepath, pattern string) bool {
	ret, err := GrepFileLines(filepath, pattern, 1, true)
	if len(ret) != 0 && err == nil {
		return true
	} else {
		return false
	}
}

func GrepStringLine(str, pat string, numberlineMatch int, literalFlags ...bool) (retarray []string, err error) {
	//func GrepStringLine(data string, pat []byte, nlines ...int) (retarray []string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(str))

	if len(literalFlags) != 0 && literalFlags[0] {
		pat = regexp.QuoteMeta(pat)
	}
	r, err := regexp.Compile(string(pat))
	if err != nil {
		return retarray, err
	}

	nmatch := 0
	for scanner.Scan() {
		if datas := r.FindAllString(string(scanner.Bytes()), -1); datas != nil {
			nmatch++
			retarray = append(retarray, datas...)
			if numberlineMatch != -1 {
				if nmatch == numberlineMatch {
					break
				}
			}
			//		matched, _ := regexp.Match(, scanner.Bytes())
		}
	}
	return retarray, err
}

func GrepStringLines(str, pat string, numberlineMatch int, literalFlags ...bool) (retarray []string, err error) {
	if len(literalFlags) != 0 && literalFlags[0] {
		pat = regexp.QuoteMeta(pat)
	}
	r, err := regexp.Compile(string(pat))
	if err != nil {
		return retarray, err
	}

	retarray = r.FindAllString(str, numberlineMatch)
	if len(retarray) != 0 {
		return retarray, nil
	} else {
		return retarray, fmt.Errorf("Can not grep pattern")
	}
}

func StringIsMatchLine(str, pat string, literalFlags ...bool) bool {
	if ret, err := GrepStringLine(str, pat, 1, literalFlags...); err == nil && len(ret) != 0 {
		return true
	}
	return false
}

func StringIsMatchLines(str, pat string, literalFlags ...bool) bool {
	if ret, err := GrepStringLines(str, pat, 1, literalFlags...); err == nil && len(ret) != 0 {
		return true
	}
	return false
}
