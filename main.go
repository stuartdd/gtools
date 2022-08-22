package main

import (
	"fmt"
	"math"
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

type ViewState int

const (
	RESET = "\033[0m"
	GREEN = "\033[;32m"
	RED   = "\033[;31m"

	CONFIG_FILE = "gtool-config.json"

	VIEW_ACTIONS ViewState = iota
	VIEW_DATA
)

var (
	stdColourPrefix       = []string{GREEN, RED}
	mainWindow            fyne.Window
	mainWindowActive      bool = false
	selectedTabIndex      int  = -1
	selectedValueTabIndex int  = -1

	currentView        ViewState = VIEW_ACTIONS
	model              *Model
	actionRunning      bool = false
	actionRunningLabel *widget.Label
	debugLogMain       *LogData
	refreshLock        sync.Mutex
	notifyChannel      chan *NotifyMessage
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
		NewNotifyMessage(ERROR, nil, "Load Error", "", 1, err)
	}

	configFileName, err := GetArg("-c")
	if err != nil {
		exitApp(NewNotifyMessage(ERROR, nil, "", "", 1, err))
	}
	logFileName, err := GetArg("-l")
	if err != nil {
		exitApp(NewNotifyMessage(ERROR, nil, "", "", 1, err))
	}
	clearLog := HasArg("-lc")

	if logFileName == "" {
		debugLogMain = &LogData{logger: nil, queue: nil, maxLineLen: 0}
	} else {
		debugLogMain, err = NewLogData(logFileName, "gtool:", 100, clearLog)
		if err != nil {
			exitApp(NewNotifyMessage(ERROR, nil, fmt.Sprintf("Failed to create logfile '%s'", logFileName), "", 1, err))
		}
	}
	notifyChannel = make(chan *NotifyMessage, 1)

	if configFileName == "" {
		path = homeDir + string(os.PathSeparator) + CONFIG_FILE
		model, err = NewModelFromFile(homeDir, path, debugLogMain, true, notifyChannel)
		if err != nil {
			_, isPathErr := err.(*os.PathError)
			if isPathErr {
				model, err = NewModelFromFile(homeDir, CONFIG_FILE, debugLogMain, true, notifyChannel)
				if err != nil {
					_, isPathErr = err.(*os.PathError)
					if isPathErr {
						exitApp(NewNotifyMessage(ERROR, nil, fmt.Sprintf("Both\nUser config data '%s' and\nLocal config data '%s'\ncould not be found.\nUse '-c=configFileName' for alternative", path, CONFIG_FILE), "", 1, err))
					} else {
						exitApp(NewNotifyMessage(ERROR, nil, "Model Load Error", "", 1, err))
					}
				}
			} else {
				exitApp(NewNotifyMessage(ERROR, nil, "Model Load Error", "", 1, err))
			}
		}
	} else {
		model, err = NewModelFromFile(homeDir, configFileName, debugLogMain, true, notifyChannel)
	}
	if err != nil {
		exitApp(NewNotifyMessage(ERROR, nil, "Model Load Error", "", 1, err))
	}
	err = model.ValidateBackgroundTasks()
	if err != nil {
		exitApp(NewNotifyMessage(ERROR, nil, "Model Validate Background Tasks Error", "", 1, err))
	}
	model.Log()
	go listenNotifyChannel()
	for _, ras := range model.RunAtStart {
		execDelayedAction(ras.action, ras.delay, notifyChannel, model.dataCache)
	}
	gui()
}

/*
Provide a thread for GUI updates that can be triggured via a queue.
*/
func listenNotifyChannel() {
	for {
		notifyMessage := <-notifyChannel
		if debugLogMain.IsLogging() {
			debugLogMain.WriteLog(notifyMessage.String())
		}
		if mainWindowActive {
			switch notifyMessage.state {
			case DONE:
				notifyActionRunning(false, notifyMessage.action.name)
				refresh()
			case START:
				notifyActionRunning(true, notifyMessage.action.name)
			case CMD_RC:
				rc := WarnDialog(fmt.Sprintf("Action '%s' failed:", notifyMessage.action.name), notifyMessage.err.Error(), notifyMessage.message, mainWindow, 99, debugLogMain)
				notifyActionRunning(false, notifyMessage.action.name)
				if rc == 1 {
					exitApp(notifyMessage)
				}
				refresh()
			case ERROR:
				WarnDialog(fmt.Sprintf("Action '%s' failed:", notifyMessage.action.name), notifyMessage.err.Error(), "", mainWindow, 8, debugLogMain)
				refresh()
			case REFRESH:
				refresh()
			case WARN:
				refresh()
			case EXIT:
				exitApp(notifyMessage)
			case LOG:
			}
		}
	}
}

