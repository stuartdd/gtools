package main

import (
	"net/http"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

var ts = fyne.TextStyle{Bold: true, Italic: false, Monospace: true, Symbol: false, TabWidth: 2}

func NewLayout(url, mimetype, data string) (int, error) {
	resp, err := http.Post(url, "text/plain", strings.NewReader(data))
	if err != nil {
		return 999, err
	}
	return resp.StatusCode, nil
}

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
		if containerSize.Width < d.minW {
			o.Resize(fyne.NewSize(d.minW, d.h))
		} else {
			o.Resize(fyne.NewSize(containerSize.Width, d.h))
		}
	}
}

func NewStringFieldRight(s string, w int) *widget.Label {
	ts := fyne.TextStyle{Bold: true, Italic: false, Monospace: true, Symbol: false, TabWidth: 2}
	return widget.NewLabelWithStyle(PadLeft(s, w), fyne.TextAlignLeading, ts)
}

func NewStringFieldLeft(s string, w int) *widget.Label {
	ts := fyne.TextStyle{Bold: true, Italic: false, Monospace: true, Symbol: false, TabWidth: 2}
	return widget.NewLabelWithStyle(PadRight(s, w), fyne.TextAlignLeading, ts)
}
