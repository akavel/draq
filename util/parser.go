package util

import (
	"errors"
	"image/color"
	"io"
	"strconv"
)

type Parser struct {
	*Scanner
}

func (p *Parser) Cmd() (cmd string, eof bool) {
	t, err := p.Next()
	if err == io.EOF {
		return "", true
	}
	return t, false
}

func (p *Parser) Color() (c color.RGBA, err error) {
	var word string
	word, err = p.Next()
	if err != nil {
		return
	}

	switch len(word) {
	case 6:
		word += "ff"
	case 8:
		// ok
	default:
		err = errors.New("COLOR must be RRGGBB or RRGGBBAA, got \"" + word + "\"")
		return
	}

	cc, err := strconv.ParseUint(word, 16, 32)
	if err != nil {
		return
	}

	return color.RGBA{
		R: uint8(cc >> 24),
		G: uint8(cc >> 16),
		B: uint8(cc >> 8),
		A: uint8(cc),
	}, nil
}

func (p *Parser) Coord() (x int, err error) {
	var s string
	s, err = p.Next()
	if err != nil {
		return
	}
	return strconv.Atoi(s)
}

func (p *Parser) Point() (x, y float64, err error) {
	xx, err := p.Coord()
	if err != nil {
		return 0, 0, errors.New("COORD 1: " + err.Error())
	}
	yy, err := p.Coord()
	if err != nil {
		return 0, 0, errors.New("COORD 2: " + err.Error())
	}
	return float64(xx), float64(yy), err
}
