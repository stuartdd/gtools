package main

import (
	"fmt"
	"testing"
)

var wr *CacheWriter

func TestFilterLineNo(t *testing.T) {

	wr, _ = NewCacheWriter("fred", "0")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0", "LineNo:1.0")
	wr, _ = NewCacheWriter("fred", "1")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "1.abc.1", "LineNo:1.1")
	wr, _ = NewCacheWriter("fred", "2")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "", "LineNo:1.2")
	wr, _ = NewCacheWriter("fred", "0|1")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.01.abc.1", "LineNo:1.3")
	wr, _ = NewCacheWriter("fred", "1|0")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.01.abc.1", "LineNo:1.4")

	wr, _ = NewCacheWriter("fred", "1,,,,|0,,,?")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0?1.abc.1,", "LineNo:2.1")

	wr, _ = NewCacheWriter("fred", "1,.,0,$|0,,,?")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0?1$", "LineNo:2.2")
	wr, _ = NewCacheWriter("fred", "1,.,1,$|0,,,?")
	testResult(t, wr, "0.abc.0\n1.xyz.1\n", "0.abc.0?xyz$", "LineNo:2.3")
	wr, _ = NewCacheWriter("fred", "1,.,2,$|0,,,?")
	testResult(t, wr, "0.abc.0\n1.xyz.1\n", "0.abc.0?1$", "LineNo:2.4")
	wr, _ = NewCacheWriter("fred", "1,.,3,$|0,,,?")
	testResult(t, wr, "0.abc.0\n1.xyz.1\n", "0.abc.0?", "LineNo:2.4")

}

func TestFilterStringPref(t *testing.T) {
	wr, _ = NewCacheWriter("fred", "abc,,,,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0,1.abc.1,", "StringPref:1.0")

	wr, _ = NewCacheWriter("fred", "abc,,,-")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0-1.abc.1-", "StringPref:1.1")

	wr, _ = NewCacheWriter("fred", "")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0\n1.abc.1\n", "StringPref:2.0")

	wr, _ = NewCacheWriter("fred", "abc")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.01.abc.1", "StringPref:3.1")
	testResult(t, wr, "0.abc.0\n2.123.2\n", "0.abc.0", "StringPref:3.2")
	testResult(t, wr, "1.aaa.1\n2.123.2\n", "", "3.3")
	testResult(t, wr, "1.aaa.1\n1.abc.1\n", "1.abc.1", "StringPref:3.4")
	testResult(t, wr, "1.aaa.11.abc.1\n", "1.aaa.11.abc.1", "StringPref:3.5")

	wr, _ = NewCacheWriter("fred", "0.abc,,,[|1.abc,,,]")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0[1.abc.1]", "StringPref:4.1")

	wr, _ = NewCacheWriter("fred", "0.abc,.,,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0", "StringPref:5.1")

	wr, _ = NewCacheWriter("fred", "abc,.,0,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "01", "StringPref:5.2")

	wr, _ = NewCacheWriter("fred", "abc,.,1,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "abcabc", "StringPref:5.3")

	wr, _ = NewCacheWriter("fred", "abc,.,2,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "01", "StringPref:5.4")

	wr, _ = NewCacheWriter("fred", "abc,.,3,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "", "StringPref:5.5")
}

func testResult(t *testing.T, wr *CacheWriter, input, exp, info string) {
	wr.Reset()
	wr.Write([]byte(input))
	act := wr.sb.String()
	if act != exp {
		t.Errorf("[%s]: Actual:'%s' != Expected:'%s'", info, act, exp)
	}
}
func TestReader(t *testing.T) {
	mr, _ := NewStringReader("0123456", nil)
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
