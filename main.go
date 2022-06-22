package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	RESET = "\033[0m"
	GREEN = "\033[;32m"
	RED   = "\033[;31m"

	RC_SETUP = -1
	RC_CLEAN = 0
	RC_ERROR = 1

	CONFIG_FILE = "gtool-config.json"
)

var (
	envMap             map[string]string
	stdColourPrefix    = []string{GREEN, RED}
	mainWindow         fyne.Window
	selectedTabIndex   int = -1
	model              *Model
	actionRunning      bool = false
	actionRunningLabel *widget.Label
	debugLog           *LogData
)

type ActionButton struct {
	widget.Button
	action *ActionData
	tapped func(action *ActionData)
}

func main() {
	var err error
	var path string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		exitApp(err.Error(), 1)
	}

	args := os.Args
	aLen := len(args)

	configFileName := ""
	logFileName := ""
	for i := 1; i < aLen; i++ {
		if args[i] == "-c" {
			if (i + 1) >= aLen {
				exitApp("-c config name is undefined", 1)
			}
			configFileName = os.Args[i+1]
			i++
		}
		if args[i] == "-l" {
			if (i + 1) >= aLen {
				exitApp("-l log name is undefined", 1)
			}
			logFileName = os.Args[i+1]
			i++
		}
	}

	if logFileName == "" {
		debugLog = &LogData{logger: nil, queue: nil}
	} else {
		debugLog, err = NewLogData(logFileName, "gtool:")
		if err != nil {
			exitApp(fmt.Sprintf("Failed to create logfile '%s'. Error:%s", logFileName, err.Error()), 1)
		}
	}

	envMap = make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = pair[1]
		}
	}

	if configFileName == "" {
		path = homeDir + string(os.PathSeparator) + CONFIG_FILE
		model, err = NewModelFromFile(homeDir, path, debugLog, true)
		if err != nil {
			_, ok := err.(*os.PathError)
			if ok {
				model, err = NewModelFromFile(homeDir, CONFIG_FILE, debugLog, true)
			} else {
				exitApp(err.Error(), 1)
			}
		}
	} else {
		model, err = NewModelFromFile(homeDir, configFileName, debugLog, true)
	}
	if err != nil {
		exitApp(err.Error(), 1)
	}

	if model.RunAtEnd != "" {
		_, _, err := model.GetActionDataForName(model.RunAtEnd)
		if err != nil {
			exitApp(fmt.Sprintf("RunAtEnd: %s", err.Error()), 1)
		}
		if debugLog.IsLogging() {
			debugLog.WriteLog(fmt.Sprintf("Run At End \"%s\"", model.RunAtEnd))
		}
	}
	if model.RunAtStart != "" {
		runAtStart()
	}

	model.Log()
	gui()
}

func warningAtStart() {
	if model.warning != "" {
		go func() {
			time.Sleep(500 * time.Millisecond)
			WarnDialog("Data Load Error", model.warning, "", mainWindow, 9, debugLog)
		}()
	}
}

func runAtStart() {
	action, _, err := model.GetActionDataForName(model.RunAtStart)
	if err != nil {
		exitApp(fmt.Sprintf("RunAtStart: %s", err.Error()), 1)
	}
	if debugLog.IsLogging() {
		debugLog.WriteLog(fmt.Sprintf("Run At Start \"%s\". Delay %d ms", model.RunAtStart, model.RunAtStartDelay))
	}

	go func() {
		if model.RunAtStartDelay > 0 {
			time.Sleep(time.Duration(model.RunAtStartDelay) * time.Millisecond)
		} else {
			time.Sleep(time.Duration(500 * time.Millisecond))
		}
		execMultipleAction(action)
	}()
}

func gui() {
	a := app.NewWithID("stuartdd.gtest")
	mainWindow = a.NewWindow("Main Window")
	mainWindow.SetCloseIntercept(func() {
		actionClose("", 0)
	})
	update()
	mainWindow.SetMaster()
	mainWindow.SetTitle("Data file:" + model.fileName)
	mainWindow.Resize(fyne.NewSize(300, 100))
	mainWindow.SetFixedSize(true)
	warningAtStart()
	mainWindow.ShowAndRun()
}

