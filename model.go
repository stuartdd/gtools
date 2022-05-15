package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/stuartdd2/JsonParser4go/parser"
)

var (
	actionsPrefName = parser.NewDotPath("actions")
)

type Model struct {
	fileName   string
	root       parser.NodeC
	actionList []*ActionData
}

type ActionData struct {
	name     string
	desc     string
	commands []*SingleAction
}

type SingleAction struct {
	command    string
	args       []string
	sysin      string
	sysoutFile string
	outFilter  string
	syserrFile string
	err        error
	delay      float64
}

func NewModelFromFile(fileName string) (*Model, error) {
	j, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(j)) == "" {
		return nil, fmt.Errorf("file '%s' is empty", fileName)
	}
	configData, err := parser.Parse(j)
	if err != nil {
		return nil, err
	}
	mod := &Model{fileName: fileName, root: configData, actionList: make([]*ActionData, 0)}
	err = mod.loadActions()
	if err != nil {
		return nil, err
	}
	return mod, nil
}

func (m *Model) loadActions() error {
	actionListNode, err := parser.Find(m.root, actionsPrefName)
	if err != nil {
		return err
	}
	if !actionListNode.IsContainer() {
		return fmt.Errorf("node at '%s' is not a container node", actionsPrefName)
	}
	var actionList []parser.NodeI

	if actionListNode.GetNodeType() == parser.NT_LIST {
		actionList = actionListNode.(*parser.JsonList).GetValues()
	} else {
		keys := actionListNode.(*parser.JsonObject).GetSortedKeys()
		actionList = make([]parser.NodeI, 0)
		for _, n := range keys {
			actionList = append(actionList, actionListNode.(*parser.JsonObject).GetNodeWithName(n))
		}
	}

	for ind, actionNode := range actionList {
		var msg string
		if actionNode.GetName() == "" {
			msg = fmt.Sprintf("%s[%d]", actionsPrefName, ind)
		} else {
			msg = fmt.Sprintf("%s{%s}", actionsPrefName, actionNode.GetName())
		}
		if !actionNode.IsContainer() {
			return fmt.Errorf("%s is not a container node", msg)
		}
		name, err := getStringNode(actionNode.(parser.NodeC), "name", msg)
		if err != nil {
			return err
		}
		desc, err := getStringNode(actionNode.(parser.NodeC), "desc", msg)
		if err != nil {
			return err
		}
		actionData := m.getActionData(name, desc)
		cmdList, err := getListNode(actionNode.(parser.NodeC), "list")
		if err != nil {
			return fmt.Errorf("node at %s does not have a list[] node", msg)
		}

		for i, cmdNode := range cmdList.GetValues() {
			msg = fmt.Sprintf("%s -> %s[%d]", msg, "list", i)
			if cmdNode.GetNodeType() != parser.NT_OBJECT {
				return fmt.Errorf("node at %s is not an object node or has only one sub node", msg)
			}
			cmd, err := getStringNode(cmdNode.(parser.NodeC), "cmd", msg)
			if err != nil {
				return err
			}
			msg = fmt.Sprintf("%s -> cmd[%s]", msg, cmd)
			data, err := getStringList(cmdNode.(parser.NodeC), "args", msg)
			if err != nil {
				return err
			}
			in, err := getStringOptNode(cmdNode.(parser.NodeC), "in", "", msg)
			if err != nil {
				return err
			}
			sysoutFile, err := getStringOptNode(cmdNode.(parser.NodeC), "outFile", "", msg)
			if err != nil {
				return err
			}
			outFilter, err := getStringOptNode(cmdNode.(parser.NodeC), "outFilter", "", msg)
			if err != nil {
				return err
			}
			syserrFile, err := getStringOptNode(cmdNode.(parser.NodeC), "errFile", "", msg)
			if err != nil {
				return err
			}
			delay, err := getNumberOptNode(cmdNode.(parser.NodeC), "delay", msg, 0.0)
			if err != nil {
				return err
			}
			actionData.AddSingleAction(cmd, data, in, sysoutFile, outFilter, syserrFile, delay)
		}
		if actionData.len() == 0 {
			return fmt.Errorf("no commands found in 'list' for action '%s' with name '%s'", msg, actionData.name)
		}
	}
	if m.len() == 0 {
		return fmt.Errorf("node at '%s' did not contain any actions", actionsPrefName)
	}
	return nil
}

func (m *Model) len() int {
	return len(m.actionList)
}

func (p *Model) getActionData(name, desc string) *ActionData {
	for _, a1 := range p.actionList {
		if a1.name == name && a1.desc == desc {
			return a1
		}
	}
	n := NewActionData(name, desc)
	p.actionList = append(p.actionList, n)
	return n
}

func getStringNode(node parser.NodeC, name, msg string) (string, error) {
	a := node.GetNodeWithName(name)
	if a == nil || a.GetNodeType() != parser.NT_STRING {
		return "", fmt.Errorf("action node '%s' does not contain the 'String' node '%s'", msg, name)
	}
	if a.String() == "" {
		return "", fmt.Errorf("action node at %s.%s is an empty string", msg, name)
	}
	return a.String(), nil
}

func getStringOptNode(node parser.NodeC, name, def, msg string) (string, error) {
	a := node.GetNodeWithName(name)
	if a == nil {
		return def, nil
	}
	if a.GetNodeType() != parser.NT_STRING {
		return "", fmt.Errorf("action node '%s' does not contain the optional 'String' node '%s'", msg, name)
	}
	return a.String(), nil
}

func getNumberOptNode(node parser.NodeC, name, msg string, def float64) (float64, error) {
	a := node.GetNodeWithName(name)
	if a == nil {
		return def, nil
	}
	if a.GetNodeType() != parser.NT_NUMBER {
		return 0, fmt.Errorf("action node '%s' does not contain the optional 'Number' node '%s'", msg, name)
	}
	return a.(*parser.JsonNumber).GetValue(), nil
}

func getListNode(node parser.NodeC, name string) (parser.NodeC, error) {
	a := node.GetNodeWithName(name)
	if a == nil || !a.IsContainer() {
		return nil, fmt.Errorf("action node '%s' does not contain the 'List[]' node '%s'", node.GetName(), name)
	}
	return a.(parser.NodeC), nil
}

func getStringList(node parser.NodeC, name, msg string) ([]string, error) {
	a := node.GetNodeWithName(name)
	if a == nil || a.GetNodeType() != parser.NT_LIST {
		return nil, fmt.Errorf("action node '%s' does not contain the 'String list[]' node '%s'", msg, name)
	}
	resp := make([]string, 0)
	for _, n := range a.(*parser.JsonList).GetValues() {
		resp = append(resp, n.String())
	}
	return resp, nil
}

func NewActionData(name string, desc string) *ActionData {
	return &ActionData{name: name, desc: desc, commands: make([]*SingleAction, 0)}
}

func NewSingleAction(cmd string, args []string, input, outFile, outFilter, errFile string, delay float64) *SingleAction {
	return &SingleAction{command: cmd, args: args, sysin: input, sysoutFile: outFile, outFilter: outFilter, syserrFile: errFile, delay: delay}
}

func (p *ActionData) AddSingleAction(cmd string, data []string, input, outFile, outFilter, errFile string, delay float64) {
	sa := NewSingleAction(cmd, data, input, outFile, outFilter, errFile, delay)
	p.commands = append(p.commands, sa)
}

func (p *ActionData) len() int {
	return len(p.commands)
}
