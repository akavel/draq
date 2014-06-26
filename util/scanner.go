package util

import (
	"bufio"
	"io"
)

type Scanner struct {
	Offset int64
	s      *bufio.Scanner
}

func NewScanner(r io.Reader) *Scanner {
	s := Scanner{
		s: bufio.NewScanner(r),
	}
	s.s.Split(s.split)
	return &s
}

func (s *Scanner) split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	advance, token, err = bufio.ScanWords(data, atEOF)
	s.Offset += int64(advance)
	return
}

func (s *Scanner) Next() (word string, err error) {
	if s.s.Scan() {
		return s.s.Text(), nil
	}
	if s.s.Err() == nil {
		return "", io.EOF
	}
	return "", s.s.Err()
}