func newActionButton(label string, icon fyne.Resource, tapped func(action *ActionData), action *ActionData) *ActionButton {
	ab := &ActionButton{action: action}
	ab.ExtendBaseWidget(ab)
	ab.SetIcon(icon)
	ab.tapped = tapped
	ab.OnTapped = func() {
		ab.tapped(ab.action)
	}
	ab.SetText(label)
	return ab
}

func update() {
	var c fyne.CanvasObject
	bb := buttonBar(action)
	for _, a := range model.actionList {
		s, _ := SubstituteValuesIntoString(a.hideExp, nil)
		a.shouldHide = strings.Contains(s, "%{") || s == "yes"
	}

	tabList, singleName := model.GetTabs()
	if len(tabList) > 1 {
		tabs := centerPanelTabbed(tabList)
		if selectedTabIndex >= 0 {
			tabs.SelectIndex(selectedTabIndex)
		}
		tabs.OnSelected = func(ti *container.TabItem) {
			selectedTabIndex = tabs.SelectedIndex()
		}
		c = container.NewBorder(bb, nil, nil, nil, tabs)
	} else {
		selectedTabIndex = -1
		cp := centerPanel(tabList[singleName])
		c = container.NewBorder(bb, nil, nil, nil, cp)
	}
	mainWindow.SetContent(c)
}

func centerPanelTabbed(actionsByTab map[string][]*ActionData) *container.AppTabs {
	tabs := container.NewAppTabs()
	names := make([]string, 0)
	for name := range actionsByTab {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cp := centerPanel(actionsByTab[name])
		ti := container.NewTabItem(getNameAfterTag(name), cp)
		tabs.Append(ti)
	}
	return tabs
}

func getNameAfterTag(in string) string {
	_, cut, found := strings.Cut(in, ":")
	if found {
		return cut
	}
	return in
}

func centerPanel(actionData []*ActionData) *fyne.Container {
	vp := container.NewVBox()
	vp.Add(widget.NewSeparator())
	min := 3
	for _, l := range actionData {
		if !l.shouldHide {
			hp := container.NewHBox()
			btn := newActionButton(l.name, theme.SettingsIcon(), func(action *ActionData) {
				if !actionRunning {
					go execMultipleAction(action)
				}
			}, l)
			hp.Add(btn)
			lab, err := SubstituteValuesIntoString(l.desc, nil)
			if err != nil {
				lab = l.desc
			}
			hp.Add(widget.NewLabel(lab))
			vp.Add(hp)
			min--
		}
	}
	for min > 0 {
		vp.Add(container.NewHBox(widget.NewLabel("")))
		min--
	}
	return vp
}

func buttonBar(exec func(string, string, string)) *fyne.Container {
	bb := container.NewHBox()
	bb.Add(widget.NewButtonWithIcon("Close(0)", theme.LogoutIcon(), func() {
		exec("exit", "", "")
	}))
	if model.ShowExit1 {
		bb.Add(widget.NewButtonWithIcon("Close(1)", theme.LogoutIcon(), func() {
			exec("exit1", "", "")
		}))
	}
	bb.Add(widget.NewButtonWithIcon("Reload", theme.MediaReplayIcon(), func() {
		m, err := NewModelFromFile(model.homePath, model.fileName, debugLog, true)
		if err != nil {
			fmt.Printf("Failed to reload")
		} else {
			model = m
			if debugLog.IsLogging() {
				debugLog.WriteLog("Model Reloaded")
			}
			go update()
		}
	}))
	actionRunningLabel = widget.NewLabel("")
	bb.Add(actionRunningLabel)
	return bb
}

func setActionRunning(newState bool, name string) {
	actionRunning = newState
	if actionRunning {
		actionRunningLabel.SetText(fmt.Sprintf("Running '%s'", name))
	} else {
		actionRunningLabel.SetText("")
	}
}

