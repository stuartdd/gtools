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

type JsonData struct {
	action string
	data1  []string
	data2  []string
}

func main() {
	gui()
}

func gui() {
	a := app.NewWithID("stuartdd.gtest")
	mainWindow = a.NewWindow("Main Window")
	mainWindow.SetCloseIntercept(func() {
		actionClose([]string{}, 0)
	})
	bb := buttonBar(action)
	cp := centerPanel(func(exec string, data []string) {
		action(exec, data)
	})
	c := container.NewBorder(bb, nil, nil, nil, cp)
	mainWindow.SetContent(c)
	mainWindow.SetMaster()
	mainWindow.Resize(fyne.NewSize(300, 100))
	mainWindow.SetFixedSize(true)
	mainWindow.ShowAndRun()
}

func centerPanel(exec func(action string, data []string)) *fyne.Container {
	vp := container.NewVBox()
	vp.Add(widget.NewButtonWithIcon("Exec", theme.LogoutIcon(), func() {
		exec("exec", []string{"ls", "-lta"})
	}))
	return vp
}

func buttonBar(exec func(action string, data []string)) *fyne.Container {
	bb := container.NewHBox()
	bb.Add(widget.NewButtonWithIcon("Exit", theme.LogoutIcon(), func() {
		exec("exit", []string{})
	}))
	return bb
}

func action(exec string, data []string) {
	switch exec {
	case "exit":
		actionClose(data, 0)
	case "exec":
		actionExec(data)
	}
}

func actionExec(data []string, sysin []string) {
	cmd := exec.Command(data[0], data[1:]...)
	// cmd.Stdin = strings.NewReader("and old falcon")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(out.String())
}

func actionClose(data []string, code int) {
	if len(data) > 0 {
		fmt.Println(data)
	}
	mainWindow.Close()
	os.Exit(code)
}
