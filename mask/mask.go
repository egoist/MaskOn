// Package mask is what's on a picture and what's covered: the model behind
// the editor. A Doc is what was found, and never changes; a State is what's
// masked, and is replaced, never changed, so undo keeps the old ones.
// Everything's in fractions of the image, as vision.Box is.
package mask

import (
	"fmt"
	"slices"

	"github.com/egoist/maskon/detect"
	"github.com/egoist/ryui/vision"
)

type Box = vision.Box

// Category is what an Item is.
type Category uint8

const (
	Face Category = iota
	Code
	Text // Item.Kind says which
)

// Key names one thing that can be masked.
type Key struct {
	Cat  Category // Face, Code, or Text for a word
	A, B int      // the face or code in A; or the line in A, the word in B
}

// Item is one private thing found: one face, one code, or a run of words.
type Item struct {
	Cat  Category
	Kind detect.Kind // for Text
	Keys []Key
	Box  Box
	Line int // the line, for Text; -1 otherwise
}

// Doc is what was found in an image.
type Doc struct {
	Width, Height int
	Lines         []vision.Line
	Faces         []Box
	Codes         []vision.Code
	Items         []Item
}

// New reads what vision found, and picks out what's private.
func New(r vision.Result) *Doc {
	d := &Doc{Width: r.Width, Height: r.Height, Lines: r.Lines, Faces: r.Faces, Codes: r.Codes}
	for i, f := range d.Faces {
		d.Items = append(d.Items, Item{Cat: Face, Keys: []Key{{Face, i, 0}}, Box: d.faceBox(f), Line: -1})
	}
	for i, c := range d.Codes {
		d.Items = append(d.Items, Item{Cat: Code, Keys: []Key{{Code, i, 0}}, Box: d.pad(c.Box, 0.03), Line: -1})
	}
	for li, l := range d.Lines {
		for _, s := range detect.Line(l.Text) {
			var keys []Key
			var box Box
			for wi, w := range l.Words {
				if w.Start < s.End && s.Start < w.End {
					keys = append(keys, Key{Text, li, wi})
					box = box.Union(d.wordBox(li, wi))
				}
			}
			if len(keys) > 0 {
				d.Items = append(d.Items, Item{Cat: Text, Kind: s.Kind, Keys: keys, Box: d.padLine(box), Line: li})
			}
		}
	}
	return d
}

// faceBox takes in what a face detector leaves out: the hair and the chin.
func (d *Doc) faceBox(f Box) Box {
	return clamp(Box{X: f.X - f.W*0.15, Y: f.Y - f.H*0.3, W: f.W * 1.3, H: f.H * 1.45})
}

// pad grows b by frac of its size on each side, within the image.
func (d *Doc) pad(b Box, frac float64) Box {
	return clamp(Box{X: b.X - b.W*frac, Y: b.Y - b.H*frac, W: b.W * (1 + 2*frac), H: b.H * (1 + 2*frac)})
}

// padLine grows a box of text by the same distance on all sides, a share of
// its height: OCR's boxes are tight, and a descender showing is a leak.
func (d *Doc) padLine(b Box) Box {
	py := b.H * 0.2
	px := py
	if d.Width > 0 {
		px = py * float64(d.Height) / float64(d.Width)
	}
	return clamp(Box{X: b.X - px, Y: b.Y - py, W: b.W + 2*px, H: b.H + 2*py})
}

func clamp(b Box) Box {
	x0, y0 := max(b.X, 0), max(b.Y, 0)
	x1, y1 := min(b.X+b.W, 1), min(b.Y+b.H, 1)
	return Box{X: x0, Y: y0, W: max(x1-x0, 0), H: max(y1-y0, 0)}
}

// wordBox is the word's box, as tall as its line: a word's own box stops at
// its letters, so "a" and "g" side by side would be different heights.
// Except on a line at an angle, whose box is much taller than its words.
func (d *Doc) wordBox(line, word int) Box {
	l := d.Lines[line]
	b := l.Words[word].Box
	tallest := 0.0
	for _, w := range l.Words {
		tallest = max(tallest, w.Box.H)
	}
	if l.Box.H <= tallest*2 {
		b.Y, b.H = l.Box.Y, l.Box.H
	}
	return b
}

// LineBox is the line's box, padded.
func (d *Doc) LineBox(line int) Box { return d.padLine(d.Lines[line].Box) }

// Rect is a rectangle the user drew.
type Rect struct {
	ID  int
	Box Box
}

// State is what's masked. It's replaced, never changed.
type State struct {
	on    map[Key]bool
	rects []Rect
	next  int
}

// Empty is nothing masked.
func Empty() *State { return &State{on: map[Key]bool{}} }

func (s *State) clone() *State {
	n := &State{on: make(map[Key]bool, len(s.on)), rects: slices.Clone(s.rects), next: s.next}
	for k, v := range s.on {
		if v {
			n.on[k] = true
		}
	}
	return n
}

// With is s with the keys masked (on) or not.
func (s *State) With(on bool, keys ...Key) *State {
	n := s.clone()
	for _, k := range keys {
		if on {
			n.on[k] = true
		} else {
			delete(n.on, k)
		}
	}
	return n
}

// Toggle is s with k the other way.
func (s *State) Toggle(k Key) *State { return s.With(!s.on[k], k) }

// Has says k is masked.
func (s *State) Has(k Key) bool { return s.on[k] }

// HasAll says every key is masked.
func (s *State) HasAll(keys []Key) bool {
	for _, k := range keys {
		if !s.on[k] {
			return false
		}
	}
	return len(keys) > 0
}

