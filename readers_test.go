package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

var (
	reader = strings.NewReader("012345678901234")
	err    error
	mr     io.Reader
)

func TestReaderFileFilter(t *testing.T) {
	mr, err = NewStringReader("file:test_data/readers_test.data|user.name", reader)
	if err != nil {
		t.Fatalf("FAIL 001: Should return nil not: %s", err.Error())
	}
	testRead(t, mr, 18, "user.name=testuser", "Reader 6.0", 100)
	testRead(t, mr, 0, "", "Reader 6.1", 100)

	mr, err = NewStringReader("file:test_data/readers_test.data|user.name,=,1", reader)
	if err != nil {
		t.Fatalf("FAIL 001: Should return nil not: %s", err.Error())
	}
	testRead(t, mr, 8, "testuser", "Reader 6.1", 100)
	testRead(t, mr, 0, "", "Reader 6.2", 100)
}

func TestReaderCacheFilter(t *testing.T) {
	createCache(t, "test_data/readers_test.data", "cw15", "012345678901234")
	mr, err = NewStringReader("memory:cw15|user.name", reader)
	if err != nil {
		t.Fatalf("FAIL 001: Should return nil not: %s", err.Error())
	}
	testRead(t, mr, 18, "user.name=testuser", "Reader 7.0", 100)
	testRead(t, mr, 0, "", "Reader 7.1", 100)

	mr, err = NewStringReader("file:test_data/readers_test.data|user.name,=,1", reader)
	if err != nil {
		t.Fatalf("FAIL 001: Should return nil not: %s", err.Error())
	}
	testRead(t, mr, 8, "testuser", "Reader 7.1", 100)
	testRead(t, mr, 0, "", "Reader 7.2", 100)
}

func TestReaderDirectFilter(t *testing.T) {
	mr, err = NewStringReader("012345678901234|user.name", reader)
	if err != nil {
		t.Fatalf("FAIL 001: Should return nil not: %s", err.Error())
	}
	testRead(t, mr, 15, "012345678901234", "Reader 8.0", 15)
	testRead(t, mr, 1, "|", "Reader 8.0", 1)
	testRead(t, mr, 9, "user.name", "Reader 8.0", 100)
	testRead(t, mr, 0, "", "Reader 6.1", 100)
}

func TestReaderDefault(t *testing.T) {
	mr, err = NewStringReader("", reader)
	if mr != reader || err != nil {
		t.Fatalf("FAIL 001: Should return nil not: %s", err.Error())
		return
	}
	testRead(t, mr, 10, "0123456789", "Reader 2.0", 10)
	testRead(t, mr, 5, "01234", "Reader 2.1", 10)
	testRead(t, mr, 0, "", "Reader 2.2", 10)
}

func TestReaderFile15(t *testing.T) {
	mr, err = NewStringReader("file:test_data/readers_test_15.data", reader)
	if err != nil {
		t.Fatalf("FAIL 001: Should return nil not: %s", err.Error())
	}
	testRead(t, mr, 10, "0123456789", "Reader 3.0", 10)
	testRead(t, mr, 5, "01234", "Reader 3.1", 10)
	testRead(t, mr, 0, "", "Reader 3.2", 10)
}

func TestReaderCache15(t *testing.T) {
	createCache(t, "test_data/readers_test_15.data", "cw15", "012345678901234")

	mr, err = NewStringReader("memory:xxxx", reader)
	if err == nil {
		t.Fatalf("FAIL 001: Must throw an error if cache entry not found")
	}
	mr, err = NewStringReader("memory:cw15", reader)
	if err != nil {
		t.Fatalf("FAIL 001: Must NOT throw an error if cache entry is found :%s", err.Error())
	}
	testRead(t, mr, 10, "0123456789", "Reader 4.0", 10)
	testRead(t, mr, 5, "01234", "Reader 4.1", 10)
	testRead(t, mr, 0, "", "Reader 4.2", 10)
}

func TestReaderDirect(t *testing.T) {
	mr, err = NewStringReader("012345678901234", reader)
	if err != nil {
		t.Fatalf("FAIL 002: Should return nil not: %s", err.Error())
	}
	_, ok := mr.(*StringReader)
	if !ok {
		t.Errorf("FAIL 002: Should return a StringReader")
	}
	testRead(t, mr, 10, "0123456789", "Reader 5.0", 10)
	testRead(t, mr, 5, "01234", "Reader 5.1", 10)
	testRead(t, mr, 0, "", "Reader 5.2", 10)
}

func testRead(t *testing.T, r io.Reader, expLen int, expStr string, info string, bufLen int) {
	buf := make([]byte, bufLen)
	l, err := r.Read(buf)
	if l == 0 && err.Error() != "EOF" {
		t.Errorf("FAIL %s: Should return EOF if ret is 0", info)
	}
	if l != 0 && err != nil {
		t.Errorf("FAIL %s: Should always retur nil err", info)
	}

	if l != expLen {
		t.Errorf("FAIL %s: Expected len %d actual len %d", info, expLen, l)
	}
	b := string(buf[:l])
	if b != expStr {
		t.Errorf("FAIL %s: Expected value '%s' actual val '%s'", info, expStr, b)
	}
}

func createCache(t *testing.T, fileName, name, content string) {
	dat, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("failed to load file '%s' from file input definition", fileName)
	}
	cw, err := NewCacheWriter(name, MEM_TYPE)
	if err != nil {
		t.Fatalf("FAIL createCache: NewCacheWriter Should return nil not: %s", err.Error())
	}
	_, err = cw.Write(dat)
	if err != nil {
		t.Fatalf("FAIL createCache: Write Should return nil not: %s", err.Error())
	}
	WriteToMemory(cw)
}