func action(exec, data1, data2 string) {
	switch exec {
	case "exit":
		actionClose(data1, 0)
	case "exit1":
		actionClose(data1, 1)
	}
}

func validatedEntryDialog(localValue *InputValue) error {
	return NewMyDialog(localValue, func(s string, iv *InputValue) bool {
		return len(strings.TrimSpace(s)) >= iv.minLen
	}, mainWindow, debugLog).Run(VALUE_DIALOG_TYPE).err
}

func sysInDialog(localValue *InputValue) error {
	return NewMyDialog(localValue, func(s string, iv *InputValue) bool {
		return true
	}, mainWindow, debugLog).Run(SYSIN_DIALOG_TYPE).err
}

func sysOutDialog(localValue *InputValue) error {
	return NewMyDialog(localValue, func(s string, iv *InputValue) bool {
		return true
	}, mainWindow, debugLog).Run(SYSOUT_DIALOG_TYPE).err
}

func deriveKeyFromName(name string, sa *SingleAction) (string, error) {
	if name != "" {
		cf, ok := model.values[name]
		if ok {
			if cf.inputRequired && !cf.inputDone {
				err := validatedEntryDialog(cf)
				if err != nil {
					return "", err
				}
			}
			if cf.GetValue() == "" {
				return "", fmt.Errorf("password not provided")
			}
			return cf.GetValue(), nil
		}
	}
	return "", nil
}

func execMultipleAction(data *ActionData) {
	setActionRunning(true, data.name)
	if debugLog.IsLogging() {
		debugLog.WriteLog("  Started " + data.String())
	}
	defer func() {
		setActionRunning(false, "")
		if debugLog.IsLogging() {
			debugLog.WriteLog("  Ended " + data.String())
		}
	}()

	stdOut := NewBaseWriter("", stdColourPrefix[STD_OUT])
	stdErr := NewBaseWriter("", stdColourPrefix[STD_ERR])
	for i, act := range data.commands {
		locationMsg := fmt.Sprintf("Action '%s' step '%d'", data.desc, i)
		rc, err := execSingleAction(act, stdOut, stdErr, data.desc)
		if err != nil {
			if rc == RC_SETUP {
				if debugLog.IsLogging() {
					debugLog.WriteLog(fmt.Sprintf("    Error Setup: %s. %s ", err.Error(), act.String()))
				}
				WarnDialog(locationMsg, err.Error(), "", mainWindow, 10, debugLog)
				return
			}
			if act.ignoreError {
				if debugLog.IsLogging() {
					debugLog.WriteLog(fmt.Sprintf("    Error Ignored: %s. %s ", err.Error(), act.String()))
				}
			} else {
				if debugLog.IsLogging() {
					debugLog.WriteLog(fmt.Sprintf("    Error: %s. %s ", err.Error(), act.String()))
				}
				exitOsMsg := fmt.Sprintf("Exit to OS with RC=%d", rc)
				resp := WarnDialog(locationMsg, err.Error(), exitOsMsg, mainWindow, 99, debugLog)
				if resp == 1 {
					exitApp(fmt.Sprintf("%s. RC[%d] Error:%s", locationMsg, rc, err.Error()), rc)
				}
				return
			}
		}
		if debugLog.IsLogging() {
			debugLog.WriteLog(fmt.Sprintf("    Command: rc:%d cmd:\"%s %s\"", rc, act.command, act.args))
		}
	}
	if data.rc >= 0 {
		exitApp("", data.rc)
	}
	update()
}

