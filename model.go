package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
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
	HTTP_TYPE
)

var (
	actionsPrefName          = parser.NewDotPath("actions")
	showExit1PrefName        = parser.NewDotPath("config.showExit1")
	runAtStartPrefName       = parser.NewDotPath("config.runAtStart")
	runAtStartDelayPrefName  = parser.NewDotPath("config.runAtStartDelay")
	runAtEndPrefName         = parser.NewDotPath("config.runAtEnd")
	localConfigPrefName      = parser.NewDotPath("config.localConfig")
	cacheInputFieldsPrefName = parser.NewDotPath("config.localValues")

	FILE_APPEND_PREF = "append:" // Used with FileWriter to indicate an append to the file
	CLIP_BOARD_PREF  = "clip:"   // Used with CacheWriter to indicate that the cache is written to the clipboard
	MEMORY_PREF      = "memory:" // Used to indicate that sysout or sysin will be written to cache
	FILE_PREF        = "file:"   // Used with FileReader to indicate a sysin from a file
	HTTP_PREF        = "http:"   // Used with Reader to indicate a sysin from a rest GET instance
	// Used with Writer to send sysout or syserr to a rest POST instance

)

type InputValue struct {
	name          string
	desc          string
	_value        string
	lastValue     string // Use by FileSave and FileOpen as that last location used
	minLen        int
	isPassword    bool
	isFileName    bool
	isFileWatch   bool
	inputDone     bool
	inputRequired bool
}

type Model struct {
	homePath        string                 // Users home directory!
	fileName        string                 // Root config file name
	jsonRoot        parser.NodeC           // Root Json objects
	actionList      []*ActionData          // List of actions
	values          map[string]*InputValue // List of values
	ShowExit1       bool                   // Show additional butten to exit with RC 1
	RunAtStart      string                 // Action to run on load
	RunAtStartDelay int                    // Run at start waits this number of milliseconds
	RunAtEnd        string                 // Action to run on exit
	warning         string                 // If the model loads dut with warnings
}

type ActionData struct {
	tab        string
	name       string
	desc       string
	hideExp    string
	shouldHide bool
	rc         int
	commands   []*SingleAction
}

type SingleAction struct {
	command    string
	args       []string
	sysin      string
	outPwName  string
	inPwName   string
	sysoutFile string
	syserrFile string
	delay      float64
}

func NewModelFromFile(home, fileName string, localConfig bool) (*Model, error) {
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
	mod := &Model{homePath: home, fileName: fileName, jsonRoot: configData, warning: "", actionList: make([]*ActionData, 0), values: make(map[string]*InputValue)}
	err = mod.loadInputFields()
	if err != nil {
		return nil, err
	}
	err = mod.loadActions()
	if err != nil {
		return nil, err
	}
	mod.ShowExit1 = mod.getBoolWithFallback(showExit1PrefName, false)
	mod.RunAtStart = mod.getStringWithFallback(runAtStartPrefName, "")
	mod.RunAtStartDelay = mod.getIntWithFallback(runAtStartDelayPrefName, 0)
	mod.RunAtEnd = mod.getStringWithFallback(runAtEndPrefName, "")
	if localConfig {
		localConfigFile := mod.getStringWithFallback(localConfigPrefName, "")
		if localConfigFile != "" {
			localMod, err := NewModelFromFile(home, localConfigFile, false)
			if err == nil {
				mod.MergeModel(localMod)
			} else {
				mod.warning = err.Error()
			}
		}
	}
	return mod, nil
}

