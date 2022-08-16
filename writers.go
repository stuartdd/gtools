package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Reset interface {
	Reset()
}

type Encrypted interface {
	SaveToEncryptedFile(string) error
}

type ClipContent interface {
	ShouldClip() bool
	GetContent() string
}

// Write stdout or stderr to stdout or stderr
type BaseWriter struct {
	prefix string //  prefix is prepended to any output line
	filter string //  filter filters the lines written (see README.md)
}

// Write stdout or stderr to a file
type FileWriter struct {
	fileName string
	filter   string      // filter filters the lines written (see README.md)
	password string      // If the file requires encryption then this is NOT ""
	file     *os.File    // The file handle
	canWrite bool        // flag indicates that io can be written to the file
	stdErr   *BaseWriter // Used to report errors with file management
	stdOut   *BaseWriter // Used if file io failed and cannot be written to
}

// Write stdout or stderr to memory cache
type CacheWriter struct {
	name      string          // Used as the name for the cache
	filter    string          // filter filters the lines written (see README.md)
	cacheType ENUM_MEM_TYPE   // Properties of the cache entry.
	sb        strings.Builder // The text in the cache
}

var _ Encrypted = (*CacheWriter)(nil)
var _ ClipContent = (*CacheWriter)(nil)
var _ Reset = (*CacheWriter)(nil)

type HttpPostWriter struct {
	filter    string        // filter filters the lines written (see README.md)
	cacheType ENUM_MEM_TYPE // Properties of the cache entry.
	url       string
	sb        strings.Builder // The text in the cache
}

// Writer takes stdout (outFile) or stderr (errFile) and writes it to the defined receiver.
//
//	  "outFile": "fileName" 			Will create the file 'fileName' and stream the content in to it
//	  "outFile": "append:fileName"   Will append to the file 'fileName'. It will be created if required
//	  "outFile": "memory:name"   	Will write the output to the memory cache with the name 'name'
//	  "outFile": "clip:name"   		Will write the output to the memory cache with the name 'name'
//										AND copy it to the clipboard
func NewWriter(outDef, key string, defaultOut, stdErr *BaseWriter, dataCache *DataCache) io.Writer {
	name, filter := splitNameFilter(outDef)
	if name == "" {
		if filter == "" {
			return defaultOut
		}
		return NewBaseWriter(filter, defaultOut.prefix)
	}
	var err error
	var fn string
	fn, typ, found := PrefixMatch(name, HTTP_PREF, HTTP_TYPE)
	if found {
		return &HttpPostWriter{url: fn, filter: filter, cacheType: typ}
	}

	fn, typ, found = PrefixMatch(name, CLIP_BOARD_PREF, CLIP_TYPE)
	if !found {
		fn, typ, found = PrefixMatch(name, MEMORY_PREF, MEM_TYPE)
	}
	if found {
		cw := dataCache.GetCacheWriter(fn)
		if cw == nil {
			cw, err = NewCacheWriter(fn+"|"+filter, typ)
			if err != nil {
				stdErr.Write([]byte(fmt.Sprintf("Failed to create '%s' writer '%s'. '%s'", CLIP_BOARD_PREF, fn, err.Error())))
				return defaultOut
			}
			dataCache.PutCacheWriter(cw)
		}
		return cw
	}
	if key != "" {
		cw, err := NewCacheWriter(fn+"|"+filter, typ)
		if err != nil {
			stdErr.Write([]byte(fmt.Sprintf("Failed to create '%s' writer '%s'. '%s'", CLIP_BOARD_PREF, fn, err.Error())))
			return defaultOut
		}
		return cw
	}
	var f *os.File
	fn, _, found = PrefixMatch(name, FILE_APPEND_PREF, FILE_TYPE)
	if found {
		f, err = os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	} else {
		f, err = os.Create(fn)
	}
	if err != nil {
		stdErr.Write([]byte(fmt.Sprintf("Failed to create file writer %s. %s", fn, err.Error())))
		return defaultOut
	}
	return &FileWriter{fileName: fn, password: key, file: f, filter: filter, canWrite: true, stdOut: defaultOut, stdErr: stdErr}
}

func (hpw *HttpPostWriter) Write(p []byte) (n int, err error) {
	return hpw.sb.Write(p)
}

func (hpw *HttpPostWriter) Post() error {
	rc, err := HttpPost(hpw.url, "text/plain", hpw.sb.String())
	if err != nil {
		return err
	}
	if rc != 201 {
		return fmt.Errorf("http post failed. Returned '%d'. URL '%s'", rc, hpw.url)
	}
	return nil
}

func NewHttpPostWriter(url, filter string, prefix string) *HttpPostWriter {
	var sb strings.Builder
	return &HttpPostWriter{url: url, filter: filter, sb: sb}
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

func NewCacheWriter(name string, cacheType ENUM_MEM_TYPE) (*CacheWriter, error) {
	cn, cf := splitNameFilter(name)
	if cn == "" {
		return nil, fmt.Errorf("memory (cache) writer must have a name")
	}
	var sb strings.Builder
	cw := &CacheWriter{name: cn, filter: cf, cacheType: cacheType, sb: sb}
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
	if len(p) > 0 {
		np, errp := cw.sb.Write(p)
		if errp != nil {
			return np, err
		}
	}
	return pLen, nil
}

func (cw *CacheWriter) SaveToEncryptedFile(key string) error {
	d, err := EncryptData([]byte(key), []byte(cw.GetContent()))
	if err != nil {
		return err
	}
	f, err := os.Create(cw.name)
	if err != nil {
		return err
	}
	_, err = f.Write(d)
	if err != nil {
		return err
	}
	return nil
}

func (cw *CacheWriter) GetContent() string {
	return cw.sb.String()
}

func (cw *CacheWriter) ShouldClip() bool {
	return cw.cacheType == CLIP_TYPE
}

func (cw *CacheWriter) Reset() {
	cw.sb.Reset()
}

func PrefixMatch(s string, pref string, typ ENUM_MEM_TYPE) (string, ENUM_MEM_TYPE, bool) {
	if len(s) > len(pref) && strings.ToLower(s)[0:len(pref)] == pref {
		return s[len(pref):], typ, true
	}
	return s, typ, false
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
			fw.stdErr.Write([]byte(fmt.Sprintf("Write Error. File:%s.\nErr:%s", fw.fileName, err.Error())))
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
