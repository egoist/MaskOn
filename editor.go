package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/egoist/maskon/detect"
	"github.com/egoist/maskon/mask"
	"github.com/egoist/ryui/imageedit"
	"github.com/egoist/ryui/library"
	"github.com/egoist/ryui/photos"
	"github.com/egoist/ryui/signal"
	"github.com/egoist/ryui/store"
	"github.com/egoist/ryui/ui"
	"github.com/egoist/ryui/vision"
)

// Source is the picture to edit: a photo of the library, or a file.
type Source struct {
	AssetID string
	Path    string
	Name    string
}

// What masks look like.
const (
	styleGreen = iota
	styleBlack
	styleWhite
	styleMosaic
)

var maskStyles = []struct {
	title   string
	color   ui.Color
	r, g, b uint8
}{
	{"Green", colorGreen, 0x65, 0xD3, 0x6E},
	{"Black", ui.Hex(0x000000), 0, 0, 0},
	{"White", ui.Hex(0xFFFFFF), 255, 255, 255},
	{"Mosaic", ui.Hex(0x8E8E93), 0, 0, 0},
}

// mosaicBlock is a mosaic's square for an image w pixels wide: big enough,
// next to a line of text, that the text can't be read back from it.
func mosaicBlock(w int) int { return max(w/32, 12) }

// view is where the picture is on the canvas.
type view struct{ scale, tx, ty float32 }

