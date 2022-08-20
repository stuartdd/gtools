package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var ts = fyne.TextStyle{Bold: true, Italic: false, Monospace: true, Symbol: false, TabWidth: 2}

type FixedHLayout struct {
	minW float32
	h    float32
}

func NewFixedHLayout(minW, h float32) *FixedHLayout {
	return &FixedHLayout{minW: minW, h: h}
}

func (d *FixedHLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(d.minW, d.h)
}

func (d *FixedHLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	for _, o := range objects {
		o.Resize(fyne.NewSize(d.minW, d.h))
	}
}

func MeasureChar() float32 {
	ts := fyne.TextStyle{Bold: true, Italic: false, Monospace: true, Symbol: false, TabWidth: 2}
	si := fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
	return fyne.MeasureText("M", si, ts).Width
}

func NewStringFieldLeft(s string) *widget.Label {
	ts := fyne.TextStyle{Bold: true, Italic: false, Monospace: true, Symbol: false, TabWidth: 2}
	return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, ts)
}
