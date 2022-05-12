package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

type StringReader struct {
	id      int
	pos     int
	resp    []byte
	delay   bool
	delayMs int64
}

type FileWriter struct {
	fileName string
	file     *os.File
	canWrite bool
}

type myWriter struct {
	id int
}

func NewMyFileWriter(fileName string) (*FileWriter, error) {
	f, err := os.Create("/tmp/dat2")
	if err != nil {
		return nil, err
	}
	return &FileWriter{fileName: fileName, file: f, canWrite: true}, nil
}

func (mw *FileWriter) Close() {
	mw.canWrite = false
	mw.file.Close()
}

func (mw *FileWriter) Write(p []byte) (n int, err error) {
	mw.file.Write(p)
	return len(p), nil
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

func NewMyWriter(id int) *myWriter {
	return &myWriter{id: id}
}

func (mw *myWriter) WriteStr(s string) (n int, err error) {
	return mw.Write([]byte(s))
}

func (mw *myWriter) Write(p []byte) (n int, err error) {
	fmt.Printf("%s%s%s", prefix[mw.id], string(p), RESET)
	return len(p), nil
}