// Editor shows the picture with what's private found, and everything to
// cover it: a tap on a line or a find, a long press for single words, a drag
// for a box, a pinch to zoom.
func Editor(src Source, title *signal.Signal[string]) ui.Node {
	path := signal.New("")
	failure := signal.New("")
	doc := signal.NewFunc[*mask.Doc](nil, nil)
	reading := signal.New(true)
	state := signal.NewFunc(mask.Empty(), nil)
	undos := signal.NewFunc([]*mask.State(nil), nil)
	redos := signal.NewFunc([]*mask.State(nil), nil)
	maskStyle := store.Signal("maskStyle", styleGreen)
	mosaic := signal.New("") // the whole picture as a mosaic, for previews
	toast := signal.New("")
	busy := signal.New(false)
	var share ui.Node

	change := func(next *mask.State) {
		ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.28}, func() {
			undos.Set(append(slices.Clone(undos.Peek()), state.Peek()))
			redos.Set(nil)
			state.Set(next)
		})
	}
	showToast := func(text string) {
		ui.Announce(text)
		ui.Animate(func() { toast.Set(text) })
		go func() {
			time.Sleep(1800 * time.Millisecond)
			ui.Post(func() { ui.Animate(func() { toast.Set("") }) })
		}()
	}

	// Copy it out of the library, find its size, show it, then read it.
	go func() {
		p, name := src.Path, src.Name
		if src.AssetID != "" {
			f, err := photos.Export(src.AssetID)
			if err != nil {
				ui.Post(func() { failure.Set(t("Couldn't read this image")) })
				return
			}
			p, name = f.Path, f.Name
		}
		// A picked photo's file is a temporary copy: its name means nothing.
		if name == "" {
			name = t("Photo") + filepath.Ext(p)
		}
		w, h, err := imageedit.Size(p)
		if err != nil {
			ui.Post(func() { failure.Set(t("Couldn't read this image")) })
			return
		}
		ui.Post(func() {
			title.Set(name)
			doc.Set(mask.New(vision.Result{Width: w, Height: h}))
			path.Set(p)
		})
		r, err := analyze(p)
		ui.Post(func() {
			reading.Set(false)
			if err != nil {
				fmt.Println("analyze:", err)
				return
			}
			r.Width, r.Height = w, h
			doc.Set(mask.New(r))
		})
	}()
	signal.OnCleanup(func() {
		if src.AssetID != "" {
			if p := path.Peek(); p != "" {
				os.Remove(p)
			}
		}
	})

	// The mosaic, the first time it's wanted: only its squares matter, so a
	// small copy will do.
	signal.Effect(func() {
		if maskStyle.Get() != styleMosaic || path.Get() == "" {
			return
		}
		signal.Untrack(func() bool {
			if mosaic.Peek() != "" {
				return true
			}
			p, d := path.Peek(), doc.Peek()
			go func() {
				w := min(d.Width, 900)
				block := max(mosaicBlock(d.Width)*w/d.Width, 1)
				h := d.Height * w / d.Width
				r, err := imageedit.Edit(p, []imageedit.Action{imageedit.Resize(w, 0), imageedit.Pixelate(0, 0, w, h, block)}, imageedit.Options{Quality: 0.8})
				if err == nil {
					ui.Post(func() { mosaic.Set(r.Path) })
				}
			}()
			return true
		})
	})
	signal.OnCleanup(func() {
		if m := mosaic.Peek(); m != "" {
			os.Remove(m)
		}
	})

	// export writes the masked picture. Being written anew, it has none of
	// the original's metadata: no location, no camera.
	export := func(done func(file string, err error)) {
		d := doc.Peek()
		if d == nil || busy.Peek() {
			return
		}
		busy.Set(true)
		masks := d.Masks(state.Peek())
		p, st, name := path.Peek(), maskStyles[maskStyle.Peek()], title.Peek()
		mosaicOn := maskStyle.Peek() == styleMosaic
		go func() {
			actions := make([]imageedit.Action, 0, len(masks))
			for _, m := range masks {
				x, y, w, h := mask.Pixels(m.Box, d.Width, d.Height)
				if mosaicOn {
					actions = append(actions, imageedit.Pixelate(x, y, w, h, mosaicBlock(d.Width)))
				} else {
					actions = append(actions, imageedit.Fill(x, y, w, h, st.r, st.g, st.b))
				}
			}
			opts := imageedit.Options{Quality: 0.92}
			ext := ".jpg"
			if strings.EqualFold(filepath.Ext(p), ".png") {
				opts.Format = imageedit.PNG
				ext = ".png"
			}
			r, err := imageedit.Edit(p, actions, opts)
			file := r.Path
			if err == nil {
				// Named after the original, for whoever receives it.
				dir, derr := os.MkdirTemp(filepath.Dir(r.Path), "export")
				if derr == nil {
					named := filepath.Join(dir, strings.TrimSuffix(name, filepath.Ext(name))+"-masked"+ext)
					if os.Rename(r.Path, named) == nil {
						file = named
					}
				}
			}
			ui.Post(func() {
				busy.Set(false)
				done(file, err)
			})
		}()
	}
	save := func() {
		export(func(file string, err error) {
			if err != nil {
				showToast(tf("Couldn't save: %s", err.Error()))
				return
			}
			library.Save(file, library.Options{Collection: library.Pictures, Move: true}, func(_ string, err error) {
				if err != nil {
					showToast(tf("Couldn't save: %s", err.Error()))
					return
				}
				ui.Haptic(ui.HapticSuccess)
				showToast(t("Saved to Photos"))
			})
		})
	}
	sharePicture := func() {
		export(func(file string, err error) {
			if err != nil {
				showToast(err.Error())
				return
			}
			ui.ShareFiles([]string{file}, ui.ShareOptions{From: share}, func(error) { os.Remove(file) })
		})
	}

	unmasked := signal.Func(func() []mask.Item {
		d := doc.Get()
		if d == nil {
			return nil
		}
		s := state.Get()
		var out []mask.Item
		for _, it := range d.Items {
			if !s.HasAll(it.Keys) {
				out = append(out, it)
			}
		}
		return out
	})
	maskAll := func() {
		ui.Haptic(ui.HapticSuccess)
		change(state.Peek().With(true, mask.AllKeys(unmasked.Get())...))
	}

	return ui.View(ui.ViewProps{Style: ui.Style{Grow: 1, Background: colorBG}},
		dots(),
		ui.SafeArea(ui.Style{Grow: 1},
			ui.Show(signal.Func(func() bool { return failure.Get() != "" }), func() ui.Node {
				return ui.Column(ui.Style{Grow: 1, Align: ui.AlignCenter, Justify: ui.JustifyCenter},
					ui.Text(failure, ui.Style{Color: colorDim}))
			}, func() ui.Node {
				return ui.Show(signal.Func(func() bool { return path.Get() == "" }), func() ui.Node {
					return ui.Column(ui.Style{Grow: 1, Align: ui.AlignCenter, Justify: ui.JustifyCenter}, ui.Activity())
				}, func() ui.Node {
					return canvas(doc, path, state, change, maskStyle, mosaic, reading)
				})
			}),
			toolbar(toolbarProps{
				doc: doc, state: state, change: change, undos: undos, redos: redos, maskStyle: maskStyle, reading: reading,
				unmasked: unmasked, maskAll: maskAll, save: save, share: sharePicture, busy: busy, shareNode: &share,
			}),
		),
		ui.Show(signal.Func(func() bool { return toast.Get() != "" }), func() ui.Node {
			return ui.View(ui.ViewProps{Style: ui.Style{
				Absolute: true, Top: ui.Pt(16), AlignSelf: ui.AlignCenter, Padding: ui.Symmetric(10, 18), CornerRadius: 20,
				Background: colorRaised, BorderWidth: 1, BorderColor: colorBorder, Transition: ui.SlideUp,
				Shadow: ui.Shadow{Color: ui.RGBA(0, 0, 0, 120), Radius: 12, Y: 4},
			}}, ui.Text(toast, ui.Style{Font: ui.Font{Size: 15, Weight: ui.Medium}, Color: colorText}))
		}),
	)
}

