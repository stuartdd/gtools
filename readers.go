package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type EncReader interface {
	SetKey(string)
}

type StringReader struct {
	pos     int
	resp    string
	key     string
	delay   bool
	delayMs int64
	typ     ENUM_MEM_TYPE
}

func NewStringReader(selectFrom string, defaultIn io.Reader, dataCache *DataCache) (io.Reader, error) {
	if selectFrom == "" {
		return defaultIn, nil
	}
	fn, typ, found := PrefixMatch(selectFrom, HTTP_PREF, HTTP_TYPE)
	if found {
		parts := strings.SplitN(fn, "|", 2)
		if len(parts) == 0 || len(parts[0]) == 0 {
			return nil, fmt.Errorf("no http url name after %s prefix of 'in' parameter", MEMORY_PREF)
		}
		resp, err := HttpGet(parts[0])
		if err != nil {
			return nil, err
		}
		filter := ""
		if len(parts) > 1 {
			filter = parts[1]
		}
		fresp, err := Filter([]byte(resp), filter)
		if err != nil {
			return nil, err
		}
		return &StringReader{resp: string(fresp), delayMs: 0, pos: 0, typ: typ, key: ""}, nil
	}

	fn, typ, found = PrefixMatch(selectFrom, MEMORY_PREF, MEM_TYPE)
	if found {
		parts := strings.SplitN(fn, "|", 2)
		if len(parts) == 0 || len(parts[0]) == 0 {
			return nil, fmt.Errorf("no cache name after %s prefix of 'in' parameter", MEMORY_PREF)
		}
		cw := dataCache.GetCacheWriter(parts[0])
		if cw != nil {
			filter := ""
			if len(parts) > 1 {
				filter = parts[1]
			}
			resp, err := Filter([]byte(cw.sb.String()), filter)
			if err != nil {
				return nil, err
			}
			return &StringReader{resp: string(resp), delayMs: 0, pos: 0, typ: typ, key: ""}, nil
		} else {
			return nil, fmt.Errorf("could not locate cache entry for in parameter %s.%s", MEMORY_PREF, parts[0])
		}
	}

	fn, typ, found = PrefixMatch(selectFrom, FILE_PREF, FILE_TYPE)
	if found {
		parts := strings.SplitN(fn, "|", 2)
		if len(parts) == 0 || len(parts[0]) == 0 {
			return nil, fmt.Errorf("could not locate file name after %s prefix of 'in' parameter", FILE_PREF)
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
		return &StringReader{resp: string(resp), delayMs: 0, pos: 0, typ: typ, key: ""}, nil
	}
	return &StringReader{resp: selectFrom, delayMs: 0, pos: 0, typ: STR_TYPE, key: ""}, nil
}

func (sr *StringReader) SetKey(key string) {
	sr.key = key
}

func (sr *StringReader) Read(p []byte) (n int, err error) {
	if sr.delay {
		time.Sleep(time.Millisecond * time.Duration(sr.delayMs))
		sr.delay = false
	}
	if sr.key != "" && sr.pos == 0 {
		tmp, err := DecryptData([]byte(sr.key), []byte(sr.resp))
		if err != nil {
			return 0, err
		}
		sr.resp = string(tmp)
		sr.key = ""
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
		return []byte{}, fmt.Errorf("failed to load file '%s'", fileName)
	}
	return dat, nil
}
