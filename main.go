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

var mainWindow fyne.Window

type MyWriter struct {
	id int
}

func NewMyWriter(id int) *MyWriter {
	return &MyWriter{id: id}
}

func (mw *MyWriter) Write(p []byte) (n int, err error) {
	fmt.Printf("ZZZZZ\n\n\n\n\n\n %d[] ", len(p))
	return len(p), nil
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

var actionList = make(map[string]*ActionData)

func main() {
	AddAction("List", "ls", []string{"-lta"}, "")
	AddAction("Last", "cat", []string{"/var/log/syslog"}, "")
	// AddAction("Last", "echo", []string{"World"}, "")
	AddAction("Push", "git", []string{"push"}, "stuartdd\nhi")
	gui()
}

func newSingleAction(cmd string, args []string, input string) *SingleAction {
	return &SingleAction{command: cmd, args: args, sysin: input}
}

func newActionData(action string) *ActionData {
	btn := widget.NewButtonWithIcon(action, theme.LogoutIcon(), func() {
		err := execMultipleAction(action)
		if err != nil {
			fmt.Printf("%s", err.Error())
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

func execMultipleAction(key string) error {
	data := actionList[key]
	for _, act := range data.commands {
		execSingleAction(act)
		if act.err != nil {
			return act.err
		}
	}
	return nil
}

func execSingleAction(sa *SingleAction) {
	cmd := exec.Command(sa.command, sa.args...)
	// if sa.sysin != "" {
	// 	cmd.Stdin = strings.NewReader(sa.sysin)
	// }
	cmd.Stdout = NewMyWriter(1)
	cmd.Stderr = NewMyWriter(2)
	sa.err = cmd.Start()
	if sa.err != nil {
		return
	}
	sa.err = cmd.Wait()
}

func actionClose(data string, code int) {
	if data != "" {
		fmt.Println(data)
	}
	mainWindow.Close()
	os.Exit(code)
}
