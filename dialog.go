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
	in      *InputValue
	parent  fyne.Window
	wait    bool
	err     error
	isValid func(string, *InputValue) bool
}

func NewMyDialog(in *InputValue, validate func(string, *InputValue) bool, parentWindow fyne.Window) *MyDialog {
	return &MyDialog{in: in, isValid: validate, parent: parentWindow, wait: true, err: nil}
}

func WarnDialog(title, message string, parentWindow fyne.Window) {
	wait := true
	ok := widget.NewButtonWithIcon("OK", theme.ConfirmIcon(), func() {
		wait = false
	})
	titleLab := container.NewCenter(widget.NewLabel(title))
	messageLab := container.NewCenter(widget.NewLabel(message))
	buttons := container.NewCenter(ok)

	border := container.NewBorder(titleLab, buttons, nil, nil, messageLab)
	popup := widget.NewModalPopUp(border, parentWindow.Canvas())
	popup.Show()
	for wait {
		time.Sleep(200 + time.Millisecond)
	}
	popup.Hide()
}

func (d *MyDialog) onChange(s string, ok *widget.Button, in *InputValue) {
	if d.isValid == nil {
		ok.Enable()
	} else {
		if d.isValid(s, d.in) {
			ok.Enable()
		} else {
			ok.Disable()
		}
	}
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
		d.err = fmt.Errorf("input cancelled by user")
		d.wait = false
	})

	entry.OnChanged = func(s string) {
		d.onChange(entry.Text, ok, d.in)
	}
	d.onChange(entry.Text, ok, d.in)

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
