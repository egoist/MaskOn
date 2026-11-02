package main

import (
	"github.com/egoist/ryui/signal"
	"github.com/egoist/ryui/ui"
)

// The sample everything's tried on.
var sampleAddress = []string{"742", "Evergreen", "Ter", ".", "Portland", ",", "Oregon", "97205"}

// Onboarding is four steps, each tried on a sample: every step's button
// waits for the step to be done, though "Next" skips it.
func Onboarding(done func()) ui.Node {
	step := signal.New(0)
	finished := signal.New(false) // this step's task is done
	next := func() {
		if step.Peek() == len(steps)-1 {
			done()
			return
		}
		ui.Animate(func() {
			finished.Set(false)
			step.Set(step.Peek() + 1)
		})
	}
	return ui.View(ui.ViewProps{Style: ui.Style{Grow: 1, Background: colorBG}},
		dots(),
		ui.SafeArea(ui.Style{Grow: 1, Padding: ui.Edges{Top: 36, Left: 22, Right: 22, Bottom: 8}},
			// Rebuilt for each step: its state goes with it.
			ui.Column(ui.Style{Grow: 1},
				ui.Dynamic(func() ui.Node {
					s := steps[step.Get()]
					return ui.Column(ui.Style{Grow: 1, Transition: ui.Fade},
						stepHeader(s),
						s.body(finished),
					)
				}),
			),
			pager(step),
			capsule(signal.Func(func() string {
				if step.Get() == len(steps)-1 {
					return t("Get Started")
				}
				return t("Continue")
			}), finished, next),
			ui.View(ui.ViewProps{
				OnPress: next,
				Label:   t("Next"),
				Role:    ui.RoleButton,
				Style:   ui.Style{AlignSelf: ui.AlignCenter, Padding: ui.Symmetric(14, 24)},
			}, ui.StaticText(t("Next"), ui.Style{Font: ui.Font{Size: 16}, Color: colorDim})),
		),
	)
}

type onboardingStep struct {
	title, subtitle string
	body            func(finished *signal.Signal[bool]) ui.Node
}

var steps = []onboardingStep{
	{"Auto Mask", "Maskon finds QR codes, barcodes, phone numbers and more, and covers them in one tap.", autoStep},
	{"Tap to Mask", "Spot something it missed? Tap it, and it's covered.", tapStep},
	{"Mask a Single Word", "Press and hold text to pick exactly which words to cover.", wordStep},
	{"Mask Any Area", "Drag across anything, and a box covers it.", dragStep},
}

func stepHeader(s onboardingStep) ui.Node {
	return ui.Column(ui.Style{Align: ui.AlignCenter, Gap: 12, MinHeight: ui.Pt(150)},
		ui.StaticText(t(s.title), ui.Style{Font: ui.Font{Size: 30, Weight: ui.Bold}, Color: colorText, TextAlign: ui.TextCenter}),
		ui.StaticText(t(s.subtitle), ui.Style{Font: ui.Font{Size: 17}, Color: colorDim, TextAlign: ui.TextCenter}),
	)
}

func pager(step *signal.Signal[int]) ui.Node {
	kids := make([]ui.Node, len(steps))
	for i := range steps {
		kids[i] = ui.View(ui.ViewProps{Decorative: true, Style: ui.StyleFunc(func() ui.Style {
			s := ui.Style{Width: ui.Pt(8), Height: ui.Pt(8), CornerRadius: 4, Background: ui.RGBA(255, 255, 255, 60)}
			if step.Get() == i {
				s.Width = ui.Pt(22)
				s.Background = colorGreen
			}
			return s
		})})
	}
	return ui.Row(ui.Style{Gap: 7, Justify: ui.JustifyCenter, Padding: ui.Symmetric(18, 0)}, kids...)
}

