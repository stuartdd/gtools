package main

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
)

var (
	stdOut = NewBaseWriter("", stdColourPrefix[STD_OUT])
	stdErr = NewBaseWriter("", stdColourPrefix[STD_ERR])
)

func TestEncryptWriter(t *testing.T) {
	fw := NewWriter("encrypt:test001", stdOut, stdErr)
	write(t, fw, "zzz", 3)
	cast(t, fw, "zzz")
	close(t, fw)
}
func TestMemoryWriter(t *testing.T) {
	fw := NewWriter("memory:test001", stdOut, stdErr)
	write(t, fw, "zzz", 3)
	cast(t, fw, "zzz")
	write(t, fw, "11", 2)
	cast(t, fw, "zzz11")
	write(t, fw, "\ntt", 3)
	cast(t, fw, "zzz11\ntt")
	m1 := ReadFromMemory("test001")
	if m1 == nil {
		t.Fatalf("Error: Could fine test001 in memory")
	}
	cast(t, m1, "zzz11\ntt")
	s := MutateStringFromMemCache("[%{abc}], [%{test001}]")
	if s != "[%{abc}], [zzz11\ntt]" {
		t.Fatalf("Error: Substitution failed %s expectd [%%{abc}], [zzz11\ntt]", s)
	}
	close(t, m1)
}

func TestFileWriter(t *testing.T) {
	fw1 := NewWriter("test001.txt", stdOut, stdErr)
	defer delete(t, "test001.txt")
	write(t, fw1, "zzz", 3)
	read(t, "test001.txt", "zzz")
	write(t, fw1, "yyy", 3)
	read(t, "test001.txt", "zzzyyy")
	close(t, fw1)
	read(t, "test001.txt", "zzzyyy")
	fw2 := NewWriter("test001.txt", stdOut, stdErr)
	write(t, fw2, "zzz", 3)
	read(t, "test001.txt", "zzz")
	write(t, fw2, "yyy", 3)
	read(t, "test001.txt", "zzzyyy")
	close(t, fw2)
	fw3 := NewWriter("append:test001.txt", stdOut, stdErr)
	read(t, "test001.txt", "zzzyyy")
	write(t, fw3, "xxx", 3)
	read(t, "test001.txt", "zzzyyyxxx")
	close(t, fw3)
	read(t, "test001.txt", "zzzyyyxxx")
}

func delete(t *testing.T, fileName string) {
	err := os.Remove(fileName)
	if err != nil {
		t.Fatalf("Error: Could not delete file %s", fileName)
	}
}

func read(t *testing.T, fileName string, expected string) {
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatalf("Error: Could not open file %s", fileName)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatalf("Error: Could not read file %s", fileName)
	}
	if string(content) != expected {
		t.Fatalf("Error: read %s, expected %s", content, expected)
	}
}

func cast(t *testing.T, w io.Writer, expected string) *CacheWriter {
	cw, ok := w.(*CacheWriter)
	if !ok {
		t.Fatalf("Error: Could not cast to CacheWriter")
	}
	if cw.GetContent() != expected {
		t.Fatalf("Error: CacheWriter content '%s' not '%s'", cw.GetContent(), expected)
	}
	return cw
}

func close(t *testing.T, w io.Writer) {
	cw, ok := w.(io.Closer)
	if !ok {
		t.Fatalf("Error: Could not cast to Closer")
	}
	err := cw.Close()
	if err != nil {
		t.Fatalf("Error: could not close. %s", err.Error())
	}
}

func write(t *testing.T, w io.Writer, stuff string, expLen int) {
	l, err := w.Write([]byte(stuff))
	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}
	if l != expLen {
		t.Fatalf("Error: written %d, expected %d", l, expLen)
	}

}
