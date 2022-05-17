package main

import (
	"fmt"
	"strings"
)

var outputCache = make(map[string]*CacheWriter, 10)

func WriteCache(cw *CacheWriter) {
	outputCache[cw.name] = cw
}

func ReadCache(name string) *CacheWriter {
	c, ok := outputCache[name]
	if ok {
		return c
	}
	return nil
}

func MutateStringFromMemCache(in string) string {
	out := in
	for n, v := range outputCache {
		out = strings.Replace(out, fmt.Sprintf("%%{%s}", n), strings.TrimSpace(v.sb.String()), -1)
	}
	return out
}

func MutateListFromMemCache(in []string) []string {
	out := make([]string, 0)
	for _, a := range in {
		out = append(out, MutateStringFromMemCache(a))
	}
	return out
}