// cover is a green, or for what's already done a dim, box over a child that
// the box hides: shown while on is true.
func cover(on signal.Accessor[bool], color ui.Color, child ui.Node) ui.Node {
	return ui.View(ui.ViewProps{Style: ui.Style{}},
		child,
		ui.Show(on, func() ui.Node {
			// A point larger all round, so nothing peeks past its round corners.
			s := ui.Style{Absolute: true, Left: ui.Pt(-1), Top: ui.Pt(-1), Right: ui.Pt(-1), Bottom: ui.Pt(-1)}
			s.Background = color
			s.CornerRadius = 4
			s.Transition = ui.Pop
			return ui.View(ui.ViewProps{Style: s})
		}),
	)
}

// sampleCard is the card the steps try things on. dim hides its private
// parts in gray, for the steps after the first; on says they're covered.
type cardParts struct {
	name    ui.Node // the name line, ready-made for the step
	address ui.Node
	covered signal.Accessor[bool]
	color   ui.Color
	extra   ui.Node // above the rest: a drawn box, a popover
	onPan   func(ui.Gesture)
}

func sampleCard(p cardParts) ui.Node {
	divider := func() ui.Node {
		return ui.View(ui.ViewProps{Style: ui.Style{Height: ui.Pt(1), Background: colorBorder, Margin: ui.Edges{Top: 11, Bottom: 11}}})
	}
	field := func(label, value string, w float32) ui.Node {
		return ui.Row(ui.Style{Align: ui.AlignCenter, Justify: ui.JustifySpaceBetween, Padding: ui.Symmetric(5, 0)},
			ui.StaticText(label, ui.Style{Font: ui.Font{Size: 16}, Color: colorDim}),
			cover(p.covered, p.color, ui.View(ui.ViewProps{Style: ui.Style{Width: ui.Pt(w), Height: ui.Pt(21), Align: ui.AlignEnd, Justify: ui.JustifyCenter}},
				ui.StaticText(value, ui.Style{Font: ui.Font{Size: 14}, Color: colorText, Lines: 1}))),
		)
	}
	return ui.View(ui.ViewProps{
		OnPan: p.onPan,
		Style: ui.Style{
			Padding: ui.Edges{Top: 16, Left: 16, Right: 16, Bottom: 16}, CornerRadius: 20,
			BorderWidth: 1, BorderColor: colorBorder, Background: colorCard,
		},
	},
		ui.Row(ui.Style{Gap: 10, Align: ui.AlignCenter},
			cover(p.covered, p.color, ui.View(ui.ViewProps{Style: ui.Style{Width: ui.Pt(40), Height: ui.Pt(40), Background: colorRaised, CornerRadius: 4, Align: ui.AlignCenter, Justify: ui.JustifyCenter}},
				ui.Symbol(ui.IconPerson.Filled(), ui.Style{Color: colorDim}))),
			ui.Column(ui.Style{Grow: 1, Gap: 3},
				p.name,
				ui.StaticText("Founder", ui.Style{Font: ui.Font{Size: 14}, Color: colorDim}),
			),
		),
		divider(),
		field("Card", "4242 4242 4242 1881", 180),
		field("Phone", "(503) 555-0142", 135),
		divider(),
		ui.Row(ui.Style{Justify: ui.JustifySpaceBetween, Align: ui.AlignCenter},
			cover(p.covered, p.color, qrSample()),
			cover(p.covered, p.color, barcodeSample()),
		),
		divider(),
		ui.StaticText(t("Ship to"), ui.Style{Font: ui.Font{Size: 14}, Color: colorDim, Margin: ui.Edges{Bottom: 4}}),
		p.address,
		p.extra,
	)
}

func nameText() ui.Node {
	return ui.StaticText("Jordan Reyes", ui.Style{Font: ui.Font{Size: 17, Weight: ui.Semibold}, Color: colorText})
}

