package main

import (
	"fmt"
	"testing"
)

func TestReader(t *testing.T) {
	mr := NewStringReader("0123456", nil, nil)
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
	mr := NewStringReader("012\n34\n56", nil, nil)
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
