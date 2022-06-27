package main

import (
	"strings"
)

type Template struct {
	str   []byte
	pos   int
	max   int
	fetch func(string) (string, error)
}

func NewTemplate(str string, fetch func(string) (string, error)) *Template {
	b := []byte(str)
	return &Template{str: b, fetch: fetch, pos: 0, max: len(b)}
}

func TemplateParse(str string, fetch func(string) (string, error)) (string, error) {
	return NewTemplate(str, fetch).Parse()
}

func (t *Template) Parse() (string, error) {
	m := t.max
	if m == 0 {
		return "", nil
	}

	var c byte
	var sb strings.Builder
	var err error
	p := 0
	cp := 0

	for p < m {
		c = t.str[p]
		if c != '%' {
			sb.WriteByte(c)
		} else {
			cp, err = t.parseName(&sb, p, c)
			if err != nil {
				return string(t.str), err
			}
			if cp < 0 {
				sb.WriteByte(c)
			} else {
				p = cp
			}
		}
		p++
	}
	return sb.String(), nil
}

func (t *Template) parseName(sb *strings.Builder, pos int, del byte) (int, error) {
	p := pos + 1
	m := t.max
	if p >= m {
		return -1, nil
	}

	c := t.str[p]
	if c != '{' {
		return -1, nil
	}

	var name strings.Builder

	p++
	for p < m {
		c = t.str[p]
		if c == '}' {
			s, err := t.fetch(name.String())
			if err != nil {
				return -1, err
			}
			sb.WriteString(s)
			return p, nil
		} else {
			name.WriteByte(c)
		}
		p++
	}
	return -1, nil
}
