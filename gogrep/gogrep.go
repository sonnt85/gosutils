package gogrep

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func grepFile(file string, pat []byte) (retarray []string, err error) {
	//	var retarray []string
	f, err := os.Open(file)
	if err != nil {
		return retarray, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		r, err := regexp.Compile(string(pat))
		if err != nil {
			return retarray, err
		}
		if datas := r.FindAllString(string(scanner.Bytes()), -1); datas != nil {
			retarray = append(retarray, datas...)
			//		matched, _ := regexp.Match(, scanner.Bytes())
		}
	}
	return retarray, err
}

func GrepString(data string, pat []byte) (retarray []string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		r, err := regexp.Compile(string(pat))
		if err != nil {
			return retarray, err
		}
		if datas := r.FindAllString(string(scanner.Bytes()), -1); datas != nil {
			retarray = append(retarray, datas...)
			//		matched, _ := regexp.Match(, scanner.Bytes())
		}
	}
	return retarray, err
}

func GrepMutipleLines(data string, pat []byte) (retarray []string, err error) {
	allbytes, err := ioutil.ReadFile(data)
	if err != nil {
		return retarray, err
	}

	r, err := regexp.Compile(string(pat))
	if err != nil {
		return retarray, err
	}

	retarray = r.FindAllString(string(allbytes), -1)

	return retarray, err
}

func grepFFile(file string, pat []byte) bool {
	f, err := os.Open(file)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if bytes.Contains(scanner.Bytes(), pat) {
			return true
		}
	}
	return false
}

func GrepMatch(filepath, pattern string) bool {
	ret, err := grepFile(filepath, []byte(pattern))
	if len(ret) != 0 && err == nil {
		return true
	} else {
		return false
	}
}

func GrepF(filepath, pattern string) bool {
	return grepFFile(filepath, []byte(pattern))
}
