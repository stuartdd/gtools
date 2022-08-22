package main

import (
	"fmt"
	"strings"
)

func CleanString(s string, max int) string {
	if max < 1 {
		return ""
	}
	var sb strings.Builder
	for _, r := range s {
		if r < 32 {
			sb.WriteString(fmt.Sprintf("[%d]", r))
		} else {
			sb.WriteRune(r)
		}
	}
	if sb.Len() > max {
		return sb.String()[0:max]
	}
	return sb.String()
}

func PadRight(s string, w int) string {
	if len(s) > w {
		return s[:w]
	}
	var sb strings.Builder
	sb.WriteString(s)
	for i := 0; i < (w - len(s)); i++ {
		sb.WriteByte(32)
	}
	return sb.String()
}

func PadLeft(s string, w int) string {
	if len(s) > w {
		return s[:w]
	}
	var sb strings.Builder
	for i := 0; i < (w - len(s)); i++ {
		sb.WriteByte(32)
	}
	sb.WriteString(s)
	return sb.String()
}
