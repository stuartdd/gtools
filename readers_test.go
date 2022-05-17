package main

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

var (
	reader = strings.NewReader("012345678901234")
	err    error

	mr io.Reader
)

func testRead(t *testing.T, r io.Reader, expLen int, expStr string, info string) {
	buf := make([]byte, 10)
	l, err := r.Read(buf)
	if err != nil {
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

func TestFileReader(t *testing.T) {

	mr, err = NewStringReader("", reader)
	if mr != reader || err != nil {
		t.Errorf("FAIL 001: Should return nil reader and error")
	}
	testRead(t, mr, 10, "0123456789", "Reader 1.0")
	testRead(t, mr, 5, "01234", "Reader 1.1")

	b := make([]byte, 3)
	c, e := mr.Read(b)
	if e != nil || string(b[0:c]) != "012" || c != 3 {
		t.Errorf("FAIL: %d %s\n", c, string(b[0:c]))
	}
	c, e = mr.Read(b)
	if e != nil || string(b[0:c]) != "345" || c != 3 {
		t.Errorf("FAIL: %d %s\n", c, string(b[0:c]))
	}
	c, e = mr.Read(b)
	if e != nil || string(b[0:c]) != "6" || c != 1 {
		t.Errorf("FAIL: %d %s\n", c, b[0:c])
	}
	c, e = mr.Read(b)
	if e == nil || string(b[0:c]) != "" || c != 0 {
		t.Errorf("FAIL: %d %s\n", c, string(b[0:c]))
	}
}

func TestLineReader(t *testing.T) {
	mr, _ := NewStringReader("012\n34\n56", nil)
	b := make([]byte, 10)
	c, e := mr.Read(b)
	s := string(b[0:c])
	fmt.Printf("%s", s)
	if e != nil || s != "012\n" || c != 4 {
		t.Errorf("FAIL: %d '%s'\n", c, s)
	}
	c, e = mr.Read(b)
	s = string(b[0:c])
	fmt.Printf("%s", s)
	if e != nil || s != "34\n" || c != 3 {
		t.Errorf("FAIL: %d '%s'\n", c, s)
	}
	c, e = mr.Read(b)
	s = string(b[0:c])
	fmt.Printf("%s", s)
	if e != nil || s != "56" || c != 2 {
		t.Errorf("FAIL: %d '%s'\n", c, s)
	}

}
