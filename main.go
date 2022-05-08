package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var mainWindow fyne.Window

type SingleAction struct {
	command string
	args    []string
	in      string
}

type ActionData struct {
	action   string
	btn      *widget.Button
	commands []*SingleAction
}

var actionList = make(map[string]*ActionData)

func main() {
	AddAction("List", "ls", []string{"-lta"})
	AddAction("Last", "echo", []string{"Hello"})
	AddAction("Last", "echo", []string{"World"})
	gui()
}

func newSingleAction(cmd string, args []string, input string) *SingleAction {
	return &SingleAction{command: cmd, args: args, in: input}
}

func newActionData(action string) *ActionData {
	btn := widget.NewButtonWithIcon(action, theme.LogoutIcon(), func() {
		execMultipleAction(action)
	})
	return &ActionData{action: action, commands: make([]*SingleAction, 0), btn: btn}
}

func (p *ActionData) addSingleAction(cmd string, data []string) {
	sa := newSingleAction(cmd, data)
	p.commands = append(p.commands, sa)
}

func AddAction(name, cmd string, data []string) {
	ac, ok := actionList[name]
	if !ok {
		ac = newActionData(name)
		actionList[name] = ac
	}
	ac.addSingleAction(cmd, data)
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

func execMultipleAction(key string) {
	data := actionList[key]
	for _, act := range data.commands {
		execSingleAction(act)
	}
}

func execSingleAction(sa *SingleAction) {
	cmd := exec.Command(sa.command, sa.args...)
	if sa.in != "" {
		var
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(stdout.String())
}

func actionClose(data string, code int) {
	if data != "" {
		fmt.Println(data)
	}
	mainWindow.Close()
	os.Exit(code)
}
