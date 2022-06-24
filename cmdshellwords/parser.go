package cmdshellwords

import (
	"os"
	"regexp"
)

var (
	ParseEnv      bool = false
	ParseBacktick bool = false
)

var envRe = regexp.MustCompile(`\$({[a-zA-Z0-9_]+}|[a-zA-Z0-9_]+)`)

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

func replaceEnv(getenv func(string) string, s string) string {
	if getenv == nil {
		getenv = os.Getenv
	}

	return envRe.ReplaceAllStringFunc(s, func(s string) string {
		s = s[1:]
		if s[0] == '{' {
			s = s[1 : len(s)-1]
		}
		return getenv(s)
	})
}

type Parser struct {
	ParseEnv      bool
	ParseBacktick bool
	Position      int
	Dir           string

	// If ParseEnv is true, use this for getenv.
	// If nil, use os.Getenv.
	Getenv func(string) string
}

func NewParser() *Parser {
	return &Parser{
		ParseEnv:      ParseEnv,
		ParseBacktick: ParseBacktick,
		Position:      0,
		Dir:           "",
	}
}

func (p *Parser) Parse(line string) ([]string, error) {
	return parser(line, p)
}

func Parse(line string) ([]string, error) {
	return NewParser().Parse(line)
}

func (p *Parser) ParsePosix(line string) ([]string, error) {
	return parserposix(line, p)
}

func ParsePosix(line string) ([]string, error) {
	return NewParser().ParsePosix(line)
}