// AddRect is s with a rectangle drawn.
func (s *State) AddRect(b Box) *State {
	n := s.clone()
	n.next++
	n.rects = append(n.rects, Rect{n.next, clamp(b)})
	return n
}

// WithoutRect is s without the rectangle.
func (s *State) WithoutRect(id int) *State {
	n := s.clone()
	n.rects = slices.DeleteFunc(n.rects, func(r Rect) bool { return r.ID == id })
	return n
}

// IsEmpty says nothing's masked.
func (s *State) IsEmpty() bool { return len(s.on) == 0 && len(s.rects) == 0 }

// LineKeys are the line's words.
func (d *Doc) LineKeys(line int) []Key {
	keys := make([]Key, len(d.Lines[line].Words))
	for i := range keys {
		keys[i] = Key{Text, line, i}
	}
	return keys
}

// Mask is one rectangle to cover: what a mask view shows, and what's drawn
// on the exported image. Masked neighbors in a line are one Mask.
type Mask struct {
	ID   string
	Box  Box
	Keys []Key // what removing it unmasks
	Rect int   // the drawn rectangle's ID, when it's one
	Line int   // the line, when it's text; else -1
}

// Masks is everything covered, in s.
func (d *Doc) Masks(s *State) []Mask {
	var out []Mask
	for i, f := range d.Faces {
		if k := (Key{Face, i, 0}); s.on[k] {
			out = append(out, Mask{ID: fmt.Sprintf("f%d", i), Box: d.faceBox(f), Keys: []Key{k}, Line: -1})
		}
	}
	for i, c := range d.Codes {
		if k := (Key{Code, i, 0}); s.on[k] {
			out = append(out, Mask{ID: fmt.Sprintf("c%d", i), Box: d.pad(c.Box, 0.03), Keys: []Key{k}, Line: -1})
		}
	}
	for li, l := range d.Lines {
		for wi := 0; wi < len(l.Words); {
			if !s.on[Key{Text, li, wi}] {
				wi++
				continue
			}
			from := wi
			var box Box
			var keys []Key
			for ; wi < len(l.Words) && s.on[Key{Text, li, wi}]; wi++ {
				box = box.Union(d.wordBox(li, wi))
				keys = append(keys, Key{Text, li, wi})
			}
			out = append(out, Mask{ID: fmt.Sprintf("l%d:%d-%d", li, from, wi), Box: d.padLine(box), Keys: keys, Line: li})
		}
	}
	for _, r := range s.rects {
		out = append(out, Mask{ID: fmt.Sprintf("r%d", r.ID), Box: r.Box, Rect: r.ID, Line: -1})
	}
	return out
}

// Without is s without the mask.
func (s *State) Without(m Mask) *State {
	if m.Rect != 0 {
		return s.WithoutRect(m.Rect)
	}
	return s.With(false, m.Keys...)
}

// AllKeys is every key of the items.
func AllKeys(items []Item) []Key {
	var keys []Key
	for _, it := range items {
		keys = append(keys, it.Keys...)
	}
	return keys
}

// Pixels is the box in pixels of an image of w×h, rounded outward so a
// partly covered pixel is covered.
func Pixels(b Box, w, h int) (x, y, pw, ph int) {
	x0 := int(b.X * float64(w))
	y0 := int(b.Y * float64(h))
	x1 := min(ceil(b.X+b.W, w), w)
	y1 := min(ceil(b.Y+b.H, h), h)
	return x0, y0, x1 - x0, y1 - y0
}

func ceil(f float64, n int) int {
	v := f * float64(n)
	i := int(v)
	if float64(i) < v {
		i++
	}
	return i
}

// Screen maps between the image's fractions and a canvas of size cw×ch on
// which an image fits, scaled by Scale around the canvas's center and moved
// by TX, TY.
type Screen struct {
	CW, CH        float32 // the canvas
	W, H          float32 // the image at scale 1
	Scale, TX, TY float32
}

// Fit is the size of an image of w×h pixels in a canvas, with a margin.
func Fit(w, h int, cw, ch float32) (float32, float32) {
	if w <= 0 || h <= 0 || cw <= 0 || ch <= 0 {
		return 0, 0
	}
	ar := float32(w) / float32(h)
	fw, fh := cw, cw/ar
	if fh > ch {
		fw, fh = ch*ar, ch
	}
	return fw, fh
}

// ToImage is the point on the image (fractions) under a point of the canvas.
func (s Screen) ToImage(x, y float32) (float32, float32) {
	return ((x-s.CW/2-s.TX)/s.Scale + s.W/2) / s.W, ((y-s.CH/2-s.TY)/s.Scale + s.H/2) / s.H
}

// ToCanvas is the point of the canvas showing a point of the image.
func (s Screen) ToCanvas(fx, fy float32) (float32, float32) {
	return s.CW/2 + s.TX + s.Scale*(fx*s.W-s.W/2), s.CH/2 + s.TY + s.Scale*(fy*s.H-s.H/2)
}

// Clamped is s moved so the image stays on the canvas: centered along an
// axis on which it's smaller, and no gap beside it along one on which it's
// larger.
func (s Screen) Clamped() Screen {
	limit := func(t, size, canvas float32) float32 {
		m := (size - canvas) / 2
		if m <= 0 {
			return 0
		}
		return min(max(t, -m), m)
	}
	s.TX = limit(s.TX, s.W*s.Scale, s.CW)
	s.TY = limit(s.TY, s.H*s.Scale, s.CH)
	return s
}
