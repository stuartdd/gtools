package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ENUM_ENTRY_TYPE int

const (
	SYSOUT_DIALOG_TYPE ENUM_ENTRY_TYPE = iota
	SYSIN_DIALOG_TYPE
	VALUE_DIALOG_TYPE
)

type MyDialog struct {
	debugLog *LogData
	value    *LocalValue
	parent   fyne.Window
	wait     bool
	err      error
	isValid  func(string, *LocalValue) bool
}

func validatedEntryDialog(localValue *LocalValue) error {
	return NewMyDialog(localValue, func(s string, iv *LocalValue) bool {
		return len(strings.TrimSpace(s)) >= iv.minLen
	}, mainWindow, debugLogMain).Run(VALUE_DIALOG_TYPE).err
}

func sysInDialog(localValue *LocalValue) error {
	return NewMyDialog(localValue, func(s string, iv *LocalValue) bool {
		return true
	}, mainWindow, debugLogMain).Run(SYSIN_DIALOG_TYPE).err
}

func sysOutDialog(localValue *LocalValue) error {
	return NewMyDialog(localValue, func(s string, iv *LocalValue) bool {
		return true
	}, mainWindow, debugLogMain).Run(SYSOUT_DIALOG_TYPE).err
}

func NewMyDialog(value *LocalValue, validate func(string, *LocalValue) bool, parentWindow fyne.Window, debugLog *LogData) *MyDialog {
	return &MyDialog{value: value, isValid: validate, parent: parentWindow, wait: true, err: nil, debugLog: debugLog}
}

func WarnDialog(title, message, additional string, parentWindow fyne.Window, timeout int, debugLog *LogData) int {
	if debugLog.IsLogging() {
		debugLog.WriteLog(fmt.Sprintf("    Warning: title:\"%s\" message:\"%s\" additional:\"%s\"", title, message, additional))
	}
	message = strings.ReplaceAll(message, "'. ", "'.\n")
	wait := true
	rc := 0
	ok := widget.NewButtonWithIcon("OK", theme.ConfirmIcon(), func() {
		wait = false
		rc = 0
	})
	ok.Importance = widget.HighImportance
	buttons := container.NewCenter()
	buttonsHbox := container.NewHBox()
	buttons.Add(buttonsHbox)
	if additional != "" {
		extra := widget.NewButtonWithIcon(additional, theme.ErrorIcon(), func() {
			wait = false
			rc = 1
		})
		buttonsHbox.Add(extra)
	}
	buttonsHbox.Add(ok)

	titleLab := container.NewCenter(widget.NewLabel(title))
	messageLab := container.NewCenter(widget.NewLabel(message))

	border := container.NewBorder(titleLab, buttons, nil, nil, messageLab)
	popup := widget.NewModalPopUp(border, parentWindow.Canvas())
	popup.Show()
	tout := timeout * 2
	for wait {
		time.Sleep(500 * time.Millisecond)
		tout--
		if tout == 0 {
			wait = false
			rc = 9
		}
	}
	popup.Hide()
	return rc
}

func (d *MyDialog) commit(s string) {
	d.value.SetValue(s)
	if d.debugLog.IsLogging() {
		d.debugLog.WriteLog(fmt.Sprintf("    Dialog commit: Value:\"%s\"", d.value))
	}
	d.value.inputDone = true
	d.wait = false
}

func (d *MyDialog) abort(s string) {
	if d.debugLog.IsLogging() {
		d.debugLog.WriteLog(fmt.Sprintf("    Dialog abort:\"%s\" Value:\"%s\"", s, d.value))
	}
	d.err = fmt.Errorf(s)
	d.wait = false
}

func (d *MyDialog) callIsValid(s string) bool {
	if d.isValid == nil {
		return true
	}
	return d.isValid(s, d.value)
}

func (d *MyDialog) onChange(s string, ok *widget.Button) {
	if d.callIsValid(s) {
		ok.Enable()
	} else {
		ok.Disable()
	}
}

func (d *MyDialog) Run(dt ENUM_ENTRY_TYPE) *MyDialog {
	if d.value.isFileName {
		if dt == SYSOUT_DIALOG_TYPE {
			fd := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
				if uc == nil {
					d.abort("output file not selected")
				} else {
					d.commit(uc.URI().Path())
					d.value.lastValue = filepath.Dir(uc.URI().Path())
				}
			}, d.parent)
			l, err := d.value.GetLastValueAsLocation()
			if err != nil {
				d.abort(err.Error())
				return d
			}
			fd.SetLocation(l)
			fd.Show()
		} else {
			fd := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
				if uc == nil {
					d.abort("input file not selected")
				} else {
					d.commit(uc.URI().Path())
					d.value.lastValue = filepath.Dir(uc.URI().Path())
				}
			}, d.parent)
			l, err := d.value.GetLastValueAsLocation()
			if err != nil {
				d.abort(err.Error())
				return d
			}
			fd.SetLocation(l)
			fd.Show()
		}
		d.wait = true
		for d.wait {
			time.Sleep(200 * time.Millisecond)
		}
		return d
	}

	entry := widget.NewEntry()
	if d.value.isPassword {
		entry = widget.NewPasswordEntry()
	}
	entry.SetText(d.value.GetValue())
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
	if d.value.minLen > 0 {
		min = fmt.Sprintf(". (minimum %d chars)", d.value.minLen)
	}
	hBox := container.NewHBox()
	hBox.Add(widget.NewLabel("    "))
	hBox.Add(ca)
	hBox.Add(widget.NewLabel(" "))
	hBox.Add(ok)
	hBox.Add(widget.NewLabel("    "))
	buttons := container.NewCenter(hBox)
	label := container.NewCenter(widget.NewLabel(fmt.Sprintf("Input %s%s", d.value.desc, min)))
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
