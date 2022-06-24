package cmdshellwords

import (
	"bytes"
	"regexp"
)

var reNeedEscape = regexp.MustCompile(`([^A-Za-z0-9_\-.,:\/@\n])`)
var reLF = regexp.MustCompile(`\n`)

// Split a string into an array of tokens in the same way the UNIX Bourne shell does.
func Split(line string) ([]string, error) {
	return Parse(line)
}

func SplitPosix(line string) ([]string, error) {
	return ParsePosix(line)
}

// Join builds a command line string from an argument list by joining
// all elements escaped for Bourne shell and separated by a space.
func Join(words ...string) string {
	return join(words...)
}

func JoinPosix(words ...string) string {
	return joinposix(words...)
}

// Escape escapes a string so that it can be safely used in a Bourne shell command line.
// Note that a resulted string should be used unquoted and is not intended for use in double quotes nor in single quotes.
func Escape(str string) string {
	if str == "" {
		return "''"
	}

	strBytes := []byte(str)
	var buf bytes.Buffer
	for _, b := range strBytes {
		switch b {
		case
			'a', 'b', 'c', 'd', 'e', 'f', 'g',
			'h', 'i', 'j', 'k', 'l', 'm', 'n',
			'o', 'p', 'q', 'r', 's', 't', 'u',
			'v', 'w', 'x', 'y', 'z',
			'A', 'B', 'C', 'D', 'E', 'F', 'G',
			'H', 'I', 'J', 'K', 'L', 'M', 'N',
			'O', 'P', 'Q', 'R', 'S', 'T', 'U',
			'V', 'W', 'X', 'Y', 'Z',
			'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
			'_', '-', '.', ',', ':', '/', '@':
			buf.WriteByte(b)
		case '\n':
			buf.WriteString("'\n'")
		default:
			buf.WriteByte('\\')
			buf.WriteByte(b)
		}
	}

	return buf.String()
}