func addressText(word func(i int, w string) ui.Node) ui.Node {
	kids := make([]ui.Node, len(sampleAddress))
	for i, w := range sampleAddress {
		kids[i] = word(i, w)
	}
	return ui.Row(ui.Style{Wrap: true, Align: ui.AlignCenter, Gap: 0}, kids...)
}

// wordGap is the space before a word, none before punctuation.
func wordGap(i int, w string) float32 {
	if i == 0 || w == "." || w == "," {
		return 0
	}
	return 5
}

func plainWord(i int, w string) ui.Node {
	return ui.StaticText(w, ui.Style{Font: ui.Font{Size: 15}, Color: colorText, Margin: ui.Edges{Left: wordGap(i, w)}})
}

func qrSample() ui.Node {
	const n = 15
	return ui.Shape(ui.ShapeProps{
		Path: func(w, h float32) *ui.Path {
			p := new(ui.Path)
			c := w / n
			on := func(x, y int) bool {
				// The three squares in the corners, then something code-like.
				for _, f := range [][2]int{{0, 0}, {n - 5, 0}, {0, n - 5}} {
					dx, dy := x-f[0], y-f[1]
					if dx >= -1 && dy >= -1 && dx <= 5 && dy <= 5 {
						return dx >= 0 && dy >= 0 && dx <= 4 && dy <= 4 && (dx == 0 || dy == 0 || dx == 4 || dy == 4 || (dx == 2 && dy == 2))
					}
				}
				return (x*7+y*13+x*y)%3 == 0
			}
			for y := range n {
				for x := range n {
					if on(x, y) {
						p.Rect(float32(x)*c, float32(y)*c, c, c)
					}
				}
			}
			return p
		},
		Fill:  colorText,
		Style: ui.Style{Width: ui.Pt(72), Height: ui.Pt(72)},
	})
}

func barcodeSample() ui.Node {
	widths := []float32{2, 1, 3, 1, 1, 2, 1, 3, 2, 1, 1, 3, 1, 2, 2, 1, 3, 1, 1, 2}
	return ui.Shape(ui.ShapeProps{
		Path: func(w, h float32) *ui.Path {
			var total float32
			for _, v := range widths {
				total += v * 2
			}
			u := w / total
			p := new(ui.Path)
			var x float32
			for _, v := range widths {
				p.Rect(x, 0, v*u, h)
				x += v * 2 * u
			}
			return p
		},
		Fill:  colorText,
		Style: ui.Style{Width: ui.Pt(100), Height: ui.Pt(66)},
	})
}

// caption is the line under the card, which turns to praise once it's done.
func caption(finished signal.Accessor[bool], hint signal.Accessor[string]) ui.Node {
	return ui.Text(signal.Func(func() string {
		if finished.Get() {
			return t("Nice, just like that!")
		}
		return hint.Get()
	}), ui.Style{Font: ui.Font{Size: 17}, Color: colorDim, TextAlign: ui.TextCenter, Margin: ui.Edges{Top: 18}})
}

func filename() ui.Node {
	return ui.Row(ui.Style{Gap: 6, Justify: ui.JustifyCenter, Align: ui.AlignCenter, Margin: ui.Edges{Bottom: 10}},
		icon(imageIcon, 16, colorDim),
		ui.StaticText("IMG_050.png", ui.Style{Font: ui.Font{Size: 15}, Color: colorDim}),
	)
}

