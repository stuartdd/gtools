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
	actionList map[string]*ActionData
}

type ActionData struct {
	action   string
	desc     string
	commands []*SingleAction
}

type SingleAction struct {
	command    string
	args       []string
	sysin      string
	sysoutFile string
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
	return &Model{fileName: fileName, root: configData, actionList: make(map[string]*ActionData)}, nil
}

func (m *Model) loadActions() error {
	actionListNode, err := parser.Find(m.root, actionsPrefName)
	if err != nil {
		return err
	}
	if !actionListNode.IsContainer() {
		return fmt.Errorf("node at '%s' is not a container node", actionsPrefName)
	}
	for _, actionNode := range actionListNode.(parser.NodeC).GetValues() {
		if !actionNode.IsContainer() {
			return fmt.Errorf("node at '%s.%s' is not a container node", actionsPrefName, actionNode.GetName())
		}
		actionName := actionNode.GetName()
		actionData, ok := m.actionList[actionName]
		if !ok {
			name, err := getStringNode(actionNode.(parser.NodeC), "name", actionNode.GetName())
			if err != nil {
				return err
			}
			desc, err := getStringNode(actionNode.(parser.NodeC), "desc", actionNode.GetName())
			if err != nil {
				return err
			}
			actionData = NewActionData(name, desc)
			m.actionList[actionName] = actionData
		}
		cmdList, err := getListNode(actionNode.(parser.NodeC), "list")
		if err != nil {
			return err
		}
		for i, cmdNode := range cmdList.GetValues() {
			msg := fmt.Sprintf("%s -> %s[%d]", actionName, cmdList.GetName(), i)
			if cmdNode.GetNodeType() != parser.NT_OBJECT {
				return fmt.Errorf("node at %s is not an object node", msg)
			}
			cmd, err := getStringNode(cmdNode.(parser.NodeC), "cmd", msg)
			if err != nil {
				return err
			}
			data, err := getStringList(cmdNode.(parser.NodeC), "data", msg)
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

			delay, err := getNumberNode(cmdNode.(parser.NodeC), "delay", msg, 0.0)
			if err != nil {
				return err
			}
			actionData.AddSingleAction(cmd, data, in, sysoutFile, delay)
		}
		if actionData.len() == 0 {
			return fmt.Errorf("no commands found for action '%s' with description '%s'", actionData.action, actionData.desc)
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

func getStringNode(node parser.NodeC, name, msg string) (string, error) {
	a := node.GetNodeWithName(name)
	if a == nil || a.GetNodeType() != parser.NT_STRING {
		return "", fmt.Errorf("action node '%s' does not contain the 'String' node '%s'", msg, name)
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

func getNumberNode(node parser.NodeC, name, msg string, def float64) (float64, error) {
	a := node.GetNodeWithName(name)
	if a == nil {
		return def, nil
	}
	if a.GetNodeType() != parser.NT_NUMBER {
		return 0, fmt.Errorf("action node '%s' does not contain a 'Number' node '%s'", msg, name)
	}
	return a.(*parser.JsonNumber).GetValue(), nil
}

func getListNode(node parser.NodeC, name string) (parser.NodeC, error) {
	a := node.GetNodeWithName(name)
	if a == nil || !a.IsContainer() {
		return nil, fmt.Errorf("action node '%s' does not contain the 'List' node '%s'", node.GetName(), name)
	}
	return a.(parser.NodeC), nil
}

func getStringList(node parser.NodeC, name, msg string) ([]string, error) {
	a := node.GetNodeWithName(name)
	if a == nil || a.GetNodeType() != parser.NT_LIST {
		return nil, fmt.Errorf("action node '%s' does not contain the 'String List' node '%s'", msg, name)
	}
	resp := make([]string, 0)
	for _, n := range a.(*parser.JsonList).GetValues() {
		resp = append(resp, n.String())
	}
	return resp, nil
}

func NewActionData(action string, desc string) *ActionData {
	return &ActionData{action: action, desc: desc, commands: make([]*SingleAction, 0)}
}

func NewSingleAction(cmd string, args []string, input, outFile string, delay float64) *SingleAction {
	return &SingleAction{command: cmd, args: args, sysin: input, sysoutFile: outFile, delay: delay}
}

func (p *ActionData) AddSingleAction(cmd string, data []string, input, outFile string, delay float64) {
	sa := NewSingleAction(cmd, data, input, outFile, delay)
	p.commands = append(p.commands, sa)
}

func (p *ActionData) len() int {
	return len(p.commands)
}

func (p *ActionData) Key() string {
	return p.Key()
}
