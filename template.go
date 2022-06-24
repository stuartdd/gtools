package main

import (
	"strings"
)

type Template struct {
	str   []byte
	pos   int
	max   int
	fetch func(string, string) string
}

func NewTemplate(str string, fetch func(string, string) string) *Template {
	b := []byte(str)
	return &Template{str: b, fetch: fetch, pos: 0, max: len(b)}
}

func TemplateParse(str string, fetch func(string, string) string) string {
	return NewTemplate(str, fetch).Parse()
}

func (t *Template) Parse() string {
	m := t.max
	if m == 0 {
		return ""
	}

	var c byte
	var sb strings.Builder

	p := 0
	cp := 0
	for p < m {
		c = t.str[p]
		if c != '%' {
			sb.WriteByte(c)
		} else {
			cp = t.parseName(&sb, p, c)
			if cp < 0 {
				sb.WriteByte(c)
			} else {
				p = cp
			}
		}
		p++
	}
	return sb.String()
}

func (t *Template) parseName(sb *strings.Builder, pos int, del byte) int {
	p := pos + 1
	m := t.max
	if p >= m {
		return -1
	}

	c := t.str[p]
	if c != '{' {
		return -1
	}

	var name strings.Builder

	p++
	for p < m {
		c = t.str[p]
		if c == '}' {
			s := t.fetch(name.String(), "")
			sb.WriteString(s)
			return p
		} else {
			name.WriteByte(c)
		}
		p++
	}
	return -1
}