// canvas is the picture and everything on it. Gestures are on the canvas,
// which doesn't move, so where they are needs no undoing of the zoom's
// transform but the arithmetic in mask.Screen.
func canvas(doc *signal.Signal[*mask.Doc], path signal.Accessor[string], state *signal.Signal[*mask.State], change func(*mask.State),
	maskStyle signal.Accessor[int], mosaic signal.Accessor[string], reading signal.Accessor[bool]) ui.Node {

	size := signal.New(ui.Frame{})
	v := signal.New(view{scale: 1})
	draft := signal.New(mask.Box{})
	openLine := signal.New(-1) // the line whose words are shown

	screen := func(cur view) mask.Screen {
		f := size.Peek()
		d := doc.Peek()
		w, h := mask.Fit(d.Width, d.Height, f.Width, f.Height)
		return mask.Screen{CW: f.Width, CH: f.Height, W: w, H: h, Scale: cur.scale, TX: cur.tx, TY: cur.ty}
	}

	// A second finger makes it a pinch: the drag it began is dropped.
	panning, voided := false, false
	onPan := func(g ui.Gesture) {
		if openLine.Peek() >= 0 {
			return
		}
		switch g.State {
		case ui.GestureBegan:
			panning = true
		case ui.GestureEnded, ui.GestureCancelled:
			wasVoided := voided
			panning, voided = false, false
			box := draft.Peek()
			draft.Set(mask.Box{})
			if g.State == ui.GestureEnded && !wasVoided && box.W > 0 {
				s := screen(v.Peek())
				// A box too small to mean anything was a slip of the finger.
				if float32(box.W)*s.W*s.Scale > 12 && float32(box.H)*s.H*s.Scale > 8 {
					ui.Haptic(ui.HapticLight)
					change(state.Peek().AddRect(box))
				}
			}
			return
		}
		if voided {
			return
		}
		s := screen(v.Peek())
		ax, ay := s.ToImage(g.X-g.TranslationX, g.Y-g.TranslationY)
		bx, by := s.ToImage(g.X, g.Y)
		x0, y0, x1, y1 := min(ax, bx), min(ay, by), max(ax, bx), max(ay, by)
		x0, y0, x1, y1 = max(x0, 0), max(y0, 0), min(x1, 1), min(y1, 1)
		if x1 > x0 && y1 > y0 {
			draft.Set(mask.Box{X: float64(x0), Y: float64(y0), W: float64(x1 - x0), H: float64(y1 - y0)})
		} else {
			draft.Set(mask.Box{})
		}
	}
	// The picture stays where the fingers hold it, as they spread and move.
	var pinchStart view
	var focusX, focusY float32
	onPinch := func(g ui.Gesture) {
		openLine.Set(-1)
		switch g.State {
		case ui.GestureBegan:
			voided = true
			draft.Set(mask.Box{})
			pinchStart = v.Peek()
			focusX, focusY = g.X, g.Y
		case ui.GestureChanged:
			f := size.Peek()
			sc := min(max(pinchStart.scale*g.Scale, 1), 12)
			r := sc / pinchStart.scale
			s := screen(view{
				scale: sc,
				tx:    g.X - f.Width/2 - r*(focusX-f.Width/2-pinchStart.tx),
				ty:    g.Y - f.Height/2 - r*(focusY-f.Height/2-pinchStart.ty),
			}).Clamped()
			v.Set(view{s.Scale, s.TX, s.TY})
		case ui.GestureEnded, ui.GestureCancelled:
			if !panning {
				voided = false
			}
			if cur := v.Peek(); cur.scale < 1.05 {
				ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.3}, func() { v.Set(view{scale: 1}) })
			}
		}
	}

	return ui.View(ui.ViewProps{
		OnLayout: func(f ui.Frame) { size.Set(f) },
		OnPan:    onPan,
		OnPinch:  onPinch,
		Style:    ui.Style{Grow: 1, Clip: true},
	},
		ui.View(ui.ViewProps{Style: ui.StyleFunc(func() ui.Style {
			f := size.Get()
			d := doc.Get()
			cur := v.Get()
			w, h := mask.Fit(d.Width, d.Height, f.Width, f.Height)
			return ui.Style{
				Absolute: true, Left: ui.Pt((f.Width - w) / 2), Top: ui.Pt((f.Height - h) / 2), Width: ui.Pt(w), Height: ui.Pt(h),
				Transform: ui.Transform{Scale: cur.scale, TranslateX: cur.tx, TranslateY: cur.ty},
			}
		})},
			ui.Image(ui.ImageProps{Source: path, Mode: ui.ImageStretch, Label: t("Picture"), Style: fill()}),
			ui.Dynamic(func() ui.Node {
				d := doc.Get()
				if d == nil {
					return ui.Node{}
				}
				return signal.Untrack(func() ui.Node { return targets(d, state, change, openLine) })
			}),
			ui.For(signal.Func(func() []mask.Mask {
				if d := doc.Get(); d != nil {
					return d.Masks(state.Get())
				}
				return nil
			}), func(m mask.Mask) string { return m.ID }, func(m signal.Accessor[mask.Mask], _ signal.Accessor[int]) ui.Node {
				return maskView(m, state, change, openLine, maskStyle, mosaic)
			}),
			ui.Show(signal.Func(func() bool { return draft.Get().W > 0 }), func() ui.Node {
				return ui.View(ui.ViewProps{Decorative: true, Style: ui.StyleFunc(func() ui.Style {
					s := boxStyle(draft.Get())
					s.Background = colorGreenBG
					s.BorderWidth = 1.5
					s.BorderColor = colorGreen
					return s
				})})
			}),
		),
		ui.Show(reading, func() ui.Node {
			return ui.View(ui.ViewProps{Style: ui.Style{
				Absolute: true, Bottom: ui.Pt(12), AlignSelf: ui.AlignCenter, Direction: ui.RowDir, Gap: 8, Align: ui.AlignCenter,
				Padding: ui.Symmetric(8, 14), CornerRadius: 18, Background: colorRaised, Transition: ui.Fade,
			}},
				ui.Activity(),
				ui.StaticText(t("Finding what's private…"), ui.Style{Font: ui.Font{Size: 14}, Color: colorDim}),
			)
		}),
		ui.Show(signal.Func(func() bool { return openLine.Get() >= 0 }), func() ui.Node {
			return wordPicker(doc, state, change, openLine, func() mask.Screen { return screen(v.Peek()) })
		}),
	)
}

