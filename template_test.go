package main

import (
	"fmt"
	"testing"
)

func TestTemplatePcCbCb(t *testing.T) {
	test(t, "%{a}%{b}", "[a][b]")
	test(t, "a%{}%{}", "a[][]")
	test(t, "ab%{c}", "ab[c]")
	test(t, "0%{}", "0[]")
	test(t, "%{}", "[]")
}

func TestTemplatePcCb(t *testing.T) {
	test(t, "a%{%{b", "a%{%{b")
	test(t, "%{%{b", "%{%{b")
	test(t, "a%{%{", "a%{%{")
	test(t, "ab%{c", "ab%{c")
	test(t, "0%{", "0%{")
	test(t, "%{", "%{")
}
func TestTemplatePc(t *testing.T) {
	test(t, "a%%b", "a%%b")
	test(t, "%%b", "%%b")
	test(t, "a%%", "a%%")
	test(t, "ab%c", "ab%c")
	test(t, "0%", "0%")
	test(t, "%", "%")
}

func TestTemplateSimple(t *testing.T) {
	test(t, "0", "0")
	test(t, "abc", "abc")
	test(t, "", "")
}

func test(t *testing.T, in, expected string) {
	str := TemplateParse(in, func(s1, s2 string) string {
		return fmt.Sprintf("[%s]", s1)
	})
	if str != expected {
		t.Fatalf("Should return '%s' not '%s'", expected, str)
	}
}
