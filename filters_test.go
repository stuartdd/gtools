package main

import (
	"strings"
	"testing"
)

var td1 = "user=stuart\nuser.email=sdd@gmail.x\n"
var td2 = "user,stuart\nuser.email,sdd@gmail.x\n"
var wr *CacheWriter

func TestFilterParserError(t *testing.T) {
	var err error
	_, err = ParseFilter("1,'abc,3,4")
	if err == nil {
		t.Fatalf("Parser should return 'uneven quotes' error")
	}
}

func TestFilterParser(t *testing.T) {
	testParseRes(t, "a,b", "a|b|", 2, "TestFilterParser: 1.0")
	testParseRes(t, "", "|", 1, "TestFilterParser: 1.1")
	testParseRes(t, ",", "||", 2, "TestFilterParser: 1.2")
	testParseRes(t, ",,,a", "|||a|", 4, "TestFilterParser: 1.3")
	testParseRes(t, "1,2,3,4", "1|2|3|4|", 4, "TestFilterParser: 1.4")
	testParseRes(t, "1,'abc',3,4", "1|abc|3|4|", 4, "TestFilterParser: 1.5")
	testParseRes(t, "1,',a,b,c,',3,4", "1|,a,b,c,|3|4|", 4, "TestFilterParser: 1.6")
}

func TestFilterTD2(t *testing.T) {
	wr, _ = NewCacheWriter("name|'user,',',',1,-->|user.email,',',1", false)
	testResult(t, wr, td2, "stuart-->sdd@gmail.x", "TD2:1.0")
	wr, _ = NewCacheWriter("name|'user,',',',1,-->,|user.,=,1,\n", false)
	testResult(t, wr, td2, "too many parts to filter element ''user,',',',1,-->,'", "TD2:1.1")
}

func TestFilterTD1(t *testing.T) {
	wr, _ = NewCacheWriter("name|user=,=,1,-->|user.email,=,1", false)
	testResult(t, wr, td1, "stuart-->sdd@gmail.x", "TD1:1.0")
	wr, _ = NewCacheWriter("name|user=,=,1,-->|user.,=,1,\n", false)
	testResult(t, wr, td1, "stuart-->sdd@gmail.x\n", "TD1:1.1")
	wr, _ = NewCacheWriter("name|user.,=,1,\n|user=,=,1,<--", false)
	testResult(t, wr, td1, "stuart<--sdd@gmail.x\n", "TD1:1.2")
}

func TestFilterLineNo(t *testing.T) {

	wr, _ = NewCacheWriter("name|0", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0", "LineNo:1.0")
	wr, _ = NewCacheWriter("name|1", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "1.abc.1", "LineNo:1.1")
	wr, _ = NewCacheWriter("name|2", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "", "LineNo:1.2")
	wr, _ = NewCacheWriter("name|0|1", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.01.abc.1", "LineNo:1.3")
	wr, _ = NewCacheWriter("name|1|0", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.01.abc.1", "LineNo:1.4")

	wr, _ = NewCacheWriter("name|1,,,','|0,,,?", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0?1.abc.1,", "LineNo:2.1")

	wr, _ = NewCacheWriter("name|1,.,0,$|0,,,?", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0?1$", "LineNo:2.2")
	wr, _ = NewCacheWriter("name|1,.,1,$|0,,,?", false)
	testResult(t, wr, "0.abc.0\n1.xyz.1\n", "0.abc.0?xyz$", "LineNo:2.3")
	wr, _ = NewCacheWriter("name|1,.,2,$|0,,,?", false)
	testResult(t, wr, "0.abc.0\n1.xyz.1\n", "0.abc.0?1$", "LineNo:2.4")
	wr, _ = NewCacheWriter("name|1,.,3,$|0,,,?", false)
	testResult(t, wr, "0.abc.0\n1.xyz.1\n", "0.abc.0?", "LineNo:2.4")

}

func TestFilterStringPref(t *testing.T) {
	wr, _ = NewCacheWriter("name|abc,,,','", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0,1.abc.1,", "StringPref:1.0")

	wr, _ = NewCacheWriter("name|abc,,,-", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0-1.abc.1-", "StringPref:1.1")

	wr, _ = NewCacheWriter("name|", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0\n1.abc.1\n", "StringPref:2.0")

	wr, _ = NewCacheWriter("name|abc", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.01.abc.1", "StringPref:3.1")
	testResult(t, wr, "0.abc.0\n2.123.2\n", "0.abc.0", "StringPref:3.2")
	testResult(t, wr, "1.aaa.1\n2.123.2\n", "", "3.3")
	testResult(t, wr, "1.aaa.1\n1.abc.1\n", "1.abc.1", "StringPref:3.4")
	testResult(t, wr, "1.aaa.11.abc.1\n", "1.aaa.11.abc.1", "StringPref:3.5")

	wr, _ = NewCacheWriter("name|0.abc,,,[|1.abc,,,]", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0[1.abc.1]", "StringPref:4.1")

	wr, _ = NewCacheWriter("name|0.abc,.,,", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "0.abc.0", "StringPref:5.1")

	wr, _ = NewCacheWriter("name|abc,.,0,", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "01", "StringPref:5.2")

	wr, _ = NewCacheWriter("name|abc,.,1,", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "abcabc", "StringPref:5.3")

	wr, _ = NewCacheWriter("name|abc,.,2,", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "01", "StringPref:5.4")

	wr, _ = NewCacheWriter("name|abc,.,3,", false)
	testResult(t, wr, "0.abc.0\n1.abc.1\n", "", "StringPref:5.5")
}

func testResult(t *testing.T, wr *CacheWriter, input, exp, info string) {
	wr.Reset()
	_, err := wr.Write([]byte(input))
	if err != nil {
		act := err.Error()
		if act != exp {
			t.Errorf("[%s]: Actual Error :'%s' != Expected Error:'%s'", info, act, exp)
		}
		return
	}
	act := wr.sb.String()
	if act != exp {
		t.Errorf("[%s]: Actual:'%s' != Expected:'%s'", info, act, exp)
	}
}

func testParseRes(t *testing.T, fil, exp string, expLen int, info string) {
	l, err := ParseFilter(fil)
	if err != nil {
		t.Fatalf("Parser returned error[%s]", err.Error())
	}
	if len(l) != expLen {
		t.Fatalf("[%s]: Actual Len:'%d' != Expected Len:'%d'", info, len(l), expLen)
	}
	var sb strings.Builder
	for _, v := range l {
		sb.WriteString(v)
		sb.WriteString("|")
	}

	act := sb.String()
	if act != exp {
		t.Fatalf("[%s]: Actual:[]%s] != Expected:[%s]", info, act, exp)
	}
}
