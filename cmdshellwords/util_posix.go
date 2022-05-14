//go:build !windows
// +build !windows

package cmdshellwords

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
)

func parser(line string, p *Parser) ([]string, error) {
	args := []string{}
	buf := ""
	var escaped, doubleQuoted, singleQuoted, backQuote, dollarQuote bool
	backtick := ""

	pos := -1
	got := false

loop:
	for i, r := range line {
		if escaped {
			buf += string(r)
			escaped = false
			continue
		}

		if r == '\\' {
			if singleQuoted {
				buf += string(r)
			} else {
				escaped = true
			}
			continue
		}

		if isSpace(r) {
			if singleQuoted || doubleQuoted || backQuote || dollarQuote {
				buf += string(r)
				backtick += string(r)
			} else if got {
				if p.ParseEnv {
					parser := &Parser{ParseEnv: false, ParseBacktick: false, Position: 0, Dir: p.Dir}
					strs, err := parser.Parse(replaceEnv(p.Getenv, buf))
					if err != nil {
						return nil, err
					}
					for _, str := range strs {
						args = append(args, str)
					}
				} else {
					args = append(args, buf)
				}
				buf = ""
				got = false
			}
			continue
		}

		switch r {
		case '`':
			if !singleQuoted && !doubleQuoted && !dollarQuote {
				if p.ParseBacktick {
					if backQuote {
						out, err := shellRun(backtick, p.Dir)
						if err != nil {
							return nil, err
						}
						buf = buf[:len(buf)-len(backtick)] + out
					}
					backtick = ""
					backQuote = !backQuote
					continue
				}
				backtick = ""
				backQuote = !backQuote
			}
		case ')':
			if !singleQuoted && !doubleQuoted && !backQuote {
				if p.ParseBacktick {
					if dollarQuote {
						out, err := shellRun(backtick, p.Dir)
						if err != nil {
							return nil, err
						}
						buf = buf[:len(buf)-len(backtick)-2] + out
					}
					backtick = ""
					dollarQuote = !dollarQuote
					continue
				}
				backtick = ""
				dollarQuote = !dollarQuote
			}
		case '(':
			if !singleQuoted && !doubleQuoted && !backQuote {
				if !dollarQuote && strings.HasSuffix(buf, "$") {
					dollarQuote = true
					buf += "("
					continue
				} else {
					return nil, errors.New("invalid command line string")
				}
			}
		case '"':
			if !singleQuoted && !dollarQuote {
				if doubleQuoted {
					got = true
				}
				doubleQuoted = !doubleQuoted
				continue
			}
		case '\'':
			if !doubleQuoted && !dollarQuote {
				if singleQuoted {
					got = true
				}
				singleQuoted = !singleQuoted
				continue
			}
		case ';', '&', '|', '<', '>':
			if !(escaped || singleQuoted || doubleQuoted || backQuote || dollarQuote) {
				if r == '>' && len(buf) > 0 {
					if c := buf[0]; '0' <= c && c <= '9' {
						i -= 1
						got = false
					}
				}
				pos = i
				break loop
			}
		}

		got = true
		buf += string(r)
		if backQuote || dollarQuote {
			backtick += string(r)
		}
	}

	if got {
		if p.ParseEnv {
			parser := &Parser{ParseEnv: false, ParseBacktick: false, Position: 0, Dir: p.Dir}
			strs, err := parser.Parse(replaceEnv(p.Getenv, buf))
			if err != nil {
				return nil, err
			}
			for _, str := range strs {
				args = append(args, str)
			}
		} else {
			args = append(args, buf)
		}
	}

	if escaped || singleQuoted || doubleQuoted || backQuote || dollarQuote {
		return nil, errors.New("invalid command line string")
	}

	p.Position = pos

	return args, nil
}

func join(words ...string) string {
	var buf bytes.Buffer
	for i, w := range words {
		if i != 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(Escape(w))
	}
	return buf.String()
}

func shellRun(line, dir string) (string, error) {
	var shell string
	if shell = os.Getenv("SHELL"); shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell, "-c", line)
	if dir != "" {
		cmd.Dir = dir
	}
	b, err := cmd.Output()
	if err != nil {
		if eerr, ok := err.(*exec.ExitError); ok {
			b = eerr.Stderr
		}
		return "", errors.New(err.Error() + ":" + string(b))
	}
	return strings.TrimSpace(string(b)), nil
}
