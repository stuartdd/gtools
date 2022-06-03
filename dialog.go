package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type MyDialog struct {
	in     *InputValue
	parent fyne.Window
	wait   bool
	err    error
}

func NewMyDialog(in *InputValue, parentWindow fyne.Window) *MyDialog {
	return &MyDialog{in: in, parent: parentWindow, wait: true, err: nil}
}

func (d *MyDialog) Run() *MyDialog {
	entry := widget.NewEntry()
	if d.in.isPassword {
		entry = widget.NewPasswordEntry()
	}
	entry.SetText(d.in.value)
	ok := widget.NewButtonWithIcon("OK", theme.ConfirmIcon(), func() {
		d.in.value = entry.Text
		d.in.inputDone = true
		d.wait = false
	})
	ca := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		d.err = fmt.Errorf("ation cancelled by user")
		d.wait = false
	})

	min := ""
	if d.in.minLen > 0 {
		min = fmt.Sprintf(". (minimum %d chars)", d.in.minLen)
	}
	hBox := container.NewHBox()
	hBox.Add(widget.NewLabel("    "))
	hBox.Add(ca)
	hBox.Add(widget.NewLabel(" "))
	hBox.Add(ok)
	hBox.Add(widget.NewLabel("    "))
	buttons := container.NewCenter(hBox)
	label := container.NewCenter(widget.NewLabel(fmt.Sprintf("Input %s%s", d.in.desc, min)))
	border := container.NewBorder(label, buttons, nil, nil, entry)

	popup := widget.NewModalPopUp(border, d.parent.Canvas())
	popup.Show()
	d.wait = true
	for d.wait {
		time.Sleep(200 + time.Millisecond)
	}
	popup.Hide()
	return d
}