// boxStyle places a view over a box of the picture.
func boxStyle(b mask.Box) ui.Style {
	return ui.Style{
		Absolute: true, Left: ui.Pct(float32(b.X * 100)), Top: ui.Pct(float32(b.Y * 100)),
		Width: ui.Pct(float32(b.W * 100)), Height: ui.Pct(float32(b.H * 100)),
	}
}

// targets are what can be tapped on the picture: each line of text, and over
// them what was found to be private, marked until it's covered.
func targets(d *mask.Doc, state *signal.Signal[*mask.State], change func(*mask.State), openLine *signal.Signal[int]) ui.Node {
	var kids []ui.Node
	for i, l := range d.Lines {
		if len(l.Words) == 0 {
			continue
		}
		kids = append(kids, ui.View(ui.ViewProps{
			OnPress: func() {
				ui.Haptic(ui.HapticLight)
				change(state.Peek().With(true, d.LineKeys(i)...))
			},
			OnLongPress: func() {
				ui.Haptic(ui.HapticMedium)
				ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.3}, func() { openLine.Set(i) })
			},
			Label: l.Text,
			Hint:  t("Tap to mask"),
			Style: boxStyle(d.LineBox(i)),
		}))
	}
	for _, it := range d.Items {
		kids = append(kids, ui.Show(signal.Func(func() bool { return !state.Get().HasAll(it.Keys) }), func() ui.Node {
			s := boxStyle(it.Box)
			s.Background = colorGreenBG
			s.BorderWidth = 1.5
			s.BorderColor = colorGreen
			s.CornerRadius = 3
			s.Transition = ui.Fade
			return ui.View(ui.ViewProps{
				OnPress: func() {
					ui.Haptic(ui.HapticLight)
					change(state.Peek().With(true, it.Keys...))
				},
				OnLongPress: func() {
					if it.Line >= 0 {
						ui.Haptic(ui.HapticMedium)
						ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.3}, func() { openLine.Set(it.Line) })
					}
				},
				Label: itemLabel(d, it),
				Hint:  t("Tap to mask"),
				Style: s,
			})
		}))
	}
	return ui.Column(fill(), kids...)
}

