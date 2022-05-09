package main

import (
	"fmt"
	"os"
	"os/exec"

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
	resp  = "Hello\nWorld\n"
)

var (
	mainWindow fyne.Window
	stdOut     = newMyWriter(1)
	stdErr     = newMyWriter(2)
	prefix     = []string{RESET, GREEN, RED}
	actionList = make(map[string]*ActionData)
)

type myWriter struct {
	id int
}

type SingleAction struct {
	command string
	args    []string
	sysin   string
	err     error
}

type ActionData struct {
	action   string
	btn      *widget.Button
	commands []*SingleAction
}

func newMyWriter(id int) *myWriter {
	return &myWriter{id: id}
}

func (mw *myWriter) WriteStr(s string) (n int, err error) {
	return mw.Write([]byte(s))
}

func (mw *myWriter) Write(p []byte) (n int, err error) {
	fmt.Printf("%s%s%s", prefix[mw.id], string(p), RESET)
	return len(p), nil
}

func main() {
	stdOut.Write([]byte("\033[;32mGreen Text\033[0m\n"))
	AddAction("List", "ls", []string{"-lta"}, "")
	AddAction("Last", "cat", []string{"/var/log/s"}, "")
	AddAction("Test", "./bashin.sh", []string{}, "Stuart\nBoy")
	AddAction("Push", "ls", []string{}, "")
	AddAction("Push", "./bashin1.sh", []string{}, "")
	gui()
}

func newSingleAction(cmd string, args []string, input string) *SingleAction {
	return &SingleAction{command: cmd, args: args, sysin: input}
}

func newActionData(action string) *ActionData {
	btn := widget.NewButtonWithIcon(action, theme.LogoutIcon(), func() {
		err := execMultipleAction(action, stdOut, stdErr)
		if err != nil {
			stdErr.WriteStr(fmt.Sprintf("%s\n", err.Error()))
		}
	})
	return &ActionData{action: action, commands: make([]*SingleAction, 0), btn: btn}
}

func (p *ActionData) addSingleAction(cmd string, data []string, input string) {
	sa := newSingleAction(cmd, data, input)
	p.commands = append(p.commands, sa)
}

func AddAction(name, cmd string, data []string, in string) {
	ac, ok := actionList[name]
	if !ok {
		ac = newActionData(name)
		actionList[name] = ac
	}
	ac.addSingleAction(cmd, data, in)
}

func gui() {
	a := app.NewWithID("stuartdd.gtest")
	mainWindow = a.NewWindow("Main Window")
	mainWindow.SetCloseIntercept(func() {
		actionClose("", 0)
	})
	bb := buttonBar(action)
	cp := centerPanel()
	c := container.NewBorder(bb, nil, nil, nil, cp)
	mainWindow.SetContent(c)
	mainWindow.SetMaster()
	mainWindow.Resize(fyne.NewSize(300, 100))
	mainWindow.SetFixedSize(true)
	mainWindow.ShowAndRun()
}

func centerPanel() *fyne.Container {
	vp := container.NewVBox()
	for _, l := range actionList {
		hp := container.NewHBox()
		hp.Add(l.btn)
		hp.Add(widget.NewLabel(l.action))
		vp.Add(hp)
	}
	return vp
}

func buttonBar(exec func(string, string, string)) *fyne.Container {
	bb := container.NewHBox()
	bb.Add(widget.NewButtonWithIcon("Exit", theme.LogoutIcon(), func() {
		exec("exit", "", "")
	}))
	return bb
}

func action(exec, data1, data2 string) {
	switch exec {
	case "exit":
		actionClose(data1, 0)
	}
}

func execMultipleAction(key string, stdOut, stdErr *myWriter) error {
	data := actionList[key]
	for _, act := range data.commands {
		execSingleAction(act, stdOut, stdErr)
		if act.err != nil {
			return act.err
		}
	}
	return nil
}

func execSingleAction(sa *SingleAction, stdOut, stdErr *myWriter) {
	cmd := exec.Command(sa.command, sa.args...)
	if sa.sysin != "" {
		cmd.Stdin = NewStringReader(1, sa.sysin)
	}
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr
	sa.err = cmd.Start()
	if sa.err != nil {
		return
	}
	sa.err = cmd.Wait()
}

func actionClose(data string, code int) {
	if code != 0 {
		stdErr.WriteStr(fmt.Sprintf("%s. Return code[%d]\n", data, code))
	}
	if data != "" {
		stdOut.WriteStr(fmt.Sprintf("%s\n", data))
	}
	mainWindow.Close()
	os.Exit(code)
}