func warningAtStart() {
	if model.warning != "" {
		if debugLogMain.IsLogging() {
			debugLogMain.WriteLog(NewNotifyMessage(WARN, nil, model.warning, "", 0, nil).String())
		}
		go func() {
			for !mainWindowActive {
				time.Sleep(500 * time.Millisecond)
			}
			WarnDialog("Data Load Error", model.warning, "", mainWindow, 5, debugLogMain)
		}()
	}
}

func gui() {
	a := app.NewWithID("stuartdd.gtest")
	mainWindow = a.NewWindow("Main Window")
	mainWindow.SetCloseIntercept(func() {
		actionClose(NewNotifyMessage(EXIT, nil, "Close intercept", "", 0, nil))
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
	warningAtStart()
	mainWindow.SetFixedSize(true)
	mainWindowActive = true
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
	if currentView == VIEW_ACTIONS {
		for _, a := range model.actionList {
			s, _ := substituteValuesIntoString(a.hideExp, nil, model.dataCache)
			a.ShouldHide = strings.Contains(s, "%{") || s == "yes"
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
			cp := centerPanelActions(tabList[singleName])
			c = container.NewBorder(bb, nil, nil, nil, cp)
		}
	}
	if currentView == VIEW_DATA {
		cw := int(math.Floor(float64(mainWindow.Canvas().Size().Width) / float64(MeasureChar())))
		tabs := container.NewAppTabs()
		tabs.Append(container.NewTabItem("Local", container.NewVScroll(centerPanelLocalData(model.dataCache, cw))))
		tabs.Append(container.NewTabItem("Memory", container.NewVScroll(centerPanelMemoryData(model.dataCache, cw))))
		tabs.Append(container.NewTabItem("Env", container.NewVScroll(centerPanelEnvData(model.dataCache, cw))))
		if selectedValueTabIndex >= 0 {
			tabs.SelectIndex(selectedValueTabIndex)
		}
		tabs.OnSelected = func(ti *container.TabItem) {
			selectedValueTabIndex = tabs.SelectedIndex()
		}
		c = container.NewBorder(bb, nil, nil, nil, tabs)
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
		cp := centerPanelActions(actionsByTab[name])
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

func centerPanelEnvData(dataCache *DataCache, cw int) *fyne.Container {
	vp := container.NewVBox()
	vp.Add(widget.NewSeparator())

	sortedNames := dataCache.GetEnvValueNamesSorted()
	max := 0
	for _, n := range sortedNames {
		if len(n) > max {
			max = len(n)
		}
	}
	maxMax := (cw / 5) * 2
	if max > maxMax {
		max = maxMax
	}
	for _, n := range sortedNames {
		ev, found := dataCache.envMap[n]
		if found {
			hp := container.NewHBox()
			var s string
			if len(n) > max {
				s = PadLeft(n, max-2) + "...= "
			} else {
				s = PadLeft(n, max) + " = "
			}
			hp.Add(container.New(NewFixedHLayout(100, 14), NewStringFieldLeft(s+CleanString(ev, cw-(len(s)+1)))))
			vp.Add(hp)
		}
	}
	return vp
}

func centerPanelMemoryData(dataCache *DataCache, cw int) *fyne.Container {
	vp := container.NewVBox()
	vp.Add(widget.NewSeparator())

	sortedNames := dataCache.GetMemoryValueNamesSorted()
	max := 0
	for _, n := range sortedNames {
		if len(n) > max {
			max = len(n)
		}
	}
	for _, n := range sortedNames {
		mv := dataCache.GetCacheWriter(n)
		if mv != nil {
			hp := container.NewHBox()
			s := PadLeft(mv.name, max) + " = "
			hp.Add(container.New(NewFixedHLayout(100, 14), NewStringFieldLeft(s+CleanString(mv.GetContent(), cw-(len(s)+1)))))
			vp.Add(hp)
		}
	}
	return vp
}

func centerPanelLocalData(dataCache *DataCache, cw int) *fyne.Container {
	vp := container.NewVBox()
	vp.Add(widget.NewSeparator())

	sortedNames := dataCache.GetLocalValueNamesSorted()
	max := 0
	for _, n := range sortedNames {
		if len(n) > max {
			max = len(n)
		}
	}
	for _, n := range sortedNames {
		l, found := dataCache.GetLocalValue(n)
		if found {
			if !l.isPassword {
				hp := container.NewHBox()
				s := PadLeft(l.name, max) + " ("
				if l.inputRequired {
					s = s + "I"
				} else {
					s = s + "-"
				}
				if l.inputDone {
					s = s + "D"
				} else {
					s = s + "-"
				}
				if l.isFileName {
					s = s + "F"
				} else {
					s = s + "-"
				}
				if l.isFileWatch {
					s = s + "W"
				} else {
					s = s + "-"
				}
				s = s + ") "
				hp.Add(container.New(NewFixedHLayout(100, 14), NewStringFieldLeft(s+l.GetValueClean(cw-(len(s)+1)))))
				vp.Add(hp)
			}
		}
	}
	return vp
}

func centerPanelActions(actionData []*MultipleActionData) *fyne.Container {
	vp := container.NewVBox()
	vp.Add(widget.NewSeparator())
	min := 3
	for _, l := range actionData {
		if !l.ShouldHide {
			hp := container.NewHBox()
			btn := newActionButton(l.name, theme.SettingsIcon(), func(action *MultipleActionData) {
				if !actionRunning {
					go func() {
						execMultipleAction(action, notifyChannel, model.dataCache)
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
		actionClose(NewNotifyMessage(EXIT, nil, "Exit 0", "", 0, nil))
	}))
	if model.AltExitTitle != "" {
		bb.Add(widget.NewButtonWithIcon(model.AltExitTitle, theme.LogoutIcon(), func() {
			actionClose(NewNotifyMessage(EXIT, nil, "Alt Exit 0", "", model.AltExitRc, nil))
		}))
	}
	bb.Add(widget.NewButtonWithIcon("Reload", theme.MediaReplayIcon(), func() {
		m, err := NewModelFromFile(model.homePath, model.fileName, debugLogMain, true, notifyChannel)
		if err != nil {
			//
			// Warn but don't wait as this button press thread must exit so WarnDialog button can do it's thing
			//
			go WarnDialog("Reload Failed", err.Error(), "", mainWindow, 20, debugLogMain)
		} else {
			err = m.ValidateBackgroundTasks()
			if err != nil {
				//
				// Warn but don't wait as this button press thread must exit so WarnDialog button can do it's thing
				//
				go WarnDialog("Reload Validaion Failed", err.Error(), "", mainWindow, 20, debugLogMain)
			} else {
				model = m
				if debugLogMain.IsLogging() {
					debugLogMain.WriteLog("Model Reloaded")
				}
				for _, ras := range model.RunAtStart {
					execDelayedAction(ras.action, ras.delay, notifyChannel, model.dataCache)
				}
			}
		}
	}))
	if currentView == VIEW_ACTIONS {
		bb.Add(widget.NewButtonWithIcon("Values", theme.ComputerIcon(), func() {
			currentView = VIEW_DATA
			if notifyChannel != nil {
				notifyChannel <- NewNotifyMessage(REFRESH, nil, "View value data", "", 0, nil)
			}
		}))
	} else {
		bb.Add(widget.NewButtonWithIcon("Actions", theme.SettingsIcon(), func() {
			currentView = VIEW_ACTIONS
			if notifyChannel != nil {
				notifyChannel <- NewNotifyMessage(REFRESH, nil, "View action data", "", 0, nil)
			}
		}))
	}
	actionRunningLabel = widget.NewLabel("")
	bb.Add(actionRunningLabel)
	return bb
}

func notifyActionRunning(newState bool, name string) {
	actionRunning = newState
	if actionRunning {
		actionRunningLabel.SetText(fmt.Sprintf("Running '%s'", name))
	} else {
		actionRunningLabel.SetText("")
	}
}

func actionClose(data *NotifyMessage) {
	for _, rae := range model.RunAtEnd {
		if rae != nil && rae.action != nil {
			execMultipleAction(rae.action, nil, model.dataCache)
		}
	}
	mainWindow.Close()
	exitApp(data)
}

func exitApp(data *NotifyMessage) {
	if debugLogMain != nil && debugLogMain.IsLogging() {
		debugLogMain.WriteLog(data.String())
	} else {
		if data.code != 0 {
			fmt.Printf("%sEXIT CODE[%d]:%s%s\n", stdColourPrefix[STD_ERR], data.code, data, RESET)
		} else {
			if data != nil {
				fmt.Printf("%s%s%s\n", stdColourPrefix[STD_OUT], data.String(), RESET)
			}
		}
	}
	if debugLogMain != nil {
		debugLogMain.Close()
	}
	os.Exit(data.code)
}
