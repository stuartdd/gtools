package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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
)

var (
	stdColourPrefix    = []string{GREEN, RED}
	mainWindow         fyne.Window
	model              *Model
	actionRunning      bool = false
	actionRunningLabel *widget.Label
)

type ActionButton struct {
	widget.Button
	action *ActionData
	tapped func(action *ActionData)
}

func main() {
	var err error
	var path string
	if len(os.Args) == 1 {
		path, err = os.UserHomeDir()
		if err != nil {
			exitApp(err.Error(), 1)
		}
		path = path + string(os.PathSeparator) + "gtool-config.json"
		model, err = NewModelFromFile(path)
	} else {
		model, err = NewModelFromFile(os.Args[1])
	}
	if err != nil {
		exitApp(err.Error(), 1)
	}

	if RunAtEnd != "" {
		_, err := model.GetActionDataForName(RunAtEnd)
		if err != nil {
			exitApp(fmt.Sprintf("RunAtEnd: %s", err.Error()), 1)
		}
	}
	if RunAtStart != "" {
		runAtStart()
	}
	gui()
}

func runAtStart() {
	action, err := model.GetActionDataForName(RunAtStart)
	if err != nil {
		exitApp(fmt.Sprintf("RunAtStart: %s", err.Error()), 1)
	}
	go func() {
		time.Sleep(time.Second)
		execMultipleAction(action)
		go func() {
			time.Sleep(time.Second)
			update()
		}()
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
	mainWindow.ShowAndRun()
}

func update() {
	bb := buttonBar(action)
	cp, err := centerPanel()
	if err != nil {
		return
	}
	c := container.NewBorder(bb, nil, nil, nil, cp)
	mainWindow.SetContent(c)
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

func centerPanel() (*fyne.Container, error) {
	vp := container.NewVBox()
	vp.Add(widget.NewSeparator())
	min := 3
	for _, l := range model.actionList {
		if !l.hide {
			hp := container.NewHBox()
			btn := newActionButton(l.name, theme.SettingsIcon(), func(action *ActionData) {
				if !actionRunning {
					go execMultipleAction(action)
				}
			}, l)
			hp.Add(btn)
			hp.Add(widget.NewLabel(MutateStringFromMemCache(l.desc)))
			vp.Add(hp)
			min--
		}
	}
	for min > 0 {
		vp.Add(container.NewHBox(widget.NewLabel("")))
		min--
	}
	return vp, nil
}

func buttonBar(exec func(string, string, string)) *fyne.Container {
	bb := container.NewHBox()
	bb.Add(widget.NewButtonWithIcon("Close(0)", theme.LogoutIcon(), func() {
		exec("exit", "", "")
	}))
	if ShowExit1 {
		bb.Add(widget.NewButtonWithIcon("Close(1)", theme.LogoutIcon(), func() {
			exec("exit1", "", "")
		}))
	}
	bb.Add(widget.NewButtonWithIcon("Reload", theme.MediaReplayIcon(), func() {
		m, err := NewModelFromFile(model.fileName)
		if err != nil {
			fmt.Printf("Failed to reload")
		} else {
			model = m
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
	}, mainWindow).Run().err
}

func execMultipleAction(data *ActionData) {
	setActionRunning(true, data.name)
	defer setActionRunning(false, "")
	stdOut := NewBaseWriter("", stdColourPrefix[STD_OUT])
	stdErr := NewBaseWriter("", stdColourPrefix[STD_ERR])
	for i, act := range data.commands {
		err := execSingleAction(act, stdOut, stdErr, data.desc)
		if err != nil {
			WarnDialog(fmt.Sprintf("Action '%s' step '%d' error", data.desc, i), err.Error(), mainWindow, 5)
			return
		}
		if act.err != nil {
			stdErr.Write([]byte(act.err.Error()))
			stdErr.Write([]byte("\n"))
			WarnDialog(fmt.Sprintf("Action '%s' step '%d' failed", data.desc, i), act.err.Error(), mainWindow, 5)
			return
		}
	}
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
			if cf.value == "" {
				return "", fmt.Errorf("password not provided")
			}
			return cf.value, nil
		}
	}
	return "", nil
}

func execSingleAction(sa *SingleAction, stdOut, stdErr *BaseWriter, actionDesc string) error {
	outEncKey, err := deriveKeyFromName(sa.outPwName, sa)
	if err != nil {
		return err
	}
	inEncKey, err := deriveKeyFromName(sa.inPwName, sa)
	if err != nil {
		return err
	}

	args, err := SubstituteValuesIntoStringList(sa.args, validatedEntryDialog)
	if err != nil {
		return err
	}
	cmd := exec.Command(sa.command, args...)

	if sa.sysin != "" {
		si, err := NewStringReader(sa.sysin, cmd.Stdin)
		if err != nil {
			return err
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
	so := NewWriter(sa.sysoutFile, outEncKey, stdOut, stdErr)
	soReset, reSoOk := so.(Reset)
	if reSoOk {
		soReset.Reset()
	}
	soCloser, soOk := so.(io.Closer)
	if soOk {
		defer soCloser.Close()
	}
	cmd.Stdout = so

	se := NewWriter(sa.syserrFile, outEncKey, stdErr, stdErr)
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
		return err
	}
	sa.err = nil
	sa.err = cmd.Wait()
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
	return nil
}

func SubstituteValuesIntoStringList(s []string, entryDialog func(*InputValue) error) ([]string, error) {
	resp := make([]string, 0)
	for _, v := range s {
		tmp, err := model.MutateStringFromValues(v, entryDialog)
		if err != nil {
			return nil, err
		}
		resp = append(resp, MutateStringFromMemCache(tmp))
	}
	return resp, nil
}

func actionClose(data string, code int) {
	if RunAtEnd != "" {
		action, err := model.GetActionDataForName(RunAtEnd)
		if err != nil {
			exitApp(err.Error(), 1)
		}
		execMultipleAction(action)
	}
	mainWindow.Close()
	exitApp(data, code)
}

func exitApp(data string, code int) {
	if code != 0 {
		fmt.Printf("%sEXIT CODE[%d]:%s%s\n", stdColourPrefix[STD_ERR], code, data, RESET)
	} else {
		if data != "" {
			fmt.Printf("%s%s%s\n", stdColourPrefix[STD_OUT], data, RESET)
		}
	}
	os.Exit(code)
}
