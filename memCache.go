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

func MutateStringFromMemCache(in string, getValue func(string, string) (string, error)) (string, error) {
	out := in
	var pwd string
	var sub string
	var err error
	for n, v := range memoryMap {
		if v.cacheType == ENC_TYPE {
			pwd, err = getValue("Encrypted Value", "")
			if err != nil {
				return "", err
			}
			sub = pwd
			out = strings.Replace(out, fmt.Sprintf("%%{%s}", n), strings.TrimSpace(sub), -1)
		} else {
			out = strings.Replace(out, fmt.Sprintf("%%{%s}", n), strings.TrimSpace(v.GetContent()), -1)
		}
	}
	return out, nil
}
