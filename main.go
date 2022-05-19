package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	RESET = "\033[0m"
	GREEN = "\033[;32m"
	RED   = "\033[;31m"
)

var (
	stdColourPrefix = []string{GREEN, RED}
	mainWindow      fyne.Window
	model           *Model
)

type ActionButton struct {
	widget.Button
	action *ActionData
	tapped func(action *ActionData)
}

func main() {
	var err error
	if len(os.Args) == 1 {
		model, err = NewModelFromFile("config.json")
	} else {
		model, err = NewModelFromFile(os.Args[1])
	}
	if err != nil {
		exitApp(err.Error(), 1)
	}

	err = model.LoadInputFields()
	if err != nil {
		exitApp(err.Error(), 1)
	}
	if RunAtStart != "" {
		action, err := model.GetActionDataForName(RunAtStart)
		if err != nil {
			exitApp(err.Error(), 1)
		}
		execMultipleAction(action)
	}
	gui()
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
	cp := centerPanel()
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

func centerPanel() *fyne.Container {
	vp := container.NewVBox()
	vp.Add(widget.NewSeparator())
	for _, l := range model.actionList {
		if !l.hide {
			hp := container.NewHBox()
			btn := newActionButton(l.name, theme.SettingsIcon(), func(action *ActionData) {
				execMultipleAction(action)
			}, l)
			hp.Add(btn)
			hp.Add(widget.NewLabel(MutateStringFromMemCache(l.desc)))
			vp.Add(hp)
		}
	}
	return vp
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
	return bb
}

func action(exec, data1, data2 string) {
	switch exec {
	case "exit":
		actionClose(data1, 0)
	case "exit1":
		actionClose(data1, 1)
	}
}

func entryDialog(desc, value string) (string, error) {
	ret := value
	var err error = nil
	wait := true
	entry := widget.NewEntry()
	items := make([]*widget.FormItem, 0)
	items = append(items, widget.NewFormItem("Default value is:", widget.NewLabel(fmt.Sprintf("'%s'", value))))
	items = append(items, widget.NewFormItem("Enter new value:", entry))
	d := dialog.NewForm("This action requires a "+desc, "OK", "Abort", items, func(b bool) {
		if b {
			if entry.Text != "" {
				ret = entry.Text
			}
		} else {
			err = fmt.Errorf("action aborted by user")
		}
		wait = false
	}, mainWindow)
	d.Show()
	for wait {
		time.Sleep(100 + time.Millisecond)
	}
	return ret, err
}

func execMultipleAction(data *ActionData) {
	model.ResetCacheValues()
	stdOut := NewBaseWriter("", stdColourPrefix[STD_OUT])
	stdErr := NewBaseWriter("", stdColourPrefix[STD_ERR])
	go func() {
		for _, act := range data.commands {
			err := execSingleAction(act, stdOut, stdErr)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			if act.err != nil {
				stdErr.Write([]byte(act.err.Error()))
				stdErr.Write([]byte("\n"))
				return
			}
		}
	}()
}

func execSingleAction(sa *SingleAction, stdOut, stdErr *BaseWriter) error {
	var err error = nil
	args := MutateListFromMemCache(sa.args)
	args, err = model.MutateListFromValues(args, entryDialog)
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
		cmd.Stdin = si
	}
	so := NewWriter(sa.sysoutFile, sa.outFilter, stdOut, stdErr)
	soReset, reSoOk := so.(Reset)
	if reSoOk {
		soReset.Reset()
	}
	soCloser, soOk := so.(io.Closer)
	if soOk {
		defer soCloser.Close()
	}
	cmd.Stdout = so

	se := NewWriter(sa.syserrFile, sa.outFilter, stdErr, stdErr)
	seReset, reSeOk := se.(Reset)
	if reSeOk {
		seReset.Reset()
	}
	seCloser, seOk := se.(io.Closer)
	if seOk {
		defer seCloser.Close()
	}
	cmd.Stderr = se

	sa.err = nil
	sa.err = cmd.Start()
	if sa.err != nil {
		return sa.err
	}
	sa.err = cmd.Wait()
	if sa.delay > 0.0 {
		time.Sleep(time.Duration(sa.delay) * time.Millisecond)
	}
	cp, ok := so.(ClipContent)
	if ok {
		if cp.shouldClip() {
			mainWindow.Clipboard().SetContent(cp.getContent())
		}
	}
	return nil
}

func actionClose(data string, code int) {
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
