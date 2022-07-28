package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	HTTP_TYPE
)

var (
	actionsPrefName          = parser.NewDotPath("actions")
	altExitPrefName          = parser.NewDotPath("config.altExit")
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

type Model struct {
	debugLog        *LogData       // Log file for events in gtool
	homePath        string         // Users home directory!
	fileName        string         // Root config file name
	jsonRoot        parser.NodeC   // Root Json objects
	actionList      []*ActionData  // List of actions
	dataCache       *DataCache     // List of values
	AltExitTitle    string         // Show additional butten to exit with RC 1
	AltExitRc       int            // Show additional butten to exit with RC 1
	RunAtStart      map[string]int // Action to run on load
	RunAtStartDelay int            // Run at start waits this number of milliseconds
	RunAtEnd        string         // Action to run on exit
	warning         string         // If the model loads dut with warnings
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

func (ad *ActionData) String() string {
	return fmt.Sprintf("Action tab:\"%s\" name:\"%s\" desc:\"%s\"", ad.tab, ad.name, ad.desc)
}

type SingleAction struct {
	command     string
	args        []string
	directory   string
	sysinDef    string
	inPwName    string
	sysoutDef   string
	syserrDef   string
	outPwName   string
	delay       float64
	ignoreError bool
}

func (sa *SingleAction) String() string {
	return fmt.Sprintf("path:\"%s\" cmd:\"%s\" args:\"%s\"", sa.Dir(), sa.command, sa.args)
}

func (sa *SingleAction) Dir() string {
	if sa.directory == "" {
		return "."
	}
	return sa.directory
}

func NewModelFromFile(home, relFileName string, debugLog *LogData, primaryConfig bool) (*Model, error) {
	absFileName, err := filepath.Abs(relFileName)
	if err != nil {
		return nil, err
	}
	j, err := ioutil.ReadFile(absFileName)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(j)) == "" {
		return nil, fmt.Errorf("file '%s' is empty", absFileName)
	}
	configData, err := parser.Parse(j)
	if err != nil {
		return nil, err
	}

	configNode := configData.GetNodeWithName("config")
	if configNode == nil {
		return nil, fmt.Errorf("primary 'config' node in file %s not found", absFileName)
	}

	mod := &Model{homePath: home, fileName: absFileName, jsonRoot: configData, warning: "", actionList: make([]*ActionData, 0), dataCache: NewDataCache(), debugLog: debugLog}
	if debugLog.IsLogging() {
		debugLog.WriteLog(fmt.Sprintf("Config data loaded %s", mod.fileName))
	}
	s, valid := ValidateNode(CONFIG_DEF, configNode.(parser.NodeC), "Config data")
	if !valid {
		return nil, fmt.Errorf("invalid config node in file %s. %s", mod.fileName, s)
	}

	err = mod.loadInputFields()
	if err != nil {
		return nil, err
	}

	err = mod.loadActions()
	if err != nil {
		return nil, err
	}

	ae, err := mod.getContainerNode(altExitPrefName)
	if err != nil {
		return nil, err
	}
	if ae != nil {
		s, valid := ValidateNode(ALT_EXIT, ae, altExitPrefName.String())
		if !valid {
			return nil, fmt.Errorf("invalid %s node in file %s. %s", altExitPrefName, mod.fileName, s)
		}
		tn := ae.GetNodeWithName("title")
		mod.AltExitTitle = tn.(*parser.JsonString).GetValue()
		rn := ae.GetNodeWithName("rc")
		mod.AltExitRc = int(rn.(*parser.JsonNumber).GetIntValue())
	} else {
		mod.AltExitTitle = ""
		mod.AltExitRc = 0
	}
	//
	// Keep a map of the runAtStart/runAtStartDelay parameters from each model
	//
	mod.RunAtStart = make(map[string]int)
	ras := mod.getStringWithFallback(runAtStartPrefName, "")
	rasD := mod.getIntWithFallback(runAtStartDelayPrefName, 100)
	if ras != "" {
		mod.RunAtStart[ras] = rasD
	}
	//
	mod.RunAtEnd = mod.getStringWithFallback(runAtEndPrefName, "")
	if primaryConfig {
		localConfigFile := mod.getStringWithFallback(localConfigPrefName, "")
		if localConfigFile != "" {
			localConfigFileAbs, _ := filepath.Abs(localConfigFile)
			if localConfigFileAbs != mod.fileName {
				if debugLog.IsLogging() {
					debugLog.WriteLog(fmt.Sprintf("Loading local config \"%s\" from \"%s\"", localConfigFileAbs, mod.fileName))
				}
				localMod, err := NewModelFromFile(home, localConfigFileAbs, debugLog, false)
				if err == nil {
					mod.MergeModel(localMod)
				} else {
					_, ok := err.(*os.PathError)
					if ok {
						mod.warning = err.Error()
					} else {
						return nil, err
					}

				}
			} else {
				if debugLog.IsLogging() {
					debugLog.WriteLog(fmt.Sprintf("Duplicate config file not loaded \"%s\"", localConfigFileAbs))
				}
			}
		}
	}
	return mod, nil
}

func (m *Model) MergeModel(localMod *Model) {
	if m.debugLog.IsLogging() {
		m.debugLog.WriteLog(fmt.Sprintf("Merging model \"%s\"", localMod.fileName))
	}
	//
	// Merge actions
	//
	for _, ac := range localMod.actionList {
		_, index, _ := m.GetActionDataForName(ac.name)
		if index < 0 {
			// Add NEW action
			m.actionList = append(m.actionList, ac)
		} else {
			// Override Existing action
			m.actionList[index] = ac
		}
	}
	//
	// Merge values. Replace values in map with same name
	//
	m.dataCache.MergeLocalValues(localMod.dataCache)
	if localMod.RunAtStartDelay > 0 {
		m.RunAtStartDelay = localMod.RunAtStartDelay
	}
	//
	// Only override if defined in local file
	//
	for n, v := range localMod.RunAtStart {
		m.RunAtStart[n] = v
		if m.debugLog.IsLogging() {
			m.debugLog.WriteLog(fmt.Sprintf("Merging RunAtStart Action:%s Delay:%d", n, v))
		}
	}
	//
	// Only override to switch it ON
	//
	if localMod.AltExitTitle != "" {
		m.AltExitTitle = localMod.AltExitTitle
		m.AltExitRc = localMod.AltExitRc
	}

}

func (m *Model) GetActionDataForName(name string) (*ActionData, int, error) {
	for i, a := range m.actionList {
		if a.name == name {
			return a, i, nil
		}
	}
	return nil, -1, fmt.Errorf("action with name '%s' could not be found", name)
}

func (m *Model) GetLocalValue(name string) (*LocalValue, bool) {
	return m.dataCache.GetLocalValue(name)
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
		s, valid := ValidateNode(VALUE_DEF, v.(parser.NodeC), name)
		if !valid {
			return fmt.Errorf("element '%s'. In the config file '%s'. %s", cacheInputFieldsPrefName, m.fileName, s)
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
		lv := m.dataCache.AddLocalValue(name, desc, defaultVal, minLen, isPassword, isFileName, isFileWatch, inputRequired)
		if m.debugLog.IsLogging() {
			m.debugLog.WriteLog(fmt.Sprintf("LocalValue loaded name:%s, desc:\"%s\"", lv.name, lv.desc))
		}
	}
	return nil
}

func (m *Model) Log() {
	if m.debugLog.IsLogging() {
		m.debugLog.WriteLog("***** Final State of the Model:")
		m.dataCache.LogLocalValues(m.debugLog)
		for _, ad := range m.actionList {
			m.debugLog.WriteLog(ad.String())
			for _, sa := range ad.commands {
				m.debugLog.WriteLog("       " + sa.String())
			}
		}
	}
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
		s, valid := ValidateNode(ACTION_DEF, actionNode.(parser.NodeC), "")
		if !valid {
			return fmt.Errorf("invalid data for action '%s'.%s", name, s)
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
			s, valid := ValidateNode(SINGLE_ACTION_DEF, cmdNode.(parser.NodeC), "Command data")
			if !valid {
				return fmt.Errorf("invalid data for action '%s' list[%d]. %s", name, i, s)
			}
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
			path, err := getStringOptNode(cmdNode.(parser.NodeC), "path", "", msg)
			if err != nil {
				return err
			}
			sysinDef, err := getStringOptNode(cmdNode.(parser.NodeC), "stdin", "", msg)
			if err != nil {
				return err
			}
			sysoutDef, err := getStringOptNode(cmdNode.(parser.NodeC), "stdout", "", msg)
			if err != nil {
				return err
			}
			syserrDef, err := getStringOptNode(cmdNode.(parser.NodeC), "syserr", "", msg)
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
				_, found := m.GetLocalValue(outPwName)
				if !found {
					return fmt.Errorf("for '%s'. 'outPwName=%s' was not found in config.cachedFields", msg, outPwName)
				}
				if invalidOutFileNameForPw(sysoutDef) {
					return fmt.Errorf("for '%s'. using 'outPwName=%s' without 'outFile' defined as a file", msg, outPwName)
				}
			}
			inPwName, err := getStringOptNode(cmdNode.(parser.NodeC), "inPwName", "", msg)
			if err != nil {
				return err
			}
			if inPwName != "" {
				if sysinDef == "" {
					return fmt.Errorf("for '%s'.' using 'inPwName=%s' without 'sysin' defined", msg, inPwName)
				}
			}
			ignoreError, err := getBoolOptNode(cmdNode.(parser.NodeC), "ignoreError", false, msg)
			if err != nil {
				return err
			}
			actionData.AddSingleAction(cmd, data, path, sysinDef, outPwName, inPwName, sysoutDef, syserrDef, delay, ignoreError)
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

func (m *Model) getContainerNode(p *parser.Path) (parser.NodeC, error) {
	n, err := parser.Find(m.jsonRoot, p)
	if err != nil || n == nil {
		return nil, nil
	}
	nc, ok := n.(parser.NodeC)
	if ok {
		return nc, nil
	}
	return nil, fmt.Errorf("node %s is not a container node", p.String())
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

func (dc *Model) GetDataCache() *DataCache {
	return dc.dataCache
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
	if a == nil {
		return make([]string, 0), nil
	}
	if a.GetNodeType() != parser.NT_LIST {
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

func NewSingleAction(cmd string, args []string, directory, sysinDef, outPwName, inPwName, sysoutDef, syserrDef string, delay float64, ignoreError bool) *SingleAction {
	return &SingleAction{command: cmd, args: args, directory: directory, outPwName: outPwName, inPwName: inPwName, sysinDef: sysinDef, sysoutDef: sysoutDef, syserrDef: syserrDef, delay: delay, ignoreError: ignoreError}
}

func (p *ActionData) AddSingleAction(cmd string, args []string, directory, sysinDef, outPwName, inPwName, sysoutDef, syserrDef string, delay float64, ignoreError bool) {
	sa := NewSingleAction(cmd, args, directory, sysinDef, outPwName, inPwName, sysoutDef, syserrDef, delay, ignoreError)
	p.commands = append(p.commands, sa)
}

func (p *ActionData) len() int {
	return len(p.commands)
}
