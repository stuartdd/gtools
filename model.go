package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/stuartdd2/JsonParser4go/parser"
)

type ENUM_MEM_TYPE int

const (
	STD_OUT = 0
	STD_ERR = 1
	STD_IN  = 2

	CLIP_TYPE ENUM_MEM_TYPE = iota
	MEM_TYPE
	FILE_TYPE
	STR_TYPE
)

var (
	actionsPrefName          = parser.NewDotPath("actions")
	showExit1PrefName        = parser.NewDotPath("config.showExit1")
	runAtStartPrefName       = parser.NewDotPath("config.runAtStart")
	runAtEndPrefName         = parser.NewDotPath("config.runAtEnd")
	cacheInputFieldsPrefName = parser.NewDotPath("config.localValues")
	ShowExit1                = false
	RunAtStart               = ""
	RunAtEnd                 = ""

	FILE_APPEND_PREF = "append:" // Used with FileWriter to indicate an append to the file
	CLIP_BOARD_PREF  = "clip:"   // Used with CacheWriter to indicate that the cache is written to the clipboard
	MEMORY_PREF      = "memory:" // Used to indicate that sysout or sysin will be written to cache
	FILE_PREF        = "file:"   // Used with FileReader to indicate a sysin from a file

)

type InputValue struct {
	name          string
	desc          string
	value         string
	minLen        int
	isPassword    bool
	inputDone     bool
	inputRequired bool
}

type Model struct {
	fileName   string
	root       parser.NodeC
	actionList []*ActionData
	values     map[string]*InputValue
}

type ActionData struct {
	tab      string
	name     string
	desc     string
	hide     bool
	commands []*SingleAction
}

type SingleAction struct {
	command    string
	args       []string
	sysin      string
	outPwName  string
	inPwName   string
	sysoutFile string
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
	mod := &Model{fileName: fileName, root: configData, actionList: make([]*ActionData, 0), values: make(map[string]*InputValue)}
	err = mod.loadInputFields()
	if err != nil {
		return nil, err
	}
	err = mod.loadActions()
	if err != nil {
		return nil, err
	}
	ShowExit1 = mod.getBoolWithFallback(showExit1PrefName, false)
	RunAtStart = mod.getStringWithFallback(runAtStartPrefName, "")
	RunAtEnd = mod.getStringWithFallback(runAtEndPrefName, "")
	return mod, nil
}

func (m *Model) GetActionDataForName(name string) (*ActionData, error) {
	for _, a := range m.actionList {
		if a.name == name {
			return a, nil
		}
	}
	return nil, fmt.Errorf("action with name '%s' could not be found", name)
}

func (m *Model) loadInputFields() error {
	n, err := parser.Find(m.root, cacheInputFieldsPrefName)
	if err != nil || n == nil {
		return nil
	}
	no, ok := n.(*parser.JsonObject)
	if !ok {
		return fmt.Errorf("element '%s' in the config file '%s' is not an Object node", cacheInputFieldsPrefName, m.fileName)
	}
	for _, v := range no.GetValues() {
		name := v.GetName()
		if v.GetNodeType() != parser.NT_OBJECT {
			return fmt.Errorf("element '%s.%s' in the config file '%s' is not an Object node", cacheInputFieldsPrefName, name, m.fileName)
		}
		desc := m.getStringWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("desc"), "")
		if desc == "" {
			return fmt.Errorf("element '%s.%s.desc' in the config file '%s' not found or not a string", cacheInputFieldsPrefName, name, m.fileName)
		}
		defaultVal := m.getStringWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("value"), "")
		inputRequired := m.getBoolWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("input"), false)
		minLen := m.getIntWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("minLen"), 1)
		isPassword := m.getBoolWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("isPassword"), false)
		v := &InputValue{name: name, desc: desc, value: defaultVal, minLen: minLen, isPassword: isPassword, inputDone: false, inputRequired: inputRequired}
		m.values[name] = v
	}
	return nil
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
		tabName, err := getStringOptNode(actionNode.(parser.NodeC), "tab", "", msg)
		if err != nil {
			return err
		}
		desc, err := getStringNode(actionNode.(parser.NodeC), "desc", msg)
		if err != nil {
			return err
		}
		hide, err := getBoolOptNode(actionNode.(parser.NodeC), "hide", false, msg)
		if err != nil {
			return err
		}
		actionData := m.getActionData(name, tabName, desc, hide)
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
			syserrFile, err := getStringOptNode(cmdNode.(parser.NodeC), "errFile", "", msg)
			if err != nil {
				return err
			}
			delay, err := getNumberOptNode(cmdNode.(parser.NodeC), "delay", 0.0, msg)
			if err != nil {
				return err
			}
			outPwName, err := getStringOptNode(cmdNode.(parser.NodeC), "outPwName", "", msg)
			if err != nil {
				return err
			}
			if outPwName != "" {
				_, found := m.values[outPwName]
				if !found {
					return fmt.Errorf("for '%s'. 'outPwName=%s' was not found in config.cachedFields", msg, outPwName)
				}
				if invalidOutFileNameForPw(sysoutFile) {
					return fmt.Errorf("for '%s'. using 'outPwName=%s' without 'outFile' defined as a file", msg, outPwName)
				}
			}
			inPwName, err := getStringOptNode(cmdNode.(parser.NodeC), "inPwName", "", msg)
			if err != nil {
				return err
			}
			if inPwName != "" {
				if in == "" {
					return fmt.Errorf("for '%s'.' using 'inPwName=%s' without 'in' file defined", msg, inPwName)
				}
			}
			actionData.AddSingleAction(cmd, data, in, outPwName, inPwName, sysoutFile, syserrFile, delay)
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