func itemTitle(it mask.Item) string {
	switch it.Cat {
	case mask.Face:
		return t("Faces")
	case mask.Code:
		return t("Codes")
	}
	return kindTitle(it.Kind)
}

func itemLabel(d *mask.Doc, it mask.Item) string {
	if it.Line >= 0 {
		return d.Lines[it.Line].Text
	}
	return itemTitle(it)
}

// maskView is a mask: a tap takes it away, a long press on text opens its
// words.
func maskView(m signal.Accessor[mask.Mask], state *signal.Signal[*mask.State], change func(*mask.State), openLine *signal.Signal[int],
	maskStyle signal.Accessor[int], mosaic signal.Accessor[string]) ui.Node {
	cur := func() mask.Mask { return signal.Untrack(m.Get) }
	return ui.View(ui.ViewProps{
		OnPress: func() {
			ui.Haptic(ui.HapticLight)
			change(state.Peek().Without(cur()))
		},
		OnLongPress: func() {
			if l := cur().Line; l >= 0 {
				ui.Haptic(ui.HapticMedium)
				ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.3}, func() { openLine.Set(l) })
			}
		},
		Label: t("Masked. Tap to remove."),
		Style: ui.StyleFunc(func() ui.Style {
			s := boxStyle(m.Get().Box)
			s.Clip = true
			s.CornerRadius = 2
			s.Transition = ui.Transition{Fade: true, Scale: 0.92}
			st := maskStyle.Get()
			s.Background = maskStyles[st].color
			return s
		}),
	},
		ui.Show(signal.Func(func() bool { return maskStyle.Get() == styleMosaic && mosaic.Get() != "" }), func() ui.Node {
			// The whole mosaic, sized and moved so its right part shows here.
			return ui.Image(ui.ImageProps{Source: mosaic, Mode: ui.ImageStretch, Style: ui.StyleFunc(func() ui.Style {
				b := m.Get().Box
				return ui.Style{
					Absolute: true, Left: ui.Pct(float32(-b.X / b.W * 100)), Top: ui.Pct(float32(-b.Y / b.H * 100)),
					Width: ui.Pct(float32(100 / b.W)), Height: ui.Pct(float32(100 / b.H)),
				}
			})})
		}),
	)
}

