package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	STD_OUT = 0
	STD_ERR = 1
	STD_IN  = 2

	FILE_APPEND_PREF = "append:"
	CACHE_PREF       = "memory:"
)

var (
	stdColourPrefix = []string{GREEN, RED}
	outCache        = make(map[string]*CacheWriter)
)

type StringReader struct {
	pos     int
	resp    []byte
	delay   bool
	delayMs int64
}

type FileWriter struct {
	fileName string
	file     *os.File
	canWrite bool
	stdErr   *MyWriter
	stdOut   *MyWriter
}
type CacheWriter struct {
	name string
	sb   strings.Builder
}

type MyWriter struct {
	id int
}

func NewMyWriter(id int) *MyWriter {
	return &MyWriter{id: id}
}

func NewCacheWriter(name string) (*CacheWriter, error) {
	if name == "" {
		return nil, fmt.Errorf("memory writer must have a name")
	}
	return &CacheWriter{name: name}, nil
}

func (mw *CacheWriter) Write(p []byte) (n int, err error) {
	mw.sb.Write(p)
	return len(p), nil
}

func (mw *MyWriter) Write(p []byte) (n int, err error) {
	fmt.Printf("%s%s%s", stdColourPrefix[mw.id], string(p), RESET)
	return len(p), nil
}

func NewWriter(fileName string, defaultOut, stdErr *MyWriter) io.Writer {
	if fileName == "" {
		return defaultOut
	}
	var err error
	var fn string
	if strings.ToLower(fileName)[0:7] == CACHE_PREF {
		fn = fileName[len(CACHE_PREF):]
		cw, found := outCache[fn]
		if !found {
			cw, err = NewCacheWriter(fn)
			if err != nil {
				stdErr.Write([]byte(fmt.Sprintf("Failed to create '%s' writer '%s'. '%s'", CACHE_PREF, fn, err.Error())))
				return defaultOut
			}
			outCache[fn] = cw
		}
		return cw
	}

	var f *os.File
	if strings.ToLower(fileName)[0:7] == FILE_APPEND_PREF {
		fn = fileName[len(FILE_APPEND_PREF):]
		f, err = os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	} else {
		fn = fileName
		f, err = os.Create(fn)
	}
	if err != nil {
		stdErr.Write([]byte(fmt.Sprintf("Failed to create file writer %s. %s", fn, err.Error())))
		return defaultOut
	}
	return &FileWriter{fileName: fn, file: f, canWrite: true, stdOut: defaultOut, stdErr: stdErr}
}

func (mw *FileWriter) Close() error {
	mw.canWrite = false
	if mw.file != nil {
		return mw.file.Close()
	}
	return nil
}

func (mw *FileWriter) Write(p []byte) (n int, err error) {
	if mw.canWrite {
		n, err = mw.file.Write(p)
		if err != nil {
			mw.stdErr.Write([]byte(fmt.Sprintf("Write Error. File:%s. Err:%s\n", mw.fileName, err.Error())))
		} else {
			return n, nil
		}
	}
	return mw.stdOut.Write(p)
}

func NewStringReader(s string, defaultIn io.Reader) io.Reader {
	if s == "" {
		return defaultIn
	}
	if strings.ToLower(s)[0:7] == CACHE_PREF {
		fn := s[len(CACHE_PREF):]
		cw, found := outCache[fn]
		if found {
			return &StringReader{resp: []byte(cw.sb.String()), delayMs: 0}
		}
	}
	return &StringReader{resp: []byte(s), delayMs: 0}
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
