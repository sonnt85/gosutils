package gogrep

import (
	"bufio"
	"bytes"
	"os"
	"regexp"
)

func grepFile(file string, pat []byte) bool {
	f, err := os.Open(file)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		matched, _ := regexp.Match(string(pat), scanner.Bytes())
		if matched {
			return true
		}
	}
	return false
}

func grepfFile(file string, pat []byte) bool {
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

func Grep(filepath, pattern string) bool {
	return grepFile(filepath, []byte(pattern))
}

func Grepf(filepath, pattern string) bool {
	return grepfFile(filepath, []byte(pattern))
}
