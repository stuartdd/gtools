package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
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

	CONFIG_FILE = "gtool-config.json"
)

var (
	stdColourPrefix    = []string{GREEN, RED}
	mainWindow         fyne.Window
	selectedTabIndex   int = -1
	model              *Model
	actionRunning      bool = false
	actionRunningLabel *widget.Label
	debugLogMain       *LogData
	refreshLock        sync.Mutex
	notifyActionLock   sync.Mutex
)

type ActionButton struct {
	widget.Button
	action *MultipleActionData
	tapped func(action *MultipleActionData)
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
	clearLog := false
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
		if args[i] == "-lc" {
			clearLog = true
		}
	}

	if logFileName == "" {
		debugLogMain = &LogData{logger: nil, queue: nil}
	} else {
		debugLogMain, err = NewLogData(logFileName, "gtool:", clearLog)
		if err != nil {
			exitApp(fmt.Sprintf("Failed to create logfile '%s'. Error:%s", logFileName, err.Error()), 1)
		}
	}

	if configFileName == "" {
		path = homeDir + string(os.PathSeparator) + CONFIG_FILE
		model, err = NewModelFromFile(homeDir, path, debugLogMain, true)
		if err != nil {
			_, isPathErr := err.(*os.PathError)
			if isPathErr {
				model, err = NewModelFromFile(homeDir, CONFIG_FILE, debugLogMain, true)
				if err != nil {
					_, isPathErr = err.(*os.PathError)
					if isPathErr {
						exitApp(fmt.Sprintf("Both\nuser config data '%s' and\nlocal config data '%s' could not be found.\nUse '-c configFileName' for alternative", path, CONFIG_FILE), 1)
					} else {
						exitApp(err.Error(), 1)
					}
				}
			} else {
				exitApp(err.Error(), 1)
			}
		}
	} else {
		model, err = NewModelFromFile(homeDir, configFileName, debugLogMain, true)
	}
	if err != nil {
		exitApp(err.Error(), 1)
	}

	if model.RunAtEnd != "" {
		_, _, err := model.GetActionDataForName(model.RunAtEnd)
		if err != nil {
			exitApp(fmt.Sprintf("RunAtEnd: %s", err.Error()), 1)
		}
		if debugLogMain.IsLogging() {
			debugLogMain.WriteLog(fmt.Sprintf("Run At End \"%s\"", model.RunAtEnd))
		}
	}
	model.Log()
	for _, ras := range model.RunAtStart {
		execDelayedAction(ras.action, ras.delay, nil, model.dataCache)
	}
	gui()
}

func warningAtStart() {
	if model.warning != "" {
		go func() {
			time.Sleep(500 * time.Millisecond)
			WarnDialog("Data Load Error", model.warning, "", mainWindow, 5, debugLogMain)
		}()
	}
}

func gui() {
	a := app.NewWithID("stuartdd.gtest")
	mainWindow = a.NewWindow("Main Window")
	mainWindow.SetCloseIntercept(func() {
		actionClose("", 0)
	})
	refresh()
	mainWindow.SetMaster()
	mainWindow.SetIcon(IconGtool)
	wd, err := os.Getwd()
	if err != nil {
		mainWindow.SetTitle("Config file:" + model.fileName)
	} else {
		mainWindow.SetTitle("Current dir:" + wd)
	}
	mainWindow.Resize(fyne.NewSize(300, 100))
	mainWindow.SetFixedSize(true)
	warningAtStart()
	mainWindow.ShowAndRun()
}

func newActionButton(label string, icon fyne.Resource, tapped func(action *MultipleActionData), action *MultipleActionData) *ActionButton {
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

func refresh() {
	refreshLock.Lock()
	defer refreshLock.Unlock()

	var c fyne.CanvasObject
	bb := buttonBar()
	for _, a := range model.actionList {
		s, _ := substituteValuesIntoString(a.hideExp, nil, model.dataCache)
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

func centerPanelTabbed(actionsByTab map[string][]*MultipleActionData) *container.AppTabs {
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

func centerPanel(actionData []*MultipleActionData) *fyne.Container {
	vp := container.NewVBox()
	vp.Add(widget.NewSeparator())
	min := 3
	for _, l := range actionData {
		if !l.shouldHide {
			hp := container.NewHBox()
			btn := newActionButton(l.name, theme.SettingsIcon(), func(action *MultipleActionData) {
				if !actionRunning {
					go func() {
						execMultipleAction(action, func(state ActionState, name string, optional string, err error) int {
							resp := 0
							switch state {
							case ERROR | WARN:
								resp = WarnDialog(name, err.Error(), optional, mainWindow, 99, debugLogMain)
							case DONE:
								notifyActionRunning(false, name)
								refresh()
							case START:
								notifyActionRunning(true, name)
							case EXIT:
								exitApp(name, 1)
							}
							return resp
						}, model.dataCache)
						refresh()
					}()
				}
			}, l)
			hp.Add(btn)
			lab, err := substituteValuesIntoString(l.desc, nil, model.dataCache)
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

func buttonBar() *fyne.Container {
	bb := container.NewHBox()
	bb.Add(widget.NewButtonWithIcon("Close", theme.LogoutIcon(), func() {
		actionClose("", 0)
	}))
	if model.AltExitTitle != "" {
		bb.Add(widget.NewButtonWithIcon(model.AltExitTitle, theme.LogoutIcon(), func() {
			actionClose(fmt.Sprintf("AltCloseAction: '%s'", model.AltExitTitle), model.AltExitRc)
		}))
	}
	bb.Add(widget.NewButtonWithIcon("Reload", theme.MediaReplayIcon(), func() {
		m, err := NewModelFromFile(model.homePath, model.fileName, debugLogMain, true)
		if err != nil {
			fmt.Printf("Failed to reload")
		} else {
			model = m
			if debugLogMain.IsLogging() {
				debugLogMain.WriteLog("Model Reloaded")
			}
			for _, ras := range model.RunAtStart {
				execDelayedAction(ras.action, ras.delay, func(state ActionState, name string, err error) {
					switch state {
					case DONE | ERROR | WARN:
						notifyActionRunning(false, name)
						refresh()
					case START:
						notifyActionRunning(true, name)
					}
				}, model.dataCache)
			}
		}
	}))
	actionRunningLabel = widget.NewLabel("")
	bb.Add(actionRunningLabel)
	return bb
}

func notifyActionRunning(newState bool, name string) {
	notifyActionLock.Lock()
	defer notifyActionLock.Unlock()
	actionRunning = newState
	if actionRunning {
		actionRunningLabel.SetText(fmt.Sprintf("Running '%s'", name))
	} else {
		actionRunningLabel.SetText("")
	}
}

func actionClose(data string, code int) {
	if model.RunAtEnd != "" {
		action, _, err := model.GetActionDataForName(model.RunAtEnd)
		if err != nil {
			exitApp(err.Error(), 1)
		}
		execMultipleAction(action, nil, model.dataCache)
	}
	mainWindow.Close()
	exitApp(data, code)
}

func exitApp(data string, code int) {
	if debugLogMain.IsLogging() {
		debugLogMain.WriteLog(fmt.Sprintf("Exit: code:%d message:\"%s\"", code, data))
	}
	if code != 0 {
		fmt.Printf("%sEXIT CODE[%d]:%s%s\n", stdColourPrefix[STD_ERR], code, data, RESET)
	} else {
		if data != "" {
			fmt.Printf("%s%s%s\n", stdColourPrefix[STD_OUT], data, RESET)
		}
	}
	debugLogMain.Close()
	os.Exit(code)
}
