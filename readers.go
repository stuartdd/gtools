package main

import (
	"fmt"
	"io"
	"os"
	"strings"
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
	stdErr   *myWriter
	stdOut   *myWriter
}

type myWriter struct {
	id int
}

func NewMyFileWriter(fileName string, stdOut, stdErr *myWriter) *FileWriter {
	var f *os.File
	var err error
	var fn string
	if strings.ToLower(fileName)[0:7] == "append:" {
		fn = fileName[7:]
		f, err = os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	} else {
		fn = fileName
		f, err = os.Create(fn)
	}
	if err != nil {
		stdErr.WriteStr(fmt.Sprintf("Failed to create output file %s. %s", fn, err.Error()))
		return &FileWriter{fileName: fn, file: nil, canWrite: false, stdOut: stdOut, stdErr: stdErr}
	}
	return &FileWriter{fileName: fn, file: f, canWrite: true, stdOut: stdOut, stdErr: stdErr}
}

func (mw *FileWriter) Close() {
	mw.canWrite = false
	if mw.file != nil {
		mw.file.Close()
	}
}

func (mw *FileWriter) Write(p []byte) (n int, err error) {
	if mw.canWrite {
		n, err = mw.file.Write(p)
		if err != nil {
			mw.stdErr.WriteStr(fmt.Sprintf("Write Error. File:%s. Err:%s\n", mw.fileName, err.Error()))
			return mw.stdOut.Write(p)
		}
		return n, nil
	}
	return mw.stdOut.Write(p)
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
