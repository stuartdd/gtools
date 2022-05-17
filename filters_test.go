package main

import (
	"testing"
)

var td1 = "user=stuart\nuser.email=sdd@gmail.x\n"
var wr *CacheWriter

func TestFilterTD1(t *testing.T) {
	wr, _ = NewCacheWriter("name", "user=,=,1,-->|user.email,=,1")
	testResult(t, wr, td1, "stuart-->sdd@gmail.x", "TD:1.0")
	wr, _ = NewCacheWriter("name", "user=,=,1,-->|user.,=,1,\n")
	testResult(t, wr, td1, "stuart-->sdd@gmail.x\n", "TD:1.1")
	wr, _ = NewCacheWriter("name", "user.,=,1,\n|user=,=,1,<--")
	testResult(t, wr, td1, "stuart<--sdd@gmail.x\n", "TD:1.1")
}

func TestFilterLineNo(t *testing.T) {

	wr, _ = NewCacheWriter("name", "0")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0", "LineNo:1.0")
	wr, _ = NewCacheWriter("name", "1")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "1.abc.1", "LineNo:1.1")
	wr, _ = NewCacheWriter("name", "2")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "", "LineNo:1.2")
	wr, _ = NewCacheWriter("name", "0|1")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.01.abc.1", "LineNo:1.3")
	wr, _ = NewCacheWriter("name", "1|0")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.01.abc.1", "LineNo:1.4")

	wr, _ = NewCacheWriter("name", "1,,,,|0,,,?")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0?1.abc.1,", "LineNo:2.1")

	wr, _ = NewCacheWriter("name", "1,.,0,$|0,,,?")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0?1$", "LineNo:2.2")
	wr, _ = NewCacheWriter("name", "1,.,1,$|0,,,?")
	testResult(t, wr, "0.abc.0\n1.xyz.1\n", "0.abc.0?xyz$", "LineNo:2.3")
	wr, _ = NewCacheWriter("name", "1,.,2,$|0,,,?")
	testResult(t, wr, "0.abc.0\n1.xyz.1\n", "0.abc.0?1$", "LineNo:2.4")
	wr, _ = NewCacheWriter("name", "1,.,3,$|0,,,?")
	testResult(t, wr, "0.abc.0\n1.xyz.1\n", "0.abc.0?", "LineNo:2.4")

}

func TestFilterStringPref(t *testing.T) {
	wr, _ = NewCacheWriter("name", "abc,,,,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0,1.abc.1,", "StringPref:1.0")

	wr, _ = NewCacheWriter("name", "abc,,,-")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0-1.abc.1-", "StringPref:1.1")

	wr, _ = NewCacheWriter("name", "")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0\n1.abc.1\n", "StringPref:2.0")

	wr, _ = NewCacheWriter("name", "abc")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.01.abc.1", "StringPref:3.1")
	testResult(t, wr, "0.abc.0\n2.123.2\n", "0.abc.0", "StringPref:3.2")
	testResult(t, wr, "1.aaa.1\n2.123.2\n", "", "3.3")
	testResult(t, wr, "1.aaa.1\n1.abc.1\n", "1.abc.1", "StringPref:3.4")
	testResult(t, wr, "1.aaa.11.abc.1\n", "1.aaa.11.abc.1", "StringPref:3.5")

	wr, _ = NewCacheWriter("name", "0.abc,,,[|1.abc,,,]")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0[1.abc.1]", "StringPref:4.1")

	wr, _ = NewCacheWriter("name", "0.abc,.,,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0", "StringPref:5.1")

	wr, _ = NewCacheWriter("name", "abc,.,0,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "01", "StringPref:5.2")

	wr, _ = NewCacheWriter("name", "abc,.,1,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "abcabc", "StringPref:5.3")

	wr, _ = NewCacheWriter("name", "abc,.,2,")
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "01", "StringPref:5.4")

	wr, _ = NewCacheWriter("name", "abc,.,3,")
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