// The first step: one button covers everything.
func autoStep(finished *signal.Signal[bool]) ui.Node {
	return ui.Column(nil,
		filename(),
		sampleCard(cardParts{
			name:    nameText(),
			address: addressText(plainWord),
			covered: finished,
			color:   colorGreen,
		}),
		ui.Show(finished, func() ui.Node {
			return caption(finished, ui.Val(""))
		}, func() ui.Node {
			return ui.View(ui.ViewProps{
				OnPress: func() {
					ui.Haptic(ui.HapticSuccess)
					ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.4}, func() { finished.Set(true) })
				},
				Label: t("Mask everything"),
				Role:  ui.RoleButton,
				Style: ui.Style{
					Direction: ui.RowDir, Gap: 8, AlignSelf: ui.AlignCenter, Align: ui.AlignCenter, Margin: ui.Edges{Top: 14},
					Padding: ui.Symmetric(10, 18), CornerRadius: 22, Background: colorGreenBG,
					Pressed: &ui.Style{Transform: ui.Transform{Scale: 0.96}},
				},
			},
				icon(wandIcon, 18, colorGreen),
				ui.StaticText(t("Mask everything"), ui.Style{Font: ui.Font{Size: 16, Weight: ui.Semibold}, Color: colorGreen}),
			)
		}),
	)
}

// The second: a tap on the name covers it.
func tapStep(finished *signal.Signal[bool]) ui.Node {
	return ui.Column(nil,
		filename(),
		sampleCard(cardParts{
			name: ui.View(ui.ViewProps{
				OnPress: func() {
					ui.Haptic(ui.HapticLight)
					ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.35}, func() { finished.Set(!finished.Peek()) })
				},
				Label: "Jordan Reyes",
				Hint:  t("Tap to mask"),
				Style: ui.Style{AlignSelf: ui.AlignStart},
			},
				cover(finished, colorGreen, nameText()),
				ui.Show(signal.Func(func() bool { return !finished.Get() }), func() ui.Node {
					return ui.View(ui.ViewProps{Style: ui.Style{Absolute: true, Left: ui.Pct(70), Top: ui.Pt(12)}}, bobbingHand())
				}),
			),
			address: addressText(plainWord),
			covered: ui.Val(true),
			color:   colorRaised,
		}),
		caption(finished, ui.Val(t("Try tapping the name"))),
	)
}

// The third: a long press on the address opens its words.
func wordStep(finished *signal.Signal[bool]) ui.Node {
	open := signal.New(false)
	covered := make([]*signal.Signal[bool], len(sampleAddress))
	for i := range covered {
		covered[i] = signal.New(false)
	}
	toggle := func(i int) {
		ui.Haptic(ui.HapticSelection)
		ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.3}, func() { covered[i].Set(!covered[i].Peek()) })
		any := false
		for _, c := range covered {
			any = any || c.Peek()
		}
		finished.Set(any)
	}
	word := func(i int, w string) ui.Node {
		return ui.View(ui.ViewProps{Style: ui.Style{Margin: ui.Edges{Left: wordGap(i, w)}}},
			cover(covered[i], colorGreen, ui.StaticText(w, ui.Style{Font: ui.Font{Size: 15}, Color: colorText})))
	}
	popover := func() ui.Node {
		chips := make([]ui.Node, len(sampleAddress))
		for i, w := range sampleAddress {
			chips[i] = ui.View(ui.ViewProps{
				OnPress: func() { toggle(i) },
				Label:   w,
				Role:    ui.RoleButton,
				Style: ui.StyleFunc(func() ui.Style {
					s := ui.Style{Padding: ui.Symmetric(8, 12), CornerRadius: 10, Background: colorChip}
					if covered[i].Get() {
						s.Background = colorGreen
					}
					return s
				}),
			}, ui.StaticText(w, ui.StyleFunc(func() ui.Style {
				s := ui.Style{Font: ui.Font{Size: 16}, Color: colorGreenHi}
				if covered[i].Get() {
					s.Color = colorOnGreen
				}
				return s
			})))
		}
		return ui.View(ui.ViewProps{Style: ui.Style{Absolute: true, Left: ui.Pt(14), Right: ui.Pt(14), Bottom: ui.Pt(62), Transition: ui.Pop}},
			ui.View(ui.ViewProps{Style: ui.Style{
				Direction: ui.RowDir, Wrap: true, Gap: 8, Padding: ui.All(12), CornerRadius: 16, Background: colorBG,
				Shadow:      ui.Shadow{Color: ui.RGBA(0, 0, 0, 170), Radius: 18, Y: 6},
				BorderWidth: 1, BorderColor: colorBorder,
			}}, chips...),
			popoverArrow(ui.Pt(40)),
		)
	}
	return ui.Column(nil,
		filename(),
		sampleCard(cardParts{
			name: nameText(),
			address: ui.View(ui.ViewProps{
				OnLongPress: func() {
					ui.Haptic(ui.HapticMedium)
					ui.AnimateWith(ui.Animation{Curve: ui.Spring, Duration: 0.3}, func() { open.Set(true) })
				},
				OnPress: func() { ui.Animate(func() { open.Set(!open.Peek()) }) },
				Label:   t("Press and hold the address"),
				Style:   ui.Style{Padding: ui.Symmetric(4, 0)},
			},
				addressText(word),
				ui.Show(signal.Func(func() bool { return !open.Get() }), func() ui.Node {
					return ui.View(ui.ViewProps{Style: ui.Style{Absolute: true, Left: ui.Pct(40), Top: ui.Pt(10)}}, bobbingHand())
				}),
			),
			covered: ui.Val(true),
			color:   colorRaised,
			extra:   ui.Show(open, popover),
		}),
		caption(finished, signal.Func(func() string {
			if open.Get() {
				return t("Tap a word to cover it")
			}
			return t("Press and hold the address")
		})),
	)
}

