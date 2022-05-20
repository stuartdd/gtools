package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var ()

type Reset interface {
	Reset()
}

type ClipContent interface {
	shouldClip() bool
	getContent() string
}

type BaseWriter struct {
	prefix string
	filter string
}

type FileWriter struct {
	fileName string
	filter   string
	file     *os.File
	canWrite bool
	stdErr   *BaseWriter
	stdOut   *BaseWriter
}

type CacheWriter struct {
	name     string
	filter   string
	copyClip bool
	sb       strings.Builder
}

func NewBaseWriter(filter string, prefix string) *BaseWriter {
	return &BaseWriter{filter: filter, prefix: prefix}
}

func (mw *BaseWriter) Write(p []byte) (n int, err error) {
	pLen := len(p)
	if mw.filter != "" {
		p, err = Filter(p, mw.filter)
		if err != nil {
			return 0, err
		}
	}
	fmt.Printf("%s%s%s", mw.prefix, string(p), RESET)
	return pLen, nil
}

func NewCacheWriter(name string, copyClip bool) (*CacheWriter, error) {
	cn, cf := splitNameFilter(name)
	if cn == "" {
		return nil, fmt.Errorf("memory (cache) writer must have a name")
	}
	var sb strings.Builder
	cw := &CacheWriter{name: cn, filter: cf, copyClip: copyClip, sb: sb}
	return cw, nil
}

func (cw *CacheWriter) Write(p []byte) (n int, err error) {
	pLen := len(p)
	if cw.filter != "" {
		p, err = Filter(p, cw.filter)
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

func (cw *CacheWriter) getContent() string {
	return cw.sb.String()
}

func (cw *CacheWriter) shouldClip() bool {
	return cw.copyClip
}

func (cw *CacheWriter) Reset() {
	cw.sb.Reset()
}

func PrefixMatch(s string, pref string) (string, string, bool) {
	if len(s) > len(pref) && strings.ToLower(s)[0:len(pref)] == pref {
		return s[len(pref):], pref, true
	}
	return s, "", false
}

func NewWriter(outName string, defaultOut, stdErr *BaseWriter) io.Writer {
	name, filter := splitNameFilter(outName)
	if name == "" {
		if filter == "" {
			return defaultOut
		}
		return NewBaseWriter(filter, defaultOut.prefix)
	}
	var err error
	var fn string

	fn, typ, found := PrefixMatch(name, CLIP_PREF)
	if !found {
		fn, typ, found = PrefixMatch(name, CACHE_PREF)
	}
	if found {
		cw := ReadCache(fn)
		if cw == nil {
			cw, err = NewCacheWriter(fn+"|"+filter, typ == CLIP_PREF)
			if err != nil {
				stdErr.Write([]byte(fmt.Sprintf("Failed to create '%s' writer '%s'. '%s'", typ, fn, err.Error())))
				return defaultOut
			}
			WriteCache(cw)
		}
		return cw
	}

	var f *os.File
	fn, _, found = PrefixMatch(name, FILE_APPEND_PREF)
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
			p, err = Filter(p, fw.filter)
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

func splitNameFilter(name string) (string, string) {
	sn := strings.TrimLeft(name, " ")
	if strings.HasPrefix(sn, "|") {
		return "", sn[1:]
	}
	parts := strings.SplitN(sn, "|", 2)
	switch len(parts) {
	case 1:
		return parts[0], ""
	case 2:
		return parts[0], parts[1]
	default:
		return "", ""
	}
}
