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
	mainWindowActive   bool = false
	selectedTabIndex   int  = -1
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

	configFileName, err := getArg("-c")
	if err != nil {
		exitApp(NewNotifyMessage(ERROR, nil, "", "", 1, err))
	}
	logFileName, err := getArg("-l")
	if err != nil {
		exitApp(NewNotifyMessage(ERROR, nil, "", "", 1, err))
	}
	clearLog := hasArg("-lc")

	if logFileName == "" {
		debugLogMain = &LogData{logger: nil, queue: nil}
	} else {
		debugLogMain, err = NewLogData(logFileName, "gtool:", clearLog)
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
	mainWindow.Resize(fyne.NewSize(300, 100))
	mainWindow.SetFixedSize(true)
	warningAtStart()
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
			go WarnDialog("Reload Failed", err.Error(), "", mainWindow, 20, debugLogMain)
		} else {
			err = m.ValidateBackgroundTasks()
			if err != nil {
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
			execMultipleAction(rae.action, notifyChannel, model.dataCache)
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

func getArg(name string) (string, error) {
	namelc := strings.ToLower(name)
	for _, v := range os.Args {
		vlc := strings.ToLower(v)
		if strings.HasPrefix(vlc, namelc) {
			l := 2
			if strings.HasPrefix(vlc, namelc+"=") {
				l = 3
			}
			s := v[l:]
			if len(s) < 1 {
				return "", fmt.Errorf("parameter '%s' value is undefined", namelc)
			}
			return s, nil
		}
	}
	return "", nil
}

func hasArg(name string) bool {
	namelc := strings.ToLower(name)
	for _, v := range os.Args {
		vlc := strings.ToLower(v)
		if vlc == namelc {
			return true
		}
	}
	return false
}
