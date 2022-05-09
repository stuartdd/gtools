package main

import (
	"io"
	"time"
)

type StringReader struct {
	id      int
	pos     int
	resp    []byte
	delay   bool
	delayMs int64
}

func NewStringReader(id int, s string) *StringReader {
	return &StringReader{id: id, resp: []byte(s), delayMs: 0, delay: false}
}

func (mr *StringReader) Read(p []byte) (n int, err error) {
	if mr.delay {
		time.Sleep(time.Millisecond * time.Duration(mr.delayMs))
		mr.delay = false
	}
	i := len(mr.resp) - mr.pos
	if len(p) < i {
		i = len(p)
	}
	j := 0
	for ; j < i; j++ {
		p[j] = mr.resp[mr.pos]
		mr.pos++
		if p[j] == '\n' {
			j++
			mr.delay = true
			break
		}
	}
	if i <= 0 {
		return 0, io.EOF
	}
	return j, nil
}
