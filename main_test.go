package main

import (
	"testing"
)

func TestReader(t *testing.T) {
	mr := NewMyReader(0, "0123456")
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