// wordPicker is the popover of a line's words, pointing at the line.
func wordPicker(doc *signal.Signal[*mask.Doc], state *signal.Signal[*mask.State], change func(*mask.State), openLine *signal.Signal[int],
	screen func() mask.Screen) ui.Node {

	d := doc.Peek()
	line := openLine.Peek()
	l := d.Lines[line]
	s := screen()
	box := d.LineBox(line)
	_, top := s.ToCanvas(float32(box.X), float32(box.Y))
	x, bottom := s.ToCanvas(float32(box.X+box.W/2), float32(box.Y+box.H))
	// Above the line if it fits there, else below.
	above := top > s.CH*0.45
	const margin, arrow = float32(12), float32(11)

	chips := make([]ui.Node, len(l.Words))
	for i, w := range l.Words {
		k := mask.Key{Cat: mask.Text, A: line, B: i}
		on := signal.Func(func() bool { return state.Get().Has(k) })
		chips[i] = ui.View(ui.ViewProps{
			OnPress: func() {
				ui.Haptic(ui.HapticSelection)
				change(state.Peek().Toggle(k))
			},
			Label: w.Text,
			Role:  ui.RoleButton,
			Style: ui.StyleFunc(func() ui.Style {
				s := ui.Style{Padding: ui.Symmetric(8, 12), CornerRadius: 10, Background: colorChip}
				if on.Get() {
					s.Background = colorGreen
				}
				return s
			}),
		}, ui.StaticText(w.Text, ui.StyleFunc(func() ui.Style {
			s := ui.Style{Font: ui.Font{Size: 16}, Color: colorGreenHi}
			if on.Get() {
				s.Color = colorOnGreen
			}
			return s
		})))
	}
	panel := ui.Style{
		Absolute: true, Left: ui.Pt(margin), Right: ui.Pt(margin), Transition: ui.Pop,
	}
	if above {
		panel.Bottom = ui.Pt(s.CH - top + arrow + 2)
	} else {
		panel.Top = ui.Pt(bottom + arrow + 2)
	}
	arrowX := min(max(x-margin-11, 14), s.CW-2*margin-36)
	arrowNode := ui.Shape(ui.ShapeProps{
		Path: func(w, h float32) *ui.Path {
			if above {
				return new(ui.Path).MoveTo(0, 0).LineTo(w, 0).LineTo(w/2, h).Close()
			}
			return new(ui.Path).MoveTo(0, h).LineTo(w, h).LineTo(w/2, 0).Close()
		},
		Fill:  colorBG,
		Style: ui.Style{Width: ui.Pt(22), Height: ui.Pt(arrow), Margin: ui.Edges{Left: arrowX}},
	})
	body := ui.View(ui.ViewProps{Style: ui.Style{
		CornerRadius: 16, Background: colorBG, BorderWidth: 1, BorderColor: colorBorder,
		Shadow: ui.Shadow{Color: ui.RGBA(0, 0, 0, 170), Radius: 18, Y: 6},
	}}, ui.Scroll(ui.ScrollProps{Style: ui.Style{MaxHeight: ui.Pt(s.CH * 0.42)}, ContentStyle: ui.Style{Padding: ui.All(12)}},
		ui.Row(ui.Style{Wrap: true, Gap: 8}, chips...)))

	var kids []ui.Node
	if above {
		kids = []ui.Node{body, arrowNode}
	} else {
		kids = []ui.Node{arrowNode, body}
	}
	return ui.View(ui.ViewProps{Style: fill()},
		// A tap anywhere else closes it.
		ui.View(ui.ViewProps{OnPress: func() { ui.Animate(func() { openLine.Set(-1) }) }, Label: t("Done"), Style: fill()}),
		ui.View(ui.ViewProps{Style: panel}, kids...),
	)
}

