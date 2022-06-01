package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type ENUM_MEM_TYPE int

const (
	CLIP_TYPE ENUM_MEM_TYPE = iota
	MEM_TYPE
	FILE_TYPE
	ENC_TYPE

	FILE_APPEND_PREF = "append:"  // Used with FileWriter to indicate an append to the file
	CLIP_BOARD_PREF  = "clip:"    // Used with CacheWriter to indicate that the cache is written to the clipboard
	MEMORY_PREF      = "memory:"  // Used to indicate that sysout or sysin will be written to cache
	ENCRYPT_PREF     = "encrypt:" // Used to indicate that sysout or sysin will be written to cache. On close()
	// the contents is encrypted and written to the file.
)

type Reset interface {
	Reset()
}

type Encrypted interface {
	ShouldEncrypt() bool
	WriteToEncryptedFile(string) error
}

type ClipContent interface {
	ShouldClip() bool
	GetContent() string
}

//
// Write stdout or stderr to stdout or stderr
//
type BaseWriter struct {
	prefix string //  prefix is prepended to any output line
	filter string //  filter filters the lines written (see README.md)
}

//
// Write stdout or stderr to a file
//
type FileWriter struct {
	fileName string
	filter   string      //  filter filters the lines written (see README.md)
	file     *os.File    // The file handle
	canWrite bool        // flag indicates that io can be written to the file
	stdErr   *BaseWriter // Used to report errors with file management
	stdOut   *BaseWriter // Used if file io failed and cannot be written to
}

//
// Write stdout or stderr to memory cache
//
type CacheWriter struct {
	name      string          // Used as the name for the cache
	filter    string          // filter filters the lines written (see README.md)
	cacheType ENUM_MEM_TYPE   // Properties of the cache entry.
	sb        strings.Builder // The text in the cache
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
	np, errp := cw.sb.Write(p)
	if errp != nil {
		return np, err
	}
	return pLen, nil
}

func (cw *CacheWriter) WriteToEncryptedFile(key string) error {
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

func (cw *CacheWriter) ShouldEncrypt() bool {
	return cw.cacheType == ENC_TYPE
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

//
// Writer takes stdout (outFile) or stderr (errFile) and writes it to the defined receiver.
//
//   "outFile": "fileName" 			Will create the file 'fileName' and stream the content in to it
//   "outFile": "append:fileName"   Will append to the file 'fileName'. It will be created if required
//   "outFile": "memory:name"   	Will write the output to the memory cache with the name 'name'
//   "outFile": "clip:name"   		Will write the output to the memory cache with the name 'name'
//									AND copy it to the clipboard
//
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

	fn, typ, found := PrefixMatch(name, CLIP_BOARD_PREF, CLIP_TYPE)
	if !found {
		fn, typ, found = PrefixMatch(name, MEMORY_PREF, MEM_TYPE)
		if !found {
			fn, typ, found = PrefixMatch(name, ENCRYPT_PREF, ENC_TYPE)
		}
	}
	if found {
		cw := ReadFromMemory(fn)
		if cw == nil {
			cw, err = NewCacheWriter(fn+"|"+filter, typ)
			if err != nil {
				stdErr.Write([]byte(fmt.Sprintf("Failed to create '%s' writer '%s'. '%s'", CLIP_BOARD_PREF, fn, err.Error())))
				return defaultOut
			}
			WriteToMemory(cw)
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
