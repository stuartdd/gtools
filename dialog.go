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
	window fyne.Window
	size   fyne.Size
	in     *InputValue
	wait   bool
}

func NewMyDialog(in *InputValue) *MyDialog {
	t := fmt.Sprintf("Required input: %s", in.desc)
	w := fyne.CurrentApp().NewWindow(t)
	size := fyne.NewSize(300, 200)
	return &MyDialog{window: w, size: size, in: in, wait: true}
}

func (d *MyDialog) Run() error {
	entry := widget.NewEntry()
	if d.in.isPassword {
		entry = widget.NewPasswordEntry()
	}
	entry.SetText(d.in.value)
	ok := widget.NewButtonWithIcon("OK", theme.ConfirmIcon(), func() {
		d.wait = false
		d.in.inputDone = true
	})
	ca := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		d.wait = false
	})
	vBox := container.NewVBox()
	hBox1 := container.NewHBox()
	hBox1.Add(widget.NewLabel(fmt.Sprintf("Input: %d", d.in.minLen)))
	hBox1.Add(entry)
	vBox.Add(hBox1)
	hBox2 := container.NewHBox()
	hBox2.Add(ok)
	hBox2.Add(ca)
	vBox.Add(hBox2)
	d.window.SetContent(vBox)
	d.window.Resize(d.size)
	d.window.SetFixedSize(true)
	d.window.Show()
	d.wait = true
	for d.wait {
		time.Sleep(200 + time.Millisecond)
	}
	d.window.Hide()
	return nil
}