type toolbarProps struct {
	doc       signal.Accessor[*mask.Doc]
	state     *signal.Signal[*mask.State]
	change    func(*mask.State)
	undos     *signal.Signal[[]*mask.State]
	redos     *signal.Signal[[]*mask.State]
	maskStyle *signal.Signal[int]
	reading   signal.Accessor[bool]
	unmasked  signal.Accessor[[]mask.Item]
	maskAll   func()
	save      func()
	share     func()
	busy      signal.Accessor[bool]
	shareNode *ui.Node
}

func toolbar(p toolbarProps) ui.Node {
	undo := func() {
		us := p.undos.Peek()
		if len(us) == 0 {
			return
		}
		ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.28}, func() {
			p.redos.Set(append(slices.Clone(p.redos.Peek()), p.state.Peek()))
			p.state.Set(us[len(us)-1])
			p.undos.Set(us[:len(us)-1])
		})
	}
	redo := func() {
		rs := p.redos.Peek()
		if len(rs) == 0 {
			return
		}
		ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.28}, func() {
			p.undos.Set(append(slices.Clone(p.undos.Peek()), p.state.Peek()))
			p.state.Set(rs[len(rs)-1])
			p.redos.Set(rs[:len(rs)-1])
		})
	}
	// Nothing leaves before the picture has been read: what's private in it
	// isn't known until then, and a save would look like a safe one.
	ready := signal.Func(func() bool { return !p.busy.Get() && !p.reading.Get() && p.doc.Get() != nil })
	*p.shareNode = toolButton(ui.IconShare, t("Share"), ready, p.share)
	swatches := make([]ui.Node, len(maskStyles))
	for i, st := range maskStyles {
		swatches[i] = ui.View(ui.ViewProps{
			OnPress: func() {
				ui.Haptic(ui.HapticSelection)
				p.maskStyle.Set(i)
			},
			Label: t(st.title),
			Role:  ui.RoleButton,
			Style: ui.StyleFunc(func() ui.Style {
				s := ui.Style{Width: ui.Pt(30), Height: ui.Pt(30), CornerRadius: 15, Align: ui.AlignCenter, Justify: ui.JustifyCenter, BorderWidth: 2, BorderColor: ui.Clear}
				if p.maskStyle.Get() == i {
					s.BorderColor = colorText
				}
				return s
			}),
		}, swatch(i))
	}
	return ui.Column(ui.Style{Gap: 12, Padding: ui.Edges{Top: 10, Left: 16, Right: 16, Bottom: 6}, Background: colorBG},
		chipsRow(p),
		ui.Row(ui.Style{Align: ui.AlignCenter, Justify: ui.JustifySpaceBetween},
			ui.Row(ui.Style{Gap: 4},
				toolButton(ui.Icon("arrow.uturn.backward"), t("Undo"), signal.Func(func() bool { return len(p.undos.Get()) > 0 }), undo),
				toolButton(ui.Icon("arrow.uturn.forward"), t("Redo"), signal.Func(func() bool { return len(p.redos.Get()) > 0 }), redo),
			),
			ui.Row(ui.Style{Gap: 6, Align: ui.AlignCenter}, swatches...),
			*p.shareNode,
		),
		capsule(signal.Func(func() string {
			if n := len(p.unmasked.Get()); n > 0 {
				return tf("Mask All (%d)", n)
			}
			return t("Save")
		}), ready, func() {
			if len(signal.Untrack(p.unmasked.Get)) > 0 {
				p.maskAll()
			} else {
				p.save()
			}
		}),
	)
}