// popoverArrow points down from a popover's bottom edge.
func popoverArrow(left ui.Dim) ui.Node {
	return ui.Shape(ui.ShapeProps{
		Path: func(w, h float32) *ui.Path {
			return new(ui.Path).MoveTo(0, 0).LineTo(w, 0).LineTo(w/2, h).Close()
		},
		Fill:  colorBG,
		Style: ui.Style{Width: ui.Pt(22), Height: ui.Pt(11), Margin: ui.Edges{Left: 30, Top: -1}},
	})
}

// The fourth: a drag draws a box.
func dragStep(finished *signal.Signal[bool]) ui.Node {
	type box struct{ x, y, w, h float32 }
	drawn := signal.New(box{})
	has := signal.Func(func() bool { return drawn.Get().w > 0 })
	onPan := func(g ui.Gesture) {
		x0, y0 := g.X-g.TranslationX, g.Y-g.TranslationY
		b := box{min(x0, g.X), min(y0, g.Y), abs32(g.TranslationX), abs32(g.TranslationY)}
		switch g.State {
		case ui.GestureBegan, ui.GestureChanged:
			drawn.Set(b)
		case ui.GestureEnded:
			drawn.Set(b)
			if b.w > 12 && b.h > 6 {
				ui.Haptic(ui.HapticSuccess)
				finished.Set(true)
			} else {
				drawn.Set(box{})
			}
		default:
			drawn.Set(box{})
		}
	}
	return ui.Column(nil,
		filename(),
		sampleCard(cardParts{
			name:    nameText(),
			address: addressText(plainWord),
			covered: ui.Val(true),
			color:   colorRaised,
			onPan:   onPan,
			extra: ui.Column(ui.Style{Absolute: true, Left: ui.Pt(0), Top: ui.Pt(0)},
				ui.Show(has, func() ui.Node {
					return ui.View(ui.ViewProps{Style: ui.StyleFunc(func() ui.Style {
						b := drawn.Get()
						s := ui.Style{Absolute: true, Left: ui.Pt(b.x), Top: ui.Pt(b.y), Width: ui.Pt(b.w), Height: ui.Pt(b.h), Background: colorGreen, CornerRadius: 3}
						if !finished.Get() {
							s.Background = colorGreenBG
							s.BorderWidth = 1.5
							s.BorderColor = colorGreen
						}
						return s
					})})
				}),
			),
		}),
		caption(finished, ui.Val(t("Drag across the card"))),
	)
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
