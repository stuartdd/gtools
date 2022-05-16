package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	STD_OUT = 0
	STD_ERR = 1
	STD_IN  = 2

	FILE_APPEND_PREF = "append:"
	CACHE_PREF       = "memory:"
	FILE_PREF        = "file:"
)

var (
	stdColourPrefix = []string{GREEN, RED}
	OutputCache     = make(map[string]*CacheWriter, 10)
)

type Reset interface {
	Reset()
}

type StringReader struct {
	pos     int
	resp    string
	delay   bool
	delayMs int64
}

type FileWriter struct {
	fileName string
	filter   string
	file     *os.File
	canWrite bool
	stdErr   *MyWriter
	stdOut   *MyWriter
}

type CacheWriter struct {
	name   string
	filter string
	desc   string
	sb     strings.Builder
}

func InitCache() {
	OutputCache = make(map[string]*CacheWriter, 10)
}

func WriteCache(cw *CacheWriter) {
	OutputCache[cw.name] = cw
}

func ReadCache(name string) *CacheWriter {
	c, ok := OutputCache[name]
	if ok {
		return c
	}
	return nil
}

func MutateStringFromMemCache(in string) string {
	out := in
	for n, v := range OutputCache {
		out = strings.Replace(out, fmt.Sprintf("%%{%s}", n), strings.TrimSpace(v.sb.String()), -1)
	}
	return out
}

func MutateListFromMemCache(in []string) []string {
	out := make([]string, 0)
	for _, a := range in {
		out = append(out, MutateStringFromMemCache(a))
	}
	return out
}

type MyWriter struct {
	id     int
	filter string
}

func NewMyWriter(id int) *MyWriter {
	return &MyWriter{id: id, filter: ""}
}

func NewMyFilterWriter(id int, filter string) *MyWriter {
	return &MyWriter{id: id, filter: filter}
}

func (mw *MyWriter) Write(p []byte) (n int, err error) {
	pLen := len(p)
	if mw.filter != "" {
		p, err = filter(p, mw.filter)
		if err != nil {
			return 0, err
		}
	}
	fmt.Printf("%s%s%s", stdColourPrefix[mw.id], string(p), RESET)
	return pLen, nil
}

func NewCacheWriterValue(name, desc, value string) (*CacheWriter, error) {
	if name == "" {
		return nil, fmt.Errorf("memory writer must have a name")
	}
	var sb strings.Builder
	sb.WriteString(value)
	cw := &CacheWriter{name: name, filter: "", desc: desc, sb: sb}
	return cw, nil
}

func NewCacheWriter(name, filter string) (*CacheWriter, error) {
	if name == "" {
		return nil, fmt.Errorf("memory writer must have a name")
	}
	var sb strings.Builder
	cw := &CacheWriter{name: name, filter: filter, desc: "", sb: sb}
	return cw, nil
}

func (cw *CacheWriter) Write(p []byte) (n int, err error) {
	pLen := len(p)
	if cw.filter != "" {
		p, err = filter(p, cw.filter)
		if err != nil {
			return 0, err
		}
	}
	np, errp := cw.sb.Write(p)
	if errp != nil {
		return np, err
	}
	return pLen, nil
}

func (cw *CacheWriter) Reset() {
	cw.sb.Reset()
}

func PrefixMatch(s string, pref string) (string, bool) {
	if len(s) > len(pref) && strings.ToLower(s)[0:len(pref)] == pref {
		return s[len(pref):], true
	}
	return s, false
}

func NewWriter(fileName, filter string, defaultOut, stdErr *MyWriter) io.Writer {
	if fileName == "" {
		if filter == "" {
			return defaultOut
		}
		return NewMyFilterWriter(defaultOut.id, filter)
	}
	var err error
	var fn string

	fn, found := PrefixMatch(fileName, CACHE_PREF)
	if found {
		cw := ReadCache(fn)
		if cw == nil {
			cw, err = NewCacheWriter(fn, filter)
			if err != nil {
				stdErr.Write([]byte(fmt.Sprintf("Failed to create '%s' writer '%s'. '%s'", CACHE_PREF, fn, err.Error())))
				return defaultOut
			}
			WriteCache(cw)
		}
		return cw
	}

	var f *os.File
	fn, found = PrefixMatch(fileName, FILE_APPEND_PREF)
	if found {
		f, err = os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	} else {
		f, err = os.Create(fn)
	}
	if err != nil {
		stdErr.Write([]byte(fmt.Sprintf("Failed to create file writer %s. %s", fn, err.Error())))
		return defaultOut
	}
	return &FileWriter{fileName: fn, file: f, filter: filter, canWrite: true, stdOut: defaultOut, stdErr: stdErr}
}

