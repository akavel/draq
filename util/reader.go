package util

import (
	"io"
	"text/scanner"
)

type Scanner struct {
	S scanner.Scanner
}

func NewScanner(r io.Reader) *Scanner {
	s := Scanner{}
	s.S.Init(r)
	return &s
}

func (s *Scanner) Cmd() (cmd string, eof bool) {
	t := s.S.Scan()
	if t == scanner.EOF {
		return "", true
	}
	return s.S.TokenText(), false
}