func (m *Model) GetTabs() (map[string][]*ActionData, string) {
	resp := make(map[string][]*ActionData, 0)
	singleName := ""
	for _, a := range m.actionList {
		singleName = a.tab
		existing, found := resp[singleName]
		if !found {
			existing = make([]*ActionData, 0)
		}
		existing = append(existing, a)
		resp[singleName] = existing
	}
	return resp, singleName
}

func invalidOutFileNameForPw(n string) bool {
	return n == "" ||
		strings.HasPrefix(n, FILE_APPEND_PREF) ||
		strings.HasPrefix(n, CLIP_BOARD_PREF) ||
		strings.HasPrefix(n, MEMORY_PREF)
}

func (m *Model) len() int {
	return len(m.actionList)
}

func (m *Model) MutateStringFromValues(in string, getValue func(*InputValue) error) (string, error) {
	out := in
	for n, v := range m.values {
		rep := fmt.Sprintf("%%{%s}", n)
		if strings.Contains(in, rep) {
			if !v.inputDone && v.inputRequired {
				err := getValue(v)
				if err != nil {
					return "", err
				}
			}
			out = strings.Replace(out, rep, strings.TrimSpace(v.value), -1)
		}
	}
	return out, nil
}

func (m *Model) MutateListFromValues(in []string, getValue func(*InputValue) error) ([]string, error) {
	out := make([]string, 0)
	for _, a := range in {
		val, err := m.MutateStringFromValues(a, getValue)
		if err != nil {
			return nil, err
		}
		out = append(out, val)
	}
	return out, nil
}

func (m *Model) getStringWithFallback(p *parser.Path, fb string) string {
	n, err := parser.Find(m.root, p)
	if err != nil || n == nil {
		return fb
	}
	nb, ok := n.(*parser.JsonString)
	if ok {
		if nb.GetValue() == "" {
			return fb
		}
		return nb.GetValue()
	}
	return fb
}

func (m *Model) getIntWithFallback(p *parser.Path, fb int) int {
	n, err := parser.Find(m.root, p)
	if err != nil || n == nil {
		return fb
	}
	nb, ok := n.(*parser.JsonNumber)
	if ok {
		if nb.GetValue() >= 0 {
			return int(nb.GetValue())
		}
	}
	return fb
}

func (m *Model) getBoolWithFallback(p *parser.Path, fb bool) bool {
	n, err := parser.Find(m.root, p)
	if err != nil || n == nil {
		return fb
	}
	nb, ok := n.(*parser.JsonBool)
	if ok {
		return nb.GetValue()
	}
	return fb
}

func (p *Model) getActionData(name, tabName, desc string, hide bool) *ActionData {
	for _, a1 := range p.actionList {
		if a1.name == name && a1.desc == desc {
			return a1
		}
	}
	n := NewActionData(name, tabName, desc, hide)
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
	as, ok := a.(*parser.JsonString)
	if !ok {
		return "", fmt.Errorf("action node '%s' optional '%s' node is not a String node", msg, name)
	}
	return as.String(), nil
}

func getBoolOptNode(node parser.NodeC, name string, def bool, msg string) (bool, error) {
	a := node.GetNodeWithName(name)
	if a == nil {
		return def, nil
	}
	ab, ok := a.(*parser.JsonBool)
	if !ok {
		return def, fmt.Errorf("in action node '%s' optional '%s' node is not a Boolean node", msg, name)
	}
	return ab.GetValue(), nil
}

func getNumberOptNode(node parser.NodeC, name string, def float64, msg string) (float64, error) {
	a := node.GetNodeWithName(name)
	if a == nil {
		return def, nil
	}
	an, ok := a.(*parser.JsonNumber)
	if !ok {
		return def, fmt.Errorf("in action node '%s' optional '%s' node is not a Number node", msg, name)
	}
	return an.GetValue(), nil
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

func NewActionData(name, tabName, desc string, hide bool) *ActionData {
	return &ActionData{name: name, tab: tabName, desc: desc, hide: hide, commands: make([]*SingleAction, 0)}
}

func NewSingleAction(cmd string, args []string, input, outPwName, inPwName, outFile, errFile string, delay float64) *SingleAction {
	return &SingleAction{command: cmd, args: args, outPwName: outPwName, inPwName: inPwName, sysin: input, sysoutFile: outFile, syserrFile: errFile, delay: delay}
}

func (p *ActionData) AddSingleAction(cmd string, data []string, input, outPwName, inPwName, outFile, errFile string, delay float64) {
	sa := NewSingleAction(cmd, data, input, outPwName, inPwName, outFile, errFile, delay)
	p.commands = append(p.commands, sa)
}

func (p *ActionData) len() int {
	return len(p.commands)
}
