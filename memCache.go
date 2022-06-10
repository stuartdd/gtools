package main

import (
	"fmt"
	"strings"
)

var memoryMap = make(map[string]*CacheWriter, 10)

func WriteToMemory(cw *CacheWriter) {
	memoryMap[cw.name] = cw
}

func ReadFromMemory(name string) *CacheWriter {
	c, ok := memoryMap[name]
	if ok {
		return c
	}
	return nil
}

func MutateStringFromMemCache(in string) string {
	out := in
	for n, v := range memoryMap {
		out = strings.ReplaceAll(out, fmt.Sprintf("%%{%s}", n), strings.TrimSpace(v.GetContent()))
	}
	for n, v := range envMap {
		out = strings.ReplaceAll(out, fmt.Sprintf("%%{%s}", n), strings.TrimSpace(v))
	}
	return out
}