func execSingleAction(sa *SingleAction, stdOut, stdErr *BaseWriter, actionDesc string) (int, error) {
	outEncKey, err := deriveKeyFromName(sa.outPwName, sa)
	if err != nil {
		return RC_SETUP, err
	}
	inEncKey, err := deriveKeyFromName(sa.inPwName, sa)
	if err != nil {
		return RC_SETUP, err
	}
	args, err := SubstituteValuesIntoArgs(sa.args, validatedEntryDialog)
	if err != nil {
		return RC_SETUP, err
	}
	cmd := exec.Command(sa.command, args...)

	if sa.sysin != "" {
		tmp, err := SubstituteValuesIntoString(sa.sysin, sysInDialog)
		if err != nil {
			return RC_SETUP, err
		}
		si, err := NewStringReader(tmp, cmd.Stdin)
		if err != nil {
			return RC_SETUP, err
		}
		siCloser, ok := si.(io.ReadCloser)
		if ok {
			defer siCloser.Close()
		}
		encR, ok := si.(EncReader)
		if ok {
			encR.SetKey(inEncKey)
		}
		cmd.Stdin = si
	}
	tmpOut, err := SubstituteValuesIntoString(sa.sysoutFile, sysOutDialog)
	if err != nil {
		return RC_SETUP, err
	}
	so := NewWriter(tmpOut, outEncKey, stdOut, stdErr)
	soReset, reSoOk := so.(Reset)
	if reSoOk {
		soReset.Reset()
	}
	soCloser, soOk := so.(io.Closer)
	if soOk {
		defer soCloser.Close()
	}
	cmd.Stdout = so

	tmpErr, err := SubstituteValuesIntoString(sa.syserrFile, sysOutDialog)
	if err != nil {
		return RC_SETUP, err
	}
	se := NewWriter(tmpErr, outEncKey, stdErr, stdErr)
	seReset, reSeOk := se.(Reset)
	if reSeOk {
		seReset.Reset()
	}
	seCloser, seOk := se.(io.Closer)
	if seOk {
		defer seCloser.Close()
	}
	cmd.Stderr = se

	err = cmd.Start()
	if err != nil {
		return cmd.ProcessState.ExitCode(), err
	}
	err = cmd.Wait()
	if err != nil {
		return cmd.ProcessState.ExitCode(), err
	}
	if sa.delay > 0.0 {
		time.Sleep(time.Duration(sa.delay) * time.Millisecond)
	}
	cp, ok := so.(ClipContent)
	if ok {
		if cp.ShouldClip() {
			mainWindow.Clipboard().SetContent(cp.GetContent())
		}
	}
	if outEncKey != "" {
		soE, ok := so.(Encrypted)
		if ok {
			soE.WriteToEncryptedFile(outEncKey)
		}
	}
	httpPost, ok := so.(*HttpPostWriter)
	if ok {
		err := httpPost.Post()
		if err != nil {
			return RC_ERROR, err
		}
	}

	return RC_CLEAN, nil
}

func SubstituteValuesIntoArgs(s []string, entryDialog func(*InputValue) error) ([]string, error) {
	resp := make([]string, 0)
	for _, v := range s {
		tmp, err := SubstituteValuesIntoString(v, entryDialog)
		if err != nil {
			return nil, err
		}
		resp = append(resp, tmp)
	}
	return resp, nil
}

func SubstituteValuesIntoString(s string, entryDialog func(*InputValue) error) (string, error) {
	var tmp string
	var err error
	tmp, err = model.MutateStringFromLocalValues(s, entryDialog)
	if err != nil {
		return "", err
	}
	tmp = MutateStringFromMemCache(tmp)
	return tmp, nil
}

func actionClose(data string, code int) {
	if model.RunAtEnd != "" {
		action, _, err := model.GetActionDataForName(model.RunAtEnd)
		if err != nil {
			exitApp(err.Error(), 1)
		}
		execMultipleAction(action)
	}
	mainWindow.Close()
	exitApp(data, code)
}

func exitApp(data string, code int) {
	if debugLog.IsLogging() {
		debugLog.WriteLog(fmt.Sprintf("Exit: code:%d message:\"%s\"", code, data))
	}
	if code != 0 {
		fmt.Printf("%sEXIT CODE[%d]:%s%s\n", stdColourPrefix[STD_ERR], code, data, RESET)
	} else {
		if data != "" {
			fmt.Printf("%s%s%s\n", stdColourPrefix[STD_OUT], data, RESET)
		}
	}
	debugLog.Close()
	os.Exit(code)
}
