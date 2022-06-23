package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stuartdd2/JsonParser4go/parser"
)

var (
	ace = []byte(`{
		"cmd": "cat"
	}`)
	cmd1 = []byte(`{
		"cmd": ""
	}`)
	cmd2 = []byte(`{
		"sysin": "sas"
	}`)
	cmd3 = []byte(`{
		"cmd": false
	}`)
	extra = []byte(`{
		"cmd": "fred", "extra": true
	}`)
	delayAsStr = []byte(`{
		"cmd": "fred", "delay": "0"
	}`)
	ignoreError = []byte(`{
		"cmd": "fred", "ignoreError": 1
	}`)
)

const (
	VALIDATE      = true
	DONT_VALIDATE = false
)

func TestValidator(t *testing.T) {
	testValidator(t, "ignoreError", ignoreError, SINGLE_ACTION_DEF, DONT_VALIDATE, "'ignoreError' should be of type 'BOOL'")
	testValidator(t, "delayAsStr", delayAsStr, SINGLE_ACTION_DEF, DONT_VALIDATE, "'delay' should be of type 'NUMBER'")
	testValidator(t, "extra", extra, SINGLE_ACTION_DEF, DONT_VALIDATE, "contains invalid node 'extra'")
	testValidator(t, "cmd3", cmd3, SINGLE_ACTION_DEF, DONT_VALIDATE, "'cmd' should be of type 'STRING'")
	testValidator(t, "cmd2", cmd2, SINGLE_ACTION_DEF, DONT_VALIDATE, "Node 'cmd' is missing")
	testValidator(t, "cmd1", cmd1, SINGLE_ACTION_DEF, DONT_VALIDATE, "'cmd' must have a value")
	testValidator(t, "ace", ace, SINGLE_ACTION_DEF, VALIDATE, "")
}

func testValidator(t *testing.T, id string, json []byte, def map[string]NodeDef, validateExp bool, msgExp string) {
	n, err := parser.Parse(json)
	if err != nil {
		t.Fatalf("Failed Parse:%s", err.Error())
	}
	s, v := ValidateNode(SINGLE_ACTION_DEF, n, fmt.Sprintf("Test %s:", id))

	if v != validateExp {
		if validateExp {
			t.Fatalf("Failed '%s'Should have validated returned msg: '%s'", id, s)
		} else {
			t.Fatalf("Failed '%s' Should NOT have validated expected msg: '%s'", id, msgExp)
		}
	}

	if !strings.Contains(s, msgExp) {
		t.Fatalf("Failed '%s' Response was [%s] it should contain [%s]'", id, s, msgExp)
	}
}
