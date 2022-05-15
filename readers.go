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
	outCache        = make(map[string]*CacheWriter)
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
	sb     strings.Builder
}

type MyWriter struct {
	id int
}

func NewMyWriter(id int) *MyWriter {
	return &MyWriter{id: id}
}

func (mw *MyWriter) Write(p []byte) (n int, err error) {
	fmt.Printf("%s%s%s", stdColourPrefix[mw.id], string(p), RESET)
	return len(p), nil
}

func NewCacheWriter(name, filter string) (*CacheWriter, error) {
	if name == "" {
		return nil, fmt.Errorf("memory writer must have a name")
	}
	cw := &CacheWriter{name: name, filter: filter}
	cw.sb.Reset()
	return cw, nil
}

func (cw *CacheWriter) Write(p []byte) (n int, err error) {
	pLen := len(p)
	if cw.filter != "" {
		p = filter(p, cw.filter)
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
		return defaultOut
	}
	var err error
	var fn string

	fn, found := PrefixMatch(fileName, CACHE_PREF)
	if found {
		cw, found := outCache[fn]
		if !found {
			cw, err = NewCacheWriter(fn, filter)
			if err != nil {
				stdErr.Write([]byte(fmt.Sprintf("Failed to create '%s' writer '%s'. '%s'", CACHE_PREF, fn, err.Error())))
				return defaultOut
			}
			outCache[fn] = cw
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
			p = filter(p, fw.filter)
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

func NewStringReader(selectFrom string, defaultIn io.Reader, stdErr *MyWriter) io.Reader {
	if selectFrom == "" {
		return defaultIn
	}

	fn, found := PrefixMatch(selectFrom, CACHE_PREF)
	if found {
		parts := strings.Split(fn, "|")
		cw, found := outCache[parts[0]]
		if found {
			return &StringReader{resp: selectWithArgs(parts[1:], cw.sb.String(), stdErr, fn), delayMs: 0}
		}
	}
	fn, found = PrefixMatch(selectFrom, FILE_PREF)
	if found {
		parts := strings.Split(fn, "|")
		if len(parts) > 1 {
			return &StringReader{resp: selectFromFileWithArgs(parts[0], parts[1:], stdErr, selectFrom), delayMs: 0}
		}
	}

	return &StringReader{resp: selectFrom, delayMs: 0}
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

func MutateListFromMemCache(in []string) []string {
	out := make([]string, 0)
	for _, a := range in {
		out = append(out, MutateStringFromMemCache(a))
	}
	return out
}

func MutateStringFromMemCache(in string) string {
	out := in
	for n, v := range outCache {
		out = strings.Replace(out, fmt.Sprintf("%%{%s}", n), strings.TrimSpace(v.sb.String()), -1)
	}
	return out
}

type Select struct {
	line     int
	contains string
	delim    string
	index    int
	suffix   string
}

func NewSelect(a string, stdErr *MyWriter, desc string) *Select {
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
			stdErr.Write([]byte(fmt.Sprintf("String to int conversion failed for selection '%s' element '%s'\n", desc, ap[0])))
			ind = -1
		}
	}
	if len(ap) > 3 {
		suffix = a[len(ap[0])+len(ap[1])+len(ap[2])+3:]
	}
	return &Select{line: line, contains: contains, delim: delim, index: ind, suffix: suffix}
}

func parseSelectArgs(args []string, stdErr *MyWriter, desc string) []*Select {
	sels := make([]*Select, 0)
	for _, a := range args {
		sels = append(sels, NewSelect(a, stdErr, desc))
	}
	return sels
}

func selectFromFileWithArgs(fileName string, args []string, stdErr *MyWriter, desc string) string {
	dat, err := os.ReadFile(fileName)
	if err != nil {
		stdErr.Write([]byte(fmt.Sprintf("Failed to load file '%s' from file input definition '%s'\n", fileName, desc)))
		return desc
	}
	selectList := parseSelectArgs(args, stdErr, desc)

	var sb strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(dat)))
	line := 0
	for scanner.Scan() {
		selectLineWithArgs(selectList, line, scanner.Text(), &sb)
	}
	return sb.String()
}

func selectWithArgs(args []string, in string, stdErr *MyWriter, desc string) string {
	if len(args) == 0 {
		return in
	}
	selectList := parseSelectArgs(args, stdErr, desc)

	var sb strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(in))
	line := 0
	for scanner.Scan() {
		selectLineWithArgs(selectList, line, scanner.Text(), &sb)
	}
	return sb.String()
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

func filter(p []byte, filter string) []byte {
	var sb strings.Builder
	var out strings.Builder
	for _, b := range p {
		sb.WriteByte(b)
		if b == '\n' {
			if strings.Contains(sb.String(), filter) {
				out.WriteString(sb.String())
			}
			sb.Reset()
		}
	}
	return []byte(out.String())
}
