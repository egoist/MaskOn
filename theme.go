package main

import (
	"github.com/egoist/ryui/signal"
	"github.com/egoist/ryui/svg"
	"github.com/egoist/ryui/ui"
)

// The look: near black, dotted like graph paper, with one green.
var (
	colorBG      = ui.Hex(0x1C1C1E)
	colorCard    = ui.Hex(0x1C1C1E)
	colorRaised  = ui.Hex(0x2C2C2E)
	colorChip    = ui.Hex(0x2C2C2E)
	colorBorder  = ui.RGBA(255, 255, 255, 26)
	colorDot     = ui.RGBA(255, 255, 255, 34)
	colorText    = ui.Hex(0xFFFFFF)
	colorDim     = ui.RGBA(235, 235, 245, 150)
	colorFaint   = ui.RGBA(235, 235, 245, 90)
	colorGreen   = ui.Hex(0x65D36E)
	colorGreenHi = ui.Hex(0x30D158)
	colorGreenBG = ui.RGBA(0x65, 0xD3, 0x6E, 46)
	colorOnGreen = ui.Hex(0x0B2A0F)
	colorOff     = ui.Hex(0x5A5A5E)
)

// fill is the style of a view that covers its parent.
func fill() ui.Style {
	return ui.Style{Absolute: true, Left: ui.Pt(0), Top: ui.Pt(0), Right: ui.Pt(0), Bottom: ui.Pt(0)}
}

// dots is the dotted paper behind everything.
func dots() ui.Node {
	const step, r = 20, 0.9
	return ui.Shape(ui.ShapeProps{
		Path: func(w, h float32) *ui.Path {
			p := new(ui.Path)
			for y := float32(step / 2); y < h; y += step {
				for x := float32(step / 2); x < w; x += step {
					p.Ellipse(x-r, y-r, 2*r, 2*r)
				}
			}
			return p
		},
		Fill:  colorDot,
		Style: fill(),
	})
}

// capsule is the screen's big button.
func capsule(title signal.Accessor[string], enabled signal.Accessor[bool], press func()) ui.Node {
	return ui.View(ui.ViewProps{
		Role: ui.RoleButton,
		OnPress: func() {
			if enabled.Get() {
				press()
			}
		},
		Style: ui.StyleFunc(func() ui.Style {
			s := ui.Style{
				Height: ui.Pt(58), CornerRadius: 29, Align: ui.AlignCenter, Justify: ui.JustifyCenter,
				Background: colorOff, BorderWidth: 1, BorderColor: ui.RGBA(255, 255, 255, 40),
				Pressed: &ui.Style{Transform: ui.Transform{Scale: 0.98}},
			}
			if enabled.Get() {
				s.Background = colorGreen
				s.BorderColor = ui.RGBA(255, 255, 255, 80)
			}
			return s
		}),
	}, ui.Text(title, ui.StyleFunc(func() ui.Style {
		s := ui.Style{Font: ui.Font{Size: 18, Weight: ui.Semibold}, Color: ui.RGBA(255, 255, 255, 110)}
		if enabled.Get() {
			s.Color = colorText
		}
		return s
	})))
}

// The icons Android's symbol set lacks, from Lucide (ISC).
var (
	handIcon  = svg.MustParse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 14a8 8 0 0 1-8 8"/><path d="M18 11v-1a2 2 0 0 0-2-2a2 2 0 0 0-2 2"/><path d="M14 10V9a2 2 0 0 0-2-2a2 2 0 0 0-2 2v1"/><path d="M10 9.5V4a2 2 0 0 0-2-2a2 2 0 0 0-2 2v10"/><path d="M18 11a2 2 0 1 1 4 0v3a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15"/></svg>`)
	wandIcon  = svg.MustParse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21.64 3.64-1.28-1.28a1.21 1.21 0 0 0-1.72 0L2.36 18.64a1.21 1.21 0 0 0 0 1.72l1.28 1.28a1.2 1.2 0 0 0 1.72 0L21.64 5.36a1.2 1.2 0 0 0 0-1.72"/><path d="m14 7 3 3"/><path d="M5 6v4"/><path d="M19 14v4"/><path d="M10 2v2"/><path d="M7 8H3"/><path d="M21 16h-4"/><path d="M11 3H9"/></svg>`)
	imageIcon = svg.MustParse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>`)
)

func icon(i *svg.Icon, size float32, color ui.Color) ui.Node {
	return svg.View(svg.Props{Icon: i, Color: ui.Val(color), Style: ui.Style{Width: ui.Pt(size), Height: ui.Pt(size)}})
}

// bobbingHand is the pointer that shows where to touch.
func bobbingHand() ui.Node {
	down := signal.New(false)
	started := false
	return ui.View(ui.ViewProps{
		OnLayout: func(ui.Frame) {
			if !started {
				started = true
				ui.AnimateWith(ui.Animation{Duration: 0.7, Repeat: ui.Forever, Autoreverses: true}, func() { down.Set(true) })
			}
		},
		Style: ui.StyleFunc(func() ui.Style {
			if down.Get() {
				return ui.Style{Transform: ui.Transform{TranslateY: 5, Scale: 0.92}}
			}
			return ui.Style{}
		}),
	}, icon(handIcon, 30, colorGreen))
}