func swatch(style int) ui.Node {
	if style != styleMosaic {
		return ui.View(ui.ViewProps{Decorative: true, Style: ui.Style{
			Width: ui.Pt(22), Height: ui.Pt(22), CornerRadius: 11, Background: maskStyles[style].color,
			BorderWidth: 1, BorderColor: ui.RGBA(255, 255, 255, 60),
		}})
	}
	// A little checkerboard.
	return ui.View(ui.ViewProps{Decorative: true, Style: ui.Style{Width: ui.Pt(22), Height: ui.Pt(22), CornerRadius: 11, Clip: true, Background: ui.Hex(0x48484A)}},
		ui.Shape(ui.ShapeProps{
			Path: func(w, h float32) *ui.Path {
				p := new(ui.Path)
				c := w / 4
				for y := range 4 {
					for x := range 4 {
						if (x+y)%2 == 0 {
							p.Rect(float32(x)*c, float32(y)*c, c, c)
						}
					}
				}
				return p
			},
			Fill:  ui.Hex(0xAEAEB2),
			Style: fill(),
		}),
	)
}

func toolButton(i ui.Icon, label string, enabled signal.Accessor[bool], press func()) ui.Node {
	return ui.View(ui.ViewProps{
		OnPress: func() {
			if enabled.Get() {
				press()
			}
		},
		Label: label,
		Role:  ui.RoleButton,
		Style: ui.StyleFunc(func() ui.Style {
			s := ui.Style{Width: ui.Pt(42), Height: ui.Pt(42), CornerRadius: 21, Align: ui.AlignCenter, Justify: ui.JustifyCenter, Background: colorRaised,
				Pressed: &ui.Style{Transform: ui.Transform{Scale: 0.92}}}
			if !enabled.Get() {
				s.Opacity = 0.4
			}
			return s
		}),
	}, ui.Symbol(i, ui.Style{Color: colorText, Font: ui.Font{Size: 17}}))
}

// chipsRow toggles a whole kind of what was found at once.
func chipsRow(p toolbarProps) ui.Node {
	type chip struct {
		title string
		cat   mask.Category
		kind  detect.Kind
		items []mask.Item
	}
	chips := signal.Func(func() []chip {
		d := p.doc.Get()
		if d == nil {
			return nil
		}
		var out []chip
		for _, it := range d.Items {
			i := slices.IndexFunc(out, func(c chip) bool {
				return c.cat == it.Cat && (it.Cat != mask.Text || c.kind == it.Kind)
			})
			if i < 0 {
				out = append(out, chip{title: itemTitle(it), cat: it.Cat, kind: it.Kind})
				i = len(out) - 1
			}
			out[i].items = append(out[i].items, it)
		}
		return out
	})
	// Its height is kept while there's nothing in it, so the picture doesn't
	// jump when the first finds come.
	return ui.Column(ui.Style{MinHeight: ui.Pt(34)}, ui.Show(signal.Func(func() bool { return len(chips.Get()) > 0 }), func() ui.Node {
		return ui.Scroll(ui.ScrollProps{Horizontal: true, ContentStyle: ui.Style{Gap: 8}},
			ui.For(chips, func(c chip) string { return fmt.Sprint(c.cat, "/", c.kind) }, func(c signal.Accessor[chip], _ signal.Accessor[int]) ui.Node {
				on := signal.Func(func() bool { return p.state.Get().HasAll(mask.AllKeys(c.Get().items)) })
				return ui.View(ui.ViewProps{
					OnPress: func() {
						ui.Haptic(ui.HapticSelection)
						keys := mask.AllKeys(signal.Untrack(c.Get).items)
						p.change(p.state.Peek().With(!p.state.Peek().HasAll(keys), keys...))
					},
					Label: signal.Untrack(c.Get).title,
					Role:  ui.RoleButton,
					Style: ui.StyleFunc(func() ui.Style {
						s := ui.Style{Padding: ui.Symmetric(7, 13), CornerRadius: 16, Background: colorRaised}
						if on.Get() {
							s.Background = colorGreen
						}
						return s
					}),
				}, ui.Text(signal.Func(func() string {
					return fmt.Sprintf("%s %d", c.Get().title, len(c.Get().items))
				}), ui.StyleFunc(func() ui.Style {
					s := ui.Style{Font: ui.Font{Size: 14, Weight: ui.Medium}, Color: colorDim}
					if on.Get() {
						s.Color = colorOnGreen
					}
					return s
				})))
			}),
		)
	}))
}
