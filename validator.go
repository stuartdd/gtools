package main

import (
	"fmt"
	"strings"

	"github.com/stuartdd2/JsonParser4go/parser"
)

type NodeDef struct {
	nType    parser.NodeType
	optional bool
}

var (
	CONFIG_DEF = map[string]NodeDef{
		"debugFile": {
			parser.NT_STRING, true,
		},
		"localValues": {
			parser.NT_OBJECT, true,
		},
		"showAltExit": {
			parser.NT_BOOL, true,
		},
		"runAtStart": {
			parser.NT_STRING, true,
		},
		"runAtEnd": {
			parser.NT_STRING, true,
		},
		"runAtStartDelay": {
			parser.NT_NUMBER, true,
		},
		"localConfig": {
			parser.NT_STRING, true,
		},
	}

	VALUE_DEF = map[string]NodeDef{
		"desc": {
			parser.NT_STRING, false,
		},
		"value": {
			parser.NT_STRING, true,
		},
		"input": {
			parser.NT_BOOL, true,
		},
		"minLen": {
			parser.NT_NUMBER, true,
		},
		"isPassword": {
			parser.NT_BOOL, true,
		},
		"isFileName": {
			parser.NT_BOOL, true,
		},
		"isFileWatch": {
			parser.NT_BOOL, true,
		},
	}

	ACTION_DEF = map[string]NodeDef{
		"name": {
			parser.NT_STRING, false,
		},
		"tab": {
			parser.NT_STRING, true,
		},
		"desc": {
			parser.NT_STRING, true,
		},
		"rc": {
			parser.NT_NUMBER, true,
		},
		"hide": {
			parser.NT_STRING, true,
		},
		"list": {
			parser.NT_LIST, true,
		},
	}

	SINGLE_ACTION_DEF = map[string]NodeDef{
		"cmd": {
			parser.NT_STRING, false,
		},
		"args": {
			parser.NT_LIST, true,
		},
		"stdin": {
			parser.NT_STRING, true,
		},
		"inPwName": {
			parser.NT_STRING, true,
		},
		"stdout": {
			parser.NT_STRING, true,
		},
		"outPwName": {
			parser.NT_STRING, true,
		},
		"stderr": {
			parser.NT_STRING, true,
		},
		"delay": {
			parser.NT_NUMBER, true,
		},
		"ignoreError": {
			parser.NT_BOOL, true,
		},
	}
)

func ValidateNode(def map[string]NodeDef, node parser.NodeC, desc string) (string, bool) {
	founds := make(map[string]bool)
	for n := range def {
		founds[n] = false
	}
	nn := node.GetName()
	if nn != "" {
		nn = fmt.Sprintf("'%s' ", nn)
	}
	for _, n := range node.GetValues() {
		name := n.GetName()
		if name == "" { // The node must have a name
			return fmt.Sprintf("%s Node %scontains a node that has no name", desc, nn), false
		}
		d, found := def[name] // The node must be in the map!
		if !found {
			return fmt.Sprintf("%s Node %scontains invalid node '%s'", desc, nn, name), false
		}
		founds[name] = true
		if d.nType != n.GetNodeType() {
			return fmt.Sprintf("%s Node '%s' should be of type '%s'", desc, name, parser.GetNodeTypeName(d.nType)), false
		}
		if !d.optional {
			switch d.nType {
			case parser.NT_STRING:
				s := strings.TrimSpace(n.(*parser.JsonString).GetValue())
				if s == "" {
					return fmt.Sprintf("%s String Node '%s' must have a value", desc, name), false
				}
			}
		}
	}
	for n, r := range founds {
		if !r && !def[n].optional {
			return fmt.Sprintf("%s Node '%s' is missing", desc, n), false
		}
	}
	return "", true
}
