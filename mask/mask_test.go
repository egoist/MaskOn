package mask

import (
	"math"
	"testing"

	"github.com/egoist/maskon/detect"
	"github.com/egoist/ryui/vision"
)

func testDoc() *Doc {
	line := vision.Line{
		Text: "Call 13812345678 now",
		Box:  Box{X: 0.1, Y: 0.1, W: 0.8, H: 0.05},
		Words: []vision.Word{
			{Text: "Call", Box: Box{X: 0.1, Y: 0.11, W: 0.1, H: 0.03}, Start: 0, End: 4},
			{Text: "13812345678", Box: Box{X: 0.25, Y: 0.1, W: 0.4, H: 0.05}, Start: 5, End: 16},
			{Text: "now", Box: Box{X: 0.7, Y: 0.12, W: 0.2, H: 0.02}, Start: 17, End: 20},
		},
	}
	return New(vision.Result{
		Width: 1000, Height: 2000,
		Lines: []vision.Line{line},
		Faces: []Box{{X: 0.5, Y: 0.5, W: 0.2, H: 0.1}},
		Codes: []vision.Code{{Kind: "qr", Value: "x", Box: Box{X: 0.1, Y: 0.7, W: 0.2, H: 0.1}}},
	})
}

func TestNewFindsItems(t *testing.T) {
	d := testDoc()
	if len(d.Items) != 3 {
		t.Fatalf("%+v", d.Items)
	}
	phone := d.Items[2]
	if phone.Cat != Text || phone.Kind != detect.Phone || phone.Line != 0 || len(phone.Keys) != 1 || phone.Keys[0].B != 1 {
		t.Fatalf("%+v", phone)
	}
}

func TestMasksMergeNeighbors(t *testing.T) {
	d := testDoc()
	s := Empty().With(true, Key{Text, 0, 0}, Key{Text, 0, 1})
	ms := d.Masks(s)
	if len(ms) != 1 || len(ms[0].Keys) != 2 {
		t.Fatalf("%+v", ms)
	}
	// The whole height of the line, and past both ends of its words.
	b := ms[0].Box
	if b.Y >= 0.1 || b.Y+b.H <= 0.15 || b.X >= 0.1 || b.X+b.W <= 0.65 {
		t.Fatalf("%+v", b)
	}
	// A gap makes two.
	s = Empty().With(true, Key{Text, 0, 0}, Key{Text, 0, 2})
	if ms := d.Masks(s); len(ms) != 2 {
		t.Fatalf("%+v", ms)
	}
}

func TestStateIsImmutable(t *testing.T) {
	a := Empty()
	b := a.With(true, Key{Face, 0, 0})
	c := b.AddRect(Box{X: 0.1, Y: 0.1, W: 0.1, H: 0.1})
	if !a.IsEmpty() || !b.Has(Key{Face, 0, 0}) || len(b.rects) != 0 || len(c.rects) != 1 {
		t.Fatalf("%+v %+v %+v", a, b, c)
	}
	d := testDoc()
	ms := d.Masks(c)
	if len(ms) != 2 {
		t.Fatalf("%+v", ms)
	}
	e := c
	for _, m := range ms {
		e = e.Without(m)
	}
	if !e.IsEmpty() {
		t.Fatalf("%+v", e)
	}
}

func TestToggleAndHasAll(t *testing.T) {
	k := Key{Text, 0, 1}
	s := Empty().Toggle(k)
	if !s.Has(k) || !s.HasAll([]Key{k}) || s.HasAll(nil) || s.HasAll([]Key{k, {Text, 0, 0}}) {
		t.Fatal("toggled on")
	}
	if s.Toggle(k).Has(k) {
		t.Fatal("toggled off")
	}
}

func TestRotatedLineKeepsWordHeight(t *testing.T) {
	d := New(vision.Result{Width: 100, Height: 100, Lines: []vision.Line{{
		Text: "a b", Box: Box{X: 0, Y: 0, W: 1, H: 0.5},
		Words: []vision.Word{{Text: "a", Box: Box{X: 0, Y: 0, W: 0.1, H: 0.05}, Start: 0, End: 1}},
	}}})
	if b := d.wordBox(0, 0); b.H != 0.05 {
		t.Fatalf("%+v", b)
	}
}

func TestPixelsCoverPartlyCovered(t *testing.T) {
	x, y, w, h := Pixels(Box{X: 0.1015, Y: 0.2, W: 0.2, H: 0.1005}, 1000, 1000)
	if x != 101 || y != 200 || w != 201 || h != 101 {
		t.Fatalf("%d %d %d %d", x, y, w, h)
	}
	x, y, w, h = Pixels(Box{X: 0.9, Y: 0.9, W: 0.5, H: 0.5}, 100, 100)
	if x != 90 || y != 90 || w != 10 || h != 10 {
		t.Fatalf("%d %d %d %d", x, y, w, h)
	}
}

func TestScreenRoundTrip(t *testing.T) {
	s := Screen{CW: 400, CH: 600, W: 300, H: 500, Scale: 2.5, TX: 30, TY: -20}
	cx, cy := s.ToCanvas(0.25, 0.75)
	fx, fy := s.ToImage(cx, cy)
	if math.Abs(float64(fx-0.25)) > 1e-5 || math.Abs(float64(fy-0.75)) > 1e-5 {
		t.Fatalf("%v %v", fx, fy)
	}
	// At scale 1 and no offset, the image is centered.
	s = Screen{CW: 400, CH: 600, W: 300, H: 500, Scale: 1}
	if x, y := s.ToCanvas(0, 0); x != 50 || y != 50 {
		t.Fatalf("%v %v", x, y)
	}
}

func TestClamped(t *testing.T) {
	s := Screen{CW: 400, CH: 600, W: 300, H: 500, Scale: 1, TX: 99, TY: 99}.Clamped()
	if s.TX != 0 || s.TY != 0 {
		t.Fatalf("%+v", s)
	}
	s = Screen{CW: 400, CH: 600, W: 300, H: 500, Scale: 2, TX: 9999, TY: -9999}.Clamped()
	if s.TX != 100 || s.TY != -200 {
		t.Fatalf("%+v", s)
	}
}

func TestFit(t *testing.T) {
	if w, h := Fit(1000, 2000, 400, 600); w != 300 || h != 600 {
		t.Fatalf("%v %v", w, h)
	}
	if w, h := Fit(2000, 1000, 400, 600); w != 400 || h != 200 {
		t.Fatalf("%v %v", w, h)
	}
}
