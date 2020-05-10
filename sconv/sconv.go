package sconv

import (
	"bytes"
	//	"context"
	"errors"
	//	"fmt"
	//	"log"
	//	"os/exec"
	"strings"
	//	"syscall"
	//	"time"
	"unicode/utf8"
)

var (
	UnterminatedSingleQuoteError = errors.New("Unterminated single-quoted string")
	UnterminatedDoubleQuoteError = errors.New("Unterminated double-quoted string")
	UnterminatedEscapeError      = errors.New("Unterminated backslash-escape")
)

var (
	splitChars        = " \n\t"
	singleChar        = '\''
	doubleChar        = '"'
	escapeChar        = '\\'
	doubleEscapeChars = "$`\"\n\\"
)

// Split splits a string according to /bin/sh's word-splitting rules. It
// supports backslash-escapes, single-quotes, and double-quotes. Notably it does
// not support the $'' style of quoting. It also doesn't attempt to perform any
// other sort of expansion, including brace expansion, shell expansion, or
// pathname expansion.
//
// If the given input has an unterminated quoted string or ends in a
// backslash-escape, one of UnterminatedSingleQuoteError,
// UnterminatedDoubleQuoteError, or UnterminatedEscapeError is returned.
func StringToArgs(input string) (words []string, err error) {
	var buf bytes.Buffer
	words = make([]string, 0)

	for len(input) > 0 {
		// skip any splitChars at the start
		c, l := utf8.DecodeRuneInString(input)
		if strings.ContainsRune(splitChars, c) {
			input = input[l:]
			continue
		} else if c == escapeChar {
			// Look ahead for escaped newline so we can skip over it
			next := input[l:]
			if len(next) == 0 {
				err = UnterminatedEscapeError
				return
			}
			c2, l2 := utf8.DecodeRuneInString(next)
			if c2 == '\n' {
				input = next[l2:]
				continue
			}
		}

		var word string
		word, input, err = splitWord(input, &buf)
		if err != nil {
			return
		}
		words = append(words, word)
	}
	return
}

func splitWord(input string, buf *bytes.Buffer) (word string, remainder string, err error) {
	buf.Reset()

raw:
	{
		cur := input
		for len(cur) > 0 {
			c, l := utf8.DecodeRuneInString(cur)
			cur = cur[l:]
			if c == singleChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				input = cur
				goto single
			} else if c == doubleChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				input = cur
				goto double
			} else if c == escapeChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				input = cur
				goto escape
			} else if strings.ContainsRune(splitChars, c) {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				return buf.String(), cur, nil
			}
		}
		if len(input) > 0 {
			buf.WriteString(input)
			input = ""
		}
		goto done
	}

escape:
	{
		if len(input) == 0 {
			return "", "", UnterminatedEscapeError
		}
		c, l := utf8.DecodeRuneInString(input)
		if c == '\n' {
			// a backslash-escaped newline is elided from the output entirely
		} else {
			buf.WriteString(input[:l])
		}
		input = input[l:]
	}
	goto raw

single:
	{
		i := strings.IndexRune(input, singleChar)
		if i == -1 {
			return "", "", UnterminatedSingleQuoteError
		}
		buf.WriteString(input[0:i])
		input = input[i+1:]
		goto raw
	}

double:
	{
		cur := input
		for len(cur) > 0 {
			c, l := utf8.DecodeRuneInString(cur)
			cur = cur[l:]
			if c == doubleChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				input = cur
				goto raw
			} else if c == escapeChar {
				// bash only supports certain escapes in double-quoted strings
				c2, l2 := utf8.DecodeRuneInString(cur)
				cur = cur[l2:]
				if strings.ContainsRune(doubleEscapeChars, c2) {
					buf.WriteString(input[0 : len(input)-len(cur)-l-l2])
					if c2 == '\n' {
						// newline is special, skip the backslash entirely
					} else {
						buf.WriteRune(c2)
					}
					input = cur
				}
			}
		}
		return "", "", UnterminatedDoubleQuoteError
	}

done:
	return buf.String(), input, nil
}

func ArgsToString(args ...string) string {
	var buf bytes.Buffer
	for i, arg := range args {
		if i != 0 {
			buf.WriteByte(' ')
		}
		quote(arg, &buf)
	}
	return buf.String()
}

const (
	specialChars      = "\\'\"`${[|&;<>()*?!"
	extraSpecialChars = " \t\n"
	prefixChars       = "~"
)

func quote(word string, buf *bytes.Buffer) {
	// We want to try to produce a "nice" output. As such, we will
	// backslash-escape most characters, but if we encounter a space, or if we
	// encounter an extra-special char (which doesn't work with
	// backslash-escaping) we switch over to quoting the whole word. We do this
	// with a space because it's typically easier for people to read multi-word
	// arguments when quoted with a space rather than with ugly backslashes
	// everywhere.
	origLen := buf.Len()

	if len(word) == 0 {
		// oops, no content
		buf.WriteString("''")
		return
	}

	cur, prev := word, word
	atStart := true
	for len(cur) > 0 {
		c, l := utf8.DecodeRuneInString(cur)
		cur = cur[l:]
		if strings.ContainsRune(specialChars, c) || (atStart && strings.ContainsRune(prefixChars, c)) {
			// copy the non-special chars up to this point
			if len(cur) < len(prev) {
				buf.WriteString(prev[0 : len(prev)-len(cur)-l])
			}
			buf.WriteByte('\\')
			buf.WriteRune(c)
			prev = cur
		} else if strings.ContainsRune(extraSpecialChars, c) {
			// start over in quote mode
			buf.Truncate(origLen)
			goto quote
		}
		atStart = false
	}
	if len(prev) > 0 {
		buf.WriteString(prev)
	}
	return

quote:
	// quote mode
	// Use single-quotes, but if we find a single-quote in the word, we need
	// to terminate the string, emit an escaped quote, and start the string up
	// again
	inQuote := false
	for len(word) > 0 {
		i := strings.IndexRune(word, '\'')
		if i == -1 {
			break
		}
		if i > 0 {
			if !inQuote {
				buf.WriteByte('\'')
				inQuote = true
			}
			buf.WriteString(word[0:i])
		}
		word = word[i+1:]
		if inQuote {
			buf.WriteByte('\'')
			inQuote = false
		}
		buf.WriteString("\\'")
	}
	if len(word) > 0 {
		if !inQuote {
			buf.WriteByte('\'')
		}
		buf.WriteString(word)
		buf.WriteByte('\'')
	}
}
