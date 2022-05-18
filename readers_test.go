package main

import (
	"io"
	"strings"
	"testing"
)

var (
	reader = strings.NewReader("012345678901234")
	err    error
	mr     io.Reader
)

func TestReaderDefaultFilter(t *testing.T) {
	mr, err = NewStringReader("file:test_data/readers_test.data|user.", reader)
	if err != nil {
		t.Fatalf("FAIL 001: Should return nil not: %s", err.Error())
	}
	testRead(t, mr, 10, "0123456789", "Reader 1.0")
	testRead(t, mr, 5, "01234", "Reader 1.1")
	testRead(t, mr, 0, "", "Reader 1.2")
}

func TestReaderDefault(t *testing.T) {
	mr, err = NewStringReader("", reader)
	if mr != reader || err != nil {
		t.Fatalf("FAIL 001: Should return nil not: %s", err.Error())
		return
	}
	testRead(t, mr, 10, "0123456789", "Reader 1.0")
	testRead(t, mr, 5, "01234", "Reader 1.1")
	testRead(t, mr, 0, "", "Reader 1.2")
}

func TestReaderFile15(t *testing.T) {
	mr, err = NewStringReader("file:test_data/readers_test_15.data", reader)
	if err != nil {
		t.Fatalf("FAIL 001: Should return nil not: %s", err.Error())
	}
	testRead(t, mr, 10, "0123456789", "Reader 1.0")
	testRead(t, mr, 5, "01234", "Reader 1.1")
	testRead(t, mr, 0, "", "Reader 1.2")
}

func TestReaderCache15(t *testing.T) {
	createCache(t, "cw15", "012345678901234")

	mr, err = NewStringReader("memory:xxxx", reader)
	if err == nil {
		t.Fatalf("FAIL 001: Must throw an error if cache entry not found")
	}
	mr, err = NewStringReader("memory:cw15", reader)
	if err != nil {
		t.Fatalf("FAIL 001: Must NOT throw an error if cache entry is found :%s", err.Error())
	}
	testRead(t, mr, 10, "0123456789", "Reader 1.0")
	testRead(t, mr, 5, "01234", "Reader 1.1")
	testRead(t, mr, 0, "", "Reader 1.2")
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
	testRead(t, mr, 10, "0123456789", "Reader 2.0")
	testRead(t, mr, 5, "01234", "Reader 2.1")
	testRead(t, mr, 0, "", "Reader 2.2")
}

func testRead(t *testing.T, r io.Reader, expLen int, expStr string, info string) {
	buf := make([]byte, 10)
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

func createCache(t *testing.T, name, content string) {
	cw, err := NewCacheWriter(name, "")
	if err != nil {
		t.Fatalf("FAIL createCache: NewCacheWriter Should return nil not: %s", err.Error())
	}
	l, err := cw.Write([]byte(content))
	if err != nil {
		t.Fatalf("FAIL createCache: Write Should return nil not: %s", err.Error())
	}
	if l != len(content) {
		t.Fatalf("FAIL createCache: Should write %d bytes not %d", len(content), l)
	}
	WriteCache(cw)
}
