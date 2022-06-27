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

func TestTemplateError(t *testing.T) {
	str, err := TemplateParse("%{name}", func(s string) (string, error) {
		return "", fmt.Errorf("ERROR!")
	})
	if err == nil {
		t.Fatalf("Should return an error")
	}
	if err.Error() != "ERROR!" {
		t.Fatalf("Should return an empty string")
	}
	if str != "%{name}" {
		t.Fatalf("Should return an empty string")
	}

}

func test(t *testing.T, in, expected string) {
	str, _ := TemplateParse(in, func(s1 string) (string, error) {
		return fmt.Sprintf("[%s]", s1), nil
	})
	if str != expected {
		t.Fatalf("Should return '%s' not '%s'", expected, str)
	}
}
