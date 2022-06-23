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
		"debugFile": NodeDef{
			parser.NT_STRING, true,
		},
		"localValues": NodeDef{
			parser.NT_OBJECT, true,
		},
		"showExit1": NodeDef{
			parser.NT_BOOL, true,
		},
		"runAtStart": NodeDef{
			parser.NT_STRING, true,
		},
		"runAtEnd": NodeDef{
			parser.NT_STRING, true,
		},
		"runAtStartDelay": NodeDef{
			parser.NT_NUMBER, true,
		},
		"localConfig": NodeDef{
			parser.NT_STRING, true,
		},
	}

	VALUE_DEF = map[string]NodeDef{
		"desc": NodeDef{
			parser.NT_STRING, false,
		},
		"value": NodeDef{
			parser.NT_STRING, true,
		},
		"input": NodeDef{
			parser.NT_BOOL, true,
		},
		"minLen": NodeDef{
			parser.NT_NUMBER, true,
		},
		"isPassword": NodeDef{
			parser.NT_BOOL, true,
		},
		"isFileName": NodeDef{
			parser.NT_BOOL, true,
		},
		"isFileWatch": NodeDef{
			parser.NT_BOOL, true,
		},
	}

	ACTION_DEF = map[string]NodeDef{
		"name": NodeDef{
			parser.NT_STRING, false,
		},
		"tab": NodeDef{
			parser.NT_STRING, true,
		},
		"desc": NodeDef{
			parser.NT_STRING, true,
		},
		"rc": NodeDef{
			parser.NT_NUMBER, true,
		},
		"hide": NodeDef{
			parser.NT_STRING, true,
		},
		"list": NodeDef{
			parser.NT_LIST, true,
		},
	}

	SINGLE_ACTION_DEF = map[string]NodeDef{
		"cmd": NodeDef{
			parser.NT_STRING, false,
		},
		"args": NodeDef{
			parser.NT_LIST, true,
		},
		"stdin": NodeDef{
			parser.NT_STRING, true,
		},
		"inPwName": NodeDef{
			parser.NT_STRING, true,
		},
		"stdout": NodeDef{
			parser.NT_STRING, true,
		},
		"outPwName": NodeDef{
			parser.NT_STRING, true,
		},
		"stderr": NodeDef{
			parser.NT_STRING, true,
		},
		"delay": NodeDef{
			parser.NT_NUMBER, true,
		},
		"ignoreError": NodeDef{
			parser.NT_BOOL, true,
		},
	}
)

func ValidateNode(def map[string]NodeDef, node parser.NodeC, desc string) (string, bool) {
	founds := make(map[string]bool)
	for n, _ := range def {
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
