package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

type Select struct {
	line     int
	contains string
	delim    string
	index    int
	suffix   string
}

func newSelect(a string, desc string) (*Select, error) {
	var line int = -1
	var contains string = ""
	var delim string = ""
	var ind int = -1
	var suffix string = ""
	var err error = nil

	ap := strings.Split(a, ",")
	if len(ap) > 0 {
		line, err = strconv.Atoi(ap[0])
		if err != nil {
			contains = ap[0]
			line = -1
		}
	}
	if len(ap) > 1 && ap[1] != "" {
		delim = ap[1]
	}
	if len(ap) > 2 && ap[2] != "" {
		ind, err = strconv.Atoi(ap[2])
		if err != nil {
			return nil, fmt.Errorf("string to int conversion failed for selection '%s' element '%s'", desc, ap[0])
		}
	}
	if len(ap) > 3 {
		suffix = a[len(ap[0])+len(ap[1])+len(ap[2])+3:]
	}
	return &Select{line: line, contains: contains, delim: delim, index: ind, suffix: suffix}, nil
}

func parseSelectArgs(args []string, desc string) ([]*Select, error) {
	sels := make([]*Select, 0)
	for _, a := range args {
		newSels, err := newSelect(a, desc)
		if err != nil {
			return nil, err
		}
		sels = append(sels, newSels)
	}
	return sels, nil
}

func selectLineWithArgs(args []*Select, ln int, line string, sb *strings.Builder) {
	for _, s := range args {
		if ln == s.line || (s.line == -1 && s.contains != "" && strings.Contains(line, s.contains)) {
			if s.index < 0 || s.delim == "" {
				sb.WriteString(line)
				sb.WriteString(s.suffix)
			} else {
				ls := strings.Split(line, s.delim)
				if s.index < len(ls) {
					sb.WriteString(ls[s.index])
					sb.WriteString(s.suffix)
				}
			}
		}
	}
}

func Filter(input []byte, filter string) ([]byte, error) {
	if filter == "" {
		return input, nil
	}
	parts := strings.Split(filter, "|")
	selectList, err := parseSelectArgs(parts, "")
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(input)))
	line := 0
	for scanner.Scan() {
		selectLineWithArgs(selectList, line, scanner.Text(), &sb)
		line++
	}
	return []byte(sb.String()), nil
}
