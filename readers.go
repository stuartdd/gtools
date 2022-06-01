package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type StringReader struct {
	pos     int
	resp    string
	delay   bool
	delayMs int64
}

func NewStringReader(selectFrom string, defaultIn io.Reader) (io.Reader, error) {
	if selectFrom == "" {
		return defaultIn, nil
	}

	fn, _, found := PrefixMatch(selectFrom, MEMORY_PREF, MEM_TYPE)
	if found {
		parts := strings.SplitN(fn, "|", 2)
		if len(parts) == 0 || len(parts[0]) == 0 {
			return nil, fmt.Errorf("no cache name after %s prefix of in parameter", MEMORY_PREF)
		}
		cw := ReadFromMemory(parts[0])
		if cw != nil {
			filter := ""
			if len(parts) > 1 {
				filter = parts[1]
			}
			resp, err := Filter([]byte(cw.sb.String()), filter)
			if err != nil {
				return nil, err
			}
			return &StringReader{resp: string(resp), delayMs: 0}, nil
		} else {
			return nil, fmt.Errorf("could not locate cache entry for in parameter %s.%s", MEMORY_PREF, parts[0])
		}
	}

	fn, _, found = PrefixMatch(selectFrom, FILE_APPEND_PREF, FILE_TYPE)
	if found {
		parts := strings.SplitN(fn, "|", 2)
		if len(parts) == 0 || len(parts[0]) == 0 {
			return nil, fmt.Errorf("could not locate file name after %s prefix of in parameter", FILE_APPEND_PREF)
		}
		data, err := readFile(parts[0])
		if err != nil {
			return nil, err
		}
		filter := ""
		if len(parts) > 1 {
			filter = parts[1]
		}
		resp, err := Filter(data, filter)
		if err != nil {
			return nil, err
		}
		return &StringReader{resp: string(resp), delayMs: 0}, nil
	}
	return &StringReader{resp: selectFrom, delayMs: 0}, nil
}

func (sr *StringReader) Read(p []byte) (n int, err error) {
	if sr.delay {
		time.Sleep(time.Millisecond * time.Duration(sr.delayMs))
		sr.delay = false
	}
	i := len(sr.resp) - sr.pos
	if len(p) < i {
		i = len(p)
	}
	j := 0
	for ; j < i; j++ {
		p[j] = sr.resp[sr.pos]
		sr.pos++
		if p[j] == '\n' {
			j++
			sr.delay = true
			break
		}
	}
	if i <= 0 {
		return 0, io.EOF
	}
	return j, nil
}

func readFile(fileName string) ([]byte, error) {
	dat, err := os.ReadFile(fileName)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to load file '%s' from file input definition", fileName)
	}
	return dat, nil
}
