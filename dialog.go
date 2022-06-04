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

func WarnDialog(title, message string, parentWindow fyne.Window, timeout int) {
	wait := true
	ok := widget.NewButtonWithIcon("OK", theme.ConfirmIcon(), func() {
		wait = false
	})
	ok.Importance = widget.HighImportance
	titleLab := container.NewCenter(widget.NewLabel(title))
	messageLab := container.NewCenter(widget.NewLabel(message))
	buttons := container.NewCenter(ok)

	border := container.NewBorder(titleLab, buttons, nil, nil, messageLab)
	popup := widget.NewModalPopUp(border, parentWindow.Canvas())
	popup.Show()
	tout := timeout * 2
	for wait {
		time.Sleep(500 * time.Millisecond)
		tout--
		if tout == 0 {
			wait = false
		}
	}
	popup.Hide()
}

func (d *MyDialog) commit(s string) {
	d.in.value = s
	d.in.inputDone = true
	d.wait = false
}

func (d *MyDialog) abort(s string) {
	d.err = fmt.Errorf(s)
	d.wait = false
}

func (d *MyDialog) callIsValid(s string) bool {
	if d.isValid == nil {
		return true
	}
	return d.isValid(s, d.in)
}

func (d *MyDialog) onChange(s string, ok *widget.Button) {
	if d.callIsValid(s) {
		ok.Enable()
	} else {
		ok.Disable()
	}
}

func (d *MyDialog) Run() *MyDialog {
	entry := widget.NewEntry()
	if d.in.isPassword {
		entry = widget.NewPasswordEntry()
	}
	entry.SetText(d.in.value)
	entry.OnSubmitted = func(s string) {
		if d.callIsValid(entry.Text) {
			d.commit(entry.Text)
		}
	}

	ok := widget.NewButtonWithIcon("OK", theme.ConfirmIcon(), func() {
		d.commit(entry.Text)
	})
	ok.Importance = widget.HighImportance

	ca := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		d.abort("input cancelled by user")
	})

	entry.OnChanged = func(s string) {
		d.onChange(entry.Text, ok)
	}
	d.onChange(entry.Text, ok)

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
	popup.Canvas.Focus(entry)
	time.Sleep(200 + time.Millisecond)
	d.wait = true
	for d.wait {
		time.Sleep(200 * time.Millisecond)
	}
	popup.Hide()
	return d
}