func (fw *FileWriter) Close() error {
	fw.canWrite = false
	if fw.file != nil {
		return fw.file.Close()
	}
	return nil
}

func (fw *FileWriter) Write(p []byte) (n int, err error) {
	if fw.canWrite {
		pLen := len(p)
		if fw.filter != "" {
			p, err = filter(p, fw.filter)
			if err != nil {
				return 0, err
			}

		}
		_, err = fw.file.Write(p)
		if err != nil {
			fw.stdErr.Write([]byte(fmt.Sprintf("Write Error. File:%s. Err:%s\n", fw.fileName, err.Error())))
		} else {
			return pLen, nil
		}
	}
	return fw.stdOut.Write(p)
}

func NewStringReader(selectFrom string, defaultIn io.Reader) (io.Reader, error) {
	if selectFrom == "" {
		return defaultIn, nil
	}

	fn, found := PrefixMatch(selectFrom, CACHE_PREF)
	if found {
		parts := strings.Split(fn, "|")
		cw := ReadCache(parts[0])
		if cw != nil {
			resp, err := selectWithArgs(parts[1:], cw.sb.String(), fn)
			if err != nil {
				return nil, err
			}
			return &StringReader{resp: resp, delayMs: 0}, nil
		}
	}
	fn, found = PrefixMatch(selectFrom, FILE_PREF)
	if found {
		parts := strings.Split(fn, "|")
		if len(parts) > 1 {
			resp, err := selectFromFileWithArgs(parts[0], parts[1:], selectFrom)
			if err != nil {
				return nil, err
			}
			return &StringReader{resp: resp, delayMs: 0}, nil
		}
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

type Select struct {
	line     int
	contains string
	delim    string
	index    int
	suffix   string
}

func NewSelect(a string, desc string) (*Select, error) {
	var line int = -1
	var contains string = ""
	var delim string = ""
	var ind int = -1
	var suffix string = ""
	var err error = nil

	ap := strings.Split(a, ",")
	if len(ap) > 0 {
		line, err = strconv.Atoi(ap[0])
		if err != nil {
			contains = ap[0]
			line = -1
		}
	}
	if len(ap) > 1 {
		delim = ap[1]
	}
	if len(ap) > 2 {
		ind, err = strconv.Atoi(ap[2])
		if err != nil {
			return nil, fmt.Errorf("string to int conversion failed for selection '%s' element '%s'", desc, ap[0])
		}
	}
	if len(ap) > 3 {
		suffix = a[len(ap[0])+len(ap[1])+len(ap[2])+3:]
	}
	return &Select{line: line, contains: contains, delim: delim, index: ind, suffix: suffix}, nil
}

func parseSelectArgs(args []string, desc string) ([]*Select, error) {
	sels := make([]*Select, 0)
	for _, a := range args {
		newSels, err := NewSelect(a, desc)
		if err != nil {
			return nil, err
		}
		sels = append(sels, newSels)
	}
	return sels, nil
}

func selectFromFileWithArgs(fileName string, args []string, desc string) (string, error) {
	dat, err := os.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("failed to load file '%s' from file input definition '%s'", fileName, desc)
	}
	selectList, err := parseSelectArgs(args, desc)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(dat)))
	line := 0
	for scanner.Scan() {
		selectLineWithArgs(selectList, line, scanner.Text(), &sb)
	}
	return sb.String(), nil
}

func selectWithArgs(args []string, in string, desc string) (string, error) {
	if len(args) == 0 {
		return in, nil
	}
	selectList, err := parseSelectArgs(args, desc)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(in))
	line := 0
	for scanner.Scan() {
		selectLineWithArgs(selectList, line, scanner.Text(), &sb)
	}
	return sb.String(), nil
}

func selectLineWithArgs(args []*Select, ln int, line string, sb *strings.Builder) {
	for _, s := range args {
		if ln == s.line || (s.line == -1 && s.contains != "" && strings.Contains(line, s.contains)) {
			if s.index < 0 || s.delim == "" {
				sb.WriteString(line)
			} else {
				ls := strings.Split(line, s.delim)
				if s.index >= len(ls) {
					sb.WriteString(line)
				} else {
					sb.WriteString(ls[s.index])
					sb.WriteString(s.suffix)
				}
			}
		}
	}
}

func filter(p []byte, filter string) ([]byte, error) {
	parts := strings.Split(filter, "|")
	selectList, err := parseSelectArgs(parts[1:], "")
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(p)))
	line := 0
	for scanner.Scan() {
		selectLineWithArgs(selectList, line, scanner.Text(), &sb)
	}
	return []byte(sb.String()), nil
}