func (m *Model) MergeModel(localMod *Model) {
	//
	// Merge actions
	//
	for i, ac := range localMod.actionList {
		_, err := m.GetActionDataForName(ac.name)
		if err != nil {
			// Add NEW action
			m.actionList = append(m.actionList, ac)
		} else {
			// Override Existing action
			m.actionList[i] = ac
		}
	}
	//
	// Merge values. Replace values in map with same name
	//
	for n, v := range localMod.values {
		m.values[n] = v
	}
	//
	// Only override if defined in local file
	//
	if localMod.RunAtStart != "" {
		m.RunAtStart = localMod.RunAtStart
	}
	if localMod.RunAtStartDelay > 0 {
		m.RunAtStartDelay = localMod.RunAtStartDelay
	}
	//
	// Only override if defined in local file
	//
	if localMod.RunAtEnd != "" {
		m.RunAtEnd = localMod.RunAtEnd
	}
	//
	// Only override to switch it ON
	//
	if localMod.ShowExit1 {
		m.ShowExit1 = localMod.ShowExit1
	}

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
	n, err := parser.Find(m.jsonRoot, cacheInputFieldsPrefName)
	if err != nil || n == nil {
		return nil
	}
	no, ok := n.(*parser.JsonObject)
	if !ok {
		return fmt.Errorf("element '%s'. In the config file '%s'. Is not an Object node", cacheInputFieldsPrefName, m.fileName)
	}
	for _, v := range no.GetValues() {
		name := v.GetName()
		if v.GetNodeType() != parser.NT_OBJECT {
			return fmt.Errorf("element '%s.%s'. In the config file '%s'. Is not an Object node", cacheInputFieldsPrefName, name, m.fileName)
		}
		desc := m.getStringWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("desc"), "")
		if desc == "" {
			return fmt.Errorf("element '%s.%s.desc'. In the config file '%s'. Not found or not a string", cacheInputFieldsPrefName, name, m.fileName)
		}
		defaultVal := m.getStringWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("value"), "")
		inputRequired := m.getBoolWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("input"), false)
		minLen := m.getIntWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("minLen"), 1)
		isPassword := m.getBoolWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("isPassword"), false)
		isFileName := m.getBoolWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("isFileName"), false)
		isFileWatch := m.getBoolWithFallback(cacheInputFieldsPrefName.StringAppend(name).StringAppend("isFileWatch"), false)
		isCount := 0
		if isPassword {
			isCount++
		}
		if isFileName {
			isCount++
		}
		if isFileWatch {
			isCount++
		}
		if isCount > 1 {
			return fmt.Errorf("element '%s.%s.desc'. In the config file '%s'. Con only be 1 of isPassword, isFileName or isFileWatch", cacheInputFieldsPrefName, name, m.fileName)
		}
		lastVal := ""
		v := &InputValue{name: name, desc: desc, _value: defaultVal, minLen: minLen, lastValue: lastVal, isPassword: isPassword, isFileName: isFileName, isFileWatch: isFileWatch, inputDone: false, inputRequired: inputRequired}
		m.values[name] = v
	}
	return nil
}

func (m *Model) loadActions() error {
	actionListNode, err := parser.Find(m.jsonRoot, actionsPrefName)
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
		desc, err := getStringOptNode(actionNode.(parser.NodeC), "desc", "", msg)
		if err != nil {
			return err
		}

		exitCode, err := getNumberOptNode(actionNode.(parser.NodeC), "rc", -1, msg)
		if err != nil {
			return err
		}

		hide, err := getStringOptNode(actionNode.(parser.NodeC), "hide", "", msg)
		if err != nil {
			return err
		}
		actionData := m.getActionData(name, tabName, desc, hide, int(exitCode))
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
		if !a.shouldHide {
			if a.tab == "" {
				singleName = "Main"
			} else {
				singleName = a.tab
			}
			existing, found := resp[singleName]
			if !found {
				existing = make([]*ActionData, 0)
			}
			existing = append(existing, a)
			resp[singleName] = existing
		}
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

func (m *Model) MutateStringFromLocalValues(in string, getValue func(*InputValue) error) (string, error) {
	out := in
	for n, v := range m.values {
		rep := fmt.Sprintf("%%{%s}", n)
		if strings.Contains(in, rep) {
			if getValue != nil {
				if !v.inputDone && v.inputRequired {
					err := getValue(v)
					if err != nil {
						return "", err
					}
				}
			}
			out = strings.Replace(out, rep, strings.TrimSpace(v.GetValue()), -1)
		}
	}
	return out, nil
}

func (m *Model) getStringWithFallback(p *parser.Path, fb string) string {
	n, err := parser.Find(m.jsonRoot, p)
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
	n, err := parser.Find(m.jsonRoot, p)
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
	n, err := parser.Find(m.jsonRoot, p)
	if err != nil || n == nil {
		return fb
	}
	nb, ok := n.(*parser.JsonBool)
	if ok {
		return nb.GetValue()
	}
	return fb
}

func (p *Model) getActionData(name, tabName, desc, hide string, exitCode int) *ActionData {
	for _, a1 := range p.actionList {
		if a1.name == name && a1.desc == desc {
			return a1
		}
	}
	n := NewActionData(name, tabName, desc, hide, exitCode)
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

func NewActionData(name, tabName, desc, hide string, exitCode int) *ActionData {
	return &ActionData{name: name, tab: tabName, desc: desc, rc: exitCode, hideExp: hide, shouldHide: false, commands: make([]*SingleAction, 0)}
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

func (v *InputValue) GetValue() string {
	if v.isFileWatch {
		_, err := os.Open(v._value)
		if err != nil {
			return fmt.Sprintf("%%{%s}", v.name)
		}
	}
	return v._value
}

func (v *InputValue) SetValue(val string) {
	v._value = val
}

func (v *InputValue) GetLastValueAsLocation() (fyne.ListableURI, error) {
	if v.lastValue == "" {
		d, err := os.Getwd()
		if err == nil {
			v.lastValue = d
		}
	}
	u, err := storage.ParseURI("file://" + v.lastValue)
	if err != nil {
		return nil, err
	}
	l, err := storage.ListerForURI(u)
	if err != nil {
		return nil, err
	}
	return l, nil
}
