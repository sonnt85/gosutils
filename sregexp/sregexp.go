// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lazyregexp is a thin wrapper over regexp, allowing the use of global
// regexp variables without forcing them to be compiled at init.
package sregexp

import (
	//	"os"
	goregexp "regexp"
	//	"strings"
	"sync"
)

// Regexp is a wrapper around goregexp.Regexp, where the underlying regexp will be
// compiled the first time it is needed.
type Regexp struct {
	pattern string
	once    sync.Once
	rx      *goregexp.Regexp
}

//compile pattern only once, can call mutiple times
func (r *Regexp) Regexp() *goregexp.Regexp {
	r.once.Do(r.build)
	return r.rx
}

//compile pattern
func (r *Regexp) build() {
	r.rx = goregexp.MustCompile(r.pattern)
	r.pattern = ""
}

// Find returns a slice holding the text of the leftmost match in b of the regular expression.
// A return value of nil indicates no match.
func (r *Regexp) Find(b []byte) []byte {
	return r.Regexp().Find(b)
}

// FindSubmatch returns a slice of slices holding the text of the leftmost
// match of the regular expression in b and the matches, if any, of its
// subexpressions, as defined by the 'Submatch' descriptions in the package
// comment.
// A return value of nil indicates no match
func (r *Regexp) FindSubmatch(s []byte) [][]byte {
	return r.Regexp().FindSubmatch(s)
}

// FindStringSubmatch returns a slice of strings holding the text of the
// leftmost match of the regular expression in s and the matches, if any, of
// its subexpressions, as defined by the 'Submatch' description in the
// package comment.
// A return value of nil indicates no match.
func (r *Regexp) FindStringSubmatch(s string) []string {
	return r.Regexp().FindStringSubmatch(s)
}

// FindStringSubmatchIndex returns a slice holding the index pairs
// identifying the leftmost match of the regular expression in s and the
// matches, if any, of its subexpressions, as defined by the 'Submatch' and
// 'Index' descriptions in the package comment.
// A return value of nil indicates no match.
func (r *Regexp) FindStringSubmatchIndex(s string) []int {
	return r.Regexp().FindStringSubmatchIndex(s)
}

// ReplaceAllString returns a copy of src, replacing matches of the Regexp
// with the replacement string repl. Inside repl, $ signs are interpreted as
// in Expand, so for instance $1 represents the text of the first submatch.
func (r *Regexp) ReplaceAllString(src, repl string) string {
	return r.Regexp().ReplaceAllString(src, repl)
}

// FindString returns a string holding the text of the leftmost match in s of the regular
// expression. If there is no match, the return value is an empty string,
// but it will also be empty if the regular expression successfully matches
// an empty string. Use FindStringIndex or FindStringSubmatch if it is
// necessary to distinguish these cases.
func (r *Regexp) FindString(s string) string {
	return r.Regexp().FindString(s)
}

// FindAll is the 'All' version of Find; it returns a slice of all successive
// matches of the expression, as defined by the 'All' description in the
// package comment.
// A return value of nil indicates no match.
func (r *Regexp) FindAll(b []byte, n int) [][]byte {
	return r.Regexp().FindAll(b, n)
}

// FindAllString is the 'All' version of FindString; it returns a slice of all
// successive matches of the expression, as defined by the 'All' description
// in the package comment.
// A return value of nil indicates no match.
func (r *Regexp) FindAllString(s string, n int) []string {
	return r.Regexp().FindAllString(s, n)
}

// Match reports whether the byte slice b
// contains any match of the regular expression re.
func (r *Regexp) MatchString(s string) bool {
	return r.Regexp().MatchString(s)
}

// SubexpNames returns the names of the parenthesized subexpressions
// in this Regexp. The name for the first sub-expression is names[1],
// so that if m is a match slice, the name for m[i] is SubexpNames()[i].
// Since the Regexp as a whole cannot be named, names[0] is always
// the empty string. The slice should not be modified.
func (r *Regexp) SubexpNames() []string {
	return r.Regexp().SubexpNames()
}

// FindAllStringSubmatch is the 'All' version of FindStringSubmatch; it
// returns a slice of all successive matches of the expression, as defined by
// the 'All' description in the package comment.
// A return value of nil indicates no match.
func (r *Regexp) FindAllStringSubmatch(s string, n int) [][]string {
	return r.Regexp().FindAllStringSubmatch(s, n)
}

// Split slices s into substrings separated by the expression and returns a slice of
// the substrings between those expression matches.
//
// The slice returned by this method consists of all the substrings of s
// not contained in the slice returned by FindAllString. When called on an expression
// that contains no metacharacters, it is equivalent to strings.SplitN.
//
// Example:
//   s := regexp.MustCompile("a*").Split("abaabaccadaaae", 5)
//   // s: ["", "b", "b", "c", "cadaaae"]
//
// The count determines the number of substrings to return:
//   n > 0: at most n substrings; the last substring will be the unsplit remainder.
//   n == 0: the result is nil (zero substrings)
//   n < 0: all substrings
func (r *Regexp) Split(s string, n int) []string {
	return r.Regexp().Split(s, n)
}

// ReplaceAllLiteralString returns a copy of src, replacing matches of the Regexp
// with the replacement string repl. The replacement repl is substituted directly,
// without using Expand.
func (r *Regexp) ReplaceAllLiteralString(src, repl string) string {
	return r.Regexp().ReplaceAllLiteralString(src, repl)
}

// FindAllIndex is the 'All' version of FindIndex; it returns a slice of all
// successive matches of the expression, as defined by the 'All' description
// in the package comment.
// A return value of nil indicates no match.
func (r *Regexp) FindAllIndex(b []byte, n int) [][]int {
	return r.Regexp().FindAllIndex(b, n)
}

// Match reports whether the byte slice b
// contains any match of the regular expression re.
func (r *Regexp) Match(b []byte) bool {
	return r.Regexp().Match(b)
}

// ReplaceAllStringFunc returns a copy of src in which all matches of the
// Regexp have been replaced by the return value of function repl applied
// to the matched substring. The replacement returned by repl is substituted
// directly, without using Expand.
func (r *Regexp) ReplaceAllStringFunc(src string, repl func(string) string) string {
	return r.Regexp().ReplaceAllStringFunc(src, repl)
}

// ReplaceAll returns a copy of src, replacing matches of the Regexp
// with the replacement text repl. Inside repl, $ signs are interpreted as
// in Expand, so for instance $1 represents the text of the first submatch.
func (r *Regexp) ReplaceAll(src, repl []byte) []byte {
	return r.Regexp().ReplaceAll(src, repl)
}

// New creates a new lazy regexp, delaying the compiling work until it is first
// needed. If the code is being run as part of tests, the regexp compiling will
// happen immediately.
func New(pattern string) *Regexp {
	lr := &Regexp{pattern: pattern}
	//lr.Regexp()
	return lr
}
