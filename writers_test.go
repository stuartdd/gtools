package main

import (
	"io"
	"os"
	"testing"
)

var (
	testStdOutW    = NewBaseWriter("", stdColourPrefix[STD_OUT])
	testStdErrW    = NewBaseWriter("", stdColourPrefix[STD_ERR])
	testDataCacheW = NewDataCache()
)

func TestEncryptWriter(t *testing.T) {
	testDataCacheW.ResetCache()
	fw := NewWriter("memory:test001", "", testStdOutW, testStdErrW, testDataCacheW)
	writeStuff(t, fw, "zzz", 3)
	castWriter(t, fw, "zzz")
	_, ok := fw.(Encrypted)
	if !ok {
		t.Fatalf("Error: Could mot cast to Encrypted")
	}
}

func TestMemoryWriter(t *testing.T) {
	testDataCacheW.ResetCache()
	fw := NewWriter("memory:test001", "", testStdOutW, testStdErrW, testDataCacheW)
	writeStuff(t, fw, "zzz", 3)
	castWriter(t, fw, "zzz")
	writeStuff(t, fw, "11", 2)
	castWriter(t, fw, "zzz11")
	writeStuff(t, fw, "\ntt", 3)
	castWriter(t, fw, "zzz11\ntt")
	m1 := testDataCacheW.GetCacheWriter("test001")
	if m1 == nil {
		t.Fatalf("Error: Could fine test001 in memory")
	}
	castWriter(t, m1, "zzz11\ntt")
	s, _ := testDataCacheW.Template("[%{abc}], [%{test001}]", nil)
	if s != "[%{abc}], [zzz11\ntt]" {
		t.Fatalf("Error: Substitution failed %s expectd [%%{abc}], [zzz11\ntt]", s)
	}
}

func TestFileWriter(t *testing.T) {
	testDataCacheW.ResetCache()
	fw1 := NewWriter("test001.txt", "", testStdOutW, testStdErrW, testDataCacheW)
	defer delete(t, "test001.txt")
	writeStuff(t, fw1, "zzz", 3)
	readFileExp(t, "test001.txt", "zzz")
	writeStuff(t, fw1, "yyy", 3)
	readFileExp(t, "test001.txt", "zzzyyy")
	closeWriter(t, fw1)
	readFileExp(t, "test001.txt", "zzzyyy")
	fw2 := NewWriter("test001.txt", "", testStdOutW, testStdErrW, testDataCacheW)
	writeStuff(t, fw2, "zzz", 3)
	readFileExp(t, "test001.txt", "zzz")
	writeStuff(t, fw2, "yyy", 3)
	readFileExp(t, "test001.txt", "zzzyyy")
	closeWriter(t, fw2)
	fw3 := NewWriter("append:test001.txt", "", testStdOutW, testStdErrW, testDataCacheW)
	readFileExp(t, "test001.txt", "zzzyyy")
	writeStuff(t, fw3, "xxx", 3)
	readFileExp(t, "test001.txt", "zzzyyyxxx")
	closeWriter(t, fw3)
	readFileExp(t, "test001.txt", "zzzyyyxxx")
}

func delete(t *testing.T, fileName string) {
	err := os.Remove(fileName)
	if err != nil {
		t.Fatalf("Error: Could not delete file %s", fileName)
	}
}

func readFileExp(t *testing.T, fileName string, expected string) {
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatalf("Error: Could not open file %s", fileName)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Error: Could not read file %s", fileName)
	}
	if string(content) != expected {
		t.Fatalf("Error: read %s, expected %s", content, expected)
	}
}

func castWriter(t *testing.T, w io.Writer, expected string) *CacheWriter {
	cw, ok := w.(*CacheWriter)
	if !ok {
		t.Fatalf("Error: Could not cast to CacheWriter")
	}
	if cw.GetContent() != expected {
		t.Fatalf("Error: CacheWriter content '%s' not '%s'", cw.GetContent(), expected)
	}
	return cw
}

func closeWriter(t *testing.T, w io.Writer) {
	cw, ok := w.(io.Closer)
	if !ok {
		t.Fatalf("Error: Could not cast to Closer")
	}
	err := cw.Close()
	if err != nil {
		t.Fatalf("Error: could not close. %s", err.Error())
	}
}

func writeStuff(t *testing.T, w io.Writer, stuff string, expLen int) {
	l, err := w.Write([]byte(stuff))
	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}
	if l != expLen {
		t.Fatalf("Error: written %d, expected %d", l, expLen)
	}

}
