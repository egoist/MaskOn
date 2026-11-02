package main

import (
	"testing"

	"github.com/egoist/ryui/vision"
)

func TestBandsOneForOrdinaryImages(t *testing.T) {
	if b := bands(1170, 2532); len(b) != 1 || b[0] != (band{0, 2532, 0, 2532}) {
		t.Fatalf("%+v", b)
	}
}

func TestBandsCoverLongImages(t *testing.T) {
	const w, h = 1000, 9300
	bs := bands(w, h)
	if len(bs) < 4 {
		t.Fatalf("%+v", bs)
	}
	at := 0
	for i, b := range bs {
		if b.keepFrom != at {
			t.Fatalf("band %d keeps from %d, after %d: %+v", i, b.keepFrom, at, bs)
		}
		if b.top > b.keepFrom || b.top+b.height < b.keepTo {
			t.Fatalf("band %d keeps what it doesn't read: %+v", i, b)
		}
		if b.top+b.height > h {
			t.Fatalf("band %d reads past the end: %+v", i, b)
		}
		at = b.keepTo
	}
	if at != h {
		t.Fatalf("ends at %d", at)
	}
}

func TestRemapKeepsOneCopyOfALineInTheOverlap(t *testing.T) {
	const w, h = 1000, 5000
	bs := bands(w, h)
	var out vision.Result
	// A line at a fixed place in the whole image, as each band that reads it
	// would report it.
	const lineTop, lineH = 1800.0, 40.0
	for _, b := range bs {
		if lineTop < float64(b.top) || lineTop+lineH > float64(b.top+b.height) {
			continue
		}
		l := vision.Line{Text: "x", Box: vision.Box{X: 0.1, Y: (lineTop - float64(b.top)) / float64(b.height), W: 0.5, H: lineH / float64(b.height)}}
		remap(vision.Result{Lines: []vision.Line{l}}, b, w, h, &out)
	}
	if len(out.Lines) != 1 {
		t.Fatalf("%d copies", len(out.Lines))
	}
	got := out.Lines[0].Box
	if diff := got.Y*h - lineTop; diff > 0.01 || diff < -0.01 {
		t.Fatalf("y %v", got.Y*h)
	}
	if diff := got.H*h - lineH; diff > 0.01 || diff < -0.01 {
		t.Fatalf("h %v", got.H*h)
	}
}
