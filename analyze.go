package main

import (
	"os"

	"github.com/egoist/ryui/imageedit"
	"github.com/egoist/ryui/vision"
)

// languages are what OCR looks for. Chinese comes first whatever the app's
// language: a screenshot is whatever its app was in, and Vision's Chinese
// model reads Latin letters too, while with English first it reads Chinese
// as nonsense.
func languages() []string { return []string{"zh-Hans", "zh-Hant", "en-US"} }

// band is a slice of a tall image: the pixels it reads, and the part of them
// whose finds it keeps. The reads overlap so that a line cut in two by one
// slice is whole in the next, and a line is kept by the slice in whose band
// its middle falls.
type band struct {
	top, height      int // read, in pixels
	keepFrom, keepTo int // kept, in pixels
}

// bands slices an image of w×h: one band unless it's much taller than wide,
// when OCR, which scales what it's given to its own size, would lose the
// small text of a long screenshot.
func bands(w, h int) []band {
	if h <= w*5/2 {
		return []band{{0, h, 0, h}}
	}
	size := w * 2
	overlap := w / 4
	var out []band
	for top := 0; ; top += size - overlap {
		b := band{top: top, height: min(size, h-top), keepFrom: top + overlap/2, keepTo: top + size - overlap/2}
		if top == 0 {
			b.keepFrom = 0
		}
		if top+size >= h {
			b.keepTo = h
			return append(out, b)
		}
		out = append(out, b)
	}
}

// remap puts what a band found into the whole image's fractions, keeping what
// the band owns.
func remap(r vision.Result, b band, w, h int, into *vision.Result) {
	box := func(x vision.Box) vision.Box {
		fh := float64(b.height) / float64(h)
		return vision.Box{X: x.X, Y: (float64(b.top) + x.Y*float64(b.height)) / float64(h), W: x.W, H: x.H * fh}
	}
	kept := func(x vision.Box) bool {
		mid := float64(b.top) + (x.Y+x.H/2)*float64(b.height)
		return mid >= float64(b.keepFrom) && mid < float64(b.keepTo)
	}
	for _, l := range r.Lines {
		if !kept(l.Box) {
			continue
		}
		l.Box = box(l.Box)
		for i := range l.Words {
			l.Words[i].Box = box(l.Words[i].Box)
		}
		into.Lines = append(into.Lines, l)
	}
	for _, f := range r.Faces {
		if kept(f) {
			into.Faces = append(into.Faces, box(f))
		}
	}
	for _, c := range r.Codes {
		if kept(c.Box) {
			c.Box = box(c.Box)
			into.Codes = append(into.Codes, c)
		}
	}
}

// analyze reads the image, a slice at a time if it's long. Not on the main
// thread.
func analyze(path string) (vision.Result, error) {
	o := vision.Options{Languages: languages()}
	w, h, err := imageedit.Size(path)
	if err != nil {
		return vision.Result{}, err
	}
	bs := bands(w, h)
	if len(bs) == 1 {
		return vision.Analyze(path, o)
	}
	out := vision.Result{Width: w, Height: h}
	for _, b := range bs {
		cut, err := imageedit.Edit(path, []imageedit.Action{imageedit.Crop(0, b.top, w, b.height)}, imageedit.Options{Format: imageedit.PNG})
		if err != nil {
			return out, err
		}
		r, err := vision.Analyze(cut.Path, o)
		os.Remove(cut.Path)
		if err != nil {
			return out, err
		}
		remap(r, b, w, h, &out)
	}
	return out, nil
}
