package main

import (
	"fmt"
	"slices"

	"github.com/egoist/ryui/app"
	"github.com/egoist/ryui/device"
	"github.com/egoist/ryui/photos"
	"github.com/egoist/ryui/signal"
	"github.com/egoist/ryui/store"
	"github.com/egoist/ryui/ui"
)

const pageSize = 60

// homeBar is what the home screen's bar buttons do.
type homeBar struct{ pick, help func() }

// Home is the grid of the latest screenshots or photos, a tap from masking.
func Home(showOnboarding bool, bar *homeBar) ui.Node {
	nav := ui.UseNavigator()
	help := signal.New(showOnboarding)
	access := signal.New(photos.NotAsked)
	asked := signal.New(!showOnboarding)
	segment := signal.New(0) // 0: screenshots; 1: everything
	assets := signal.NewFunc([]photos.Asset(nil), nil)
	loading := false
	more := true
	generation := 0 // a reload's answer replaces all before it

	load := func(reset, quiet bool) {
		if reset {
			generation++
		} else if loading || !more {
			return
		}
		loading = true
		gen := generation
		offset := len(assets.Peek())
		if reset {
			offset = 0
		}
		shots := segment.Peek() == 0
		go func() {
			list, err := photos.List(photos.Query{Screenshots: shots, Offset: offset, Limit: pageSize})
			ui.Post(func() {
				if gen != generation {
					return
				}
				loading = false
				if err != nil {
					fmt.Println("photos:", err)
					return
				}
				cur := assets.Peek()
				// Back in the app with nothing new: stay where the grid is,
				// however far it has scrolled.
				if quiet && len(list) > 0 && len(cur) > 0 && list[0].ID == cur[0].ID {
					return
				}
				more = len(list) == pageSize
				if reset {
					assets.Set(list)
				} else {
					assets.Set(slices.Concat(cur, list))
				}
			})
		}()
	}
	check := func(ask bool) {
		go func() {
			var a photos.Access
			var err error
			if ask {
				a, err = photos.Request()
			} else {
				a, err = photos.Current()
			}
			ui.Post(func() {
				if err != nil {
					fmt.Println("photos:", err)
				}
				access.Set(a)
				if a == photos.Allowed || a == photos.Limited {
					load(true, !ask)
				}
			})
		}()
	}
	signal.Effect(func() {
		segment.Get() // switching reloads
		signal.Untrack(func() bool {
			more = true
			if a := access.Peek(); a == photos.Allowed || a == photos.Limited {
				load(true, false)
			}
			return true
		})
	})
	if showOnboarding {
		// Asked once the onboarding has made its case.
	} else {
		check(true)
	}
	// Back from Settings or the screenshot just taken.
	app.OnLifecycle(func(s app.State) {
		if s == app.Foreground && asked.Peek() {
			check(false)
		}
	})

	open := func(src Source, zoom string) {
		title := signal.New(src.Name)
		nav.Push(func() ui.Node { return Editor(src, title) }, ui.ScreenOptions{
			Title:    title,
			ZoomFrom: zoom,
			// A drag from the edge draws a box, as a drag from anywhere does.
			NoBackGesture: true,
		})
	}
	pick := func() {
		device.PickPhotos(1, func(paths []string, err error) {
			if err == nil && len(paths) > 0 {
				open(Source{Path: paths[0]}, "")
			}
		})
	}
	ui.OnShare(func(items []ui.SharedItem) {
		for _, it := range items {
			if it.Kind == ui.SharedImage {
				open(Source{Path: it.Path, Name: it.Name}, "")
				return
			}
		}
	})
	bar.pick = pick
	bar.help = func() { help.Set(true) }

	finishOnboarding := func() {
		store.Set("onboarded", true)
		help.Set(false)
		if !asked.Peek() {
			asked.Set(true)
			check(true)
		}
	}

	grid := func() ui.Node {
		return ui.List(ui.ListProps{
			Columns:      3,
			Gap:          2,
			Style:        ui.Style{Background: colorBG},
			OnEndReached: func() { load(false, false) },
			OnRefresh:    func() { more = true; load(true, false) },
		}, assets, func(a photos.Asset) string { return a.ID }, func(a signal.Accessor[photos.Asset], _ signal.Accessor[int]) ui.Node {
			return gridCell(a, open)
		})
	}

	allowed := signal.Func(func() bool {
		a := access.Get()
		return a == photos.Allowed || a == photos.Limited
	})
	return ui.View(ui.ViewProps{Style: ui.Style{Grow: 1, Background: colorBG}},
		ui.SafeArea(ui.Style{Grow: 1},
			ui.View(ui.ViewProps{Style: ui.Style{Padding: ui.Edges{Left: 16, Right: 16, Top: 4, Bottom: 10}}},
				ui.Segmented(ui.SegmentedProps{Segments: []string{t("Screenshots"), t("All Photos")}, Selected: segment}),
			),
			ui.Show(signal.Func(func() bool { return access.Get() == photos.Limited }), func() ui.Node {
				return ui.View(ui.ViewProps{
					OnPress: func() {
						go func() {
							photos.ManageLimited()
							ui.Post(func() { load(true, false) })
						}()
					},
					Label: t("Choose More Photos"),
					Role:  ui.RoleButton,
					Style: ui.Style{AlignSelf: ui.AlignCenter, Padding: ui.Symmetric(4, 12), Margin: ui.Edges{Bottom: 8}},
				}, ui.StaticText(t("Choose More Photos"), ui.Style{Font: ui.Font{Size: 14, Weight: ui.Medium}, Color: colorGreen}))
			}),
			ui.Show(allowed, func() ui.Node {
				return ui.View(ui.ViewProps{Style: ui.Style{Grow: 1}},
					grid(),
					ui.Show(signal.Func(func() bool { return len(assets.Get()) == 0 }), func() ui.Node {
						return emptyState(t("Nothing here yet"), t("Take a screenshot, or pick a photo."), t("Pick a photo"), pick)
					}),
				)
			}, func() ui.Node {
				return ui.Show(signal.Func(func() bool { return access.Get() == photos.Denied }), func() ui.Node {
					// Android has no link to an app's settings, but its picker needs
					// no permission.
					if signal.Untrack(ui.UseEnvironment().Get).Platform != ui.PlatformIOS {
						return emptyState(t("Allow access to your photos"), t("Maskon shows your latest screenshots here. Nothing ever leaves your device."), t("Pick a photo"), pick)
					}
					return emptyState(t("Allow access to your photos"), t("Maskon shows your latest screenshots here. Nothing ever leaves your device."), t("Open Settings"), func() {
						ui.OpenURL("app-settings:")
					})
				}, func() ui.Node {
					return emptyState(t("Allow access to your photos"), t("Maskon shows your latest screenshots here. Nothing ever leaves your device."), t("Allow"), func() {
						asked.Set(true)
						check(true)
					})
				})
			}),
		),
		ui.Sheet(ui.SheetProps{
			Open:      help,
			Content:   func() ui.Node { return Onboarding(finishOnboarding) },
			OnDismiss: finishOnboarding,
		}),
	)
}

func emptyState(title, body, action string, press func()) ui.Node {
	s := fill()
	s.Align = ui.AlignCenter
	s.Justify = ui.JustifyCenter
	s.Gap = 10
	s.Padding = ui.All(36)
	s.Background = colorBG
	return ui.View(ui.ViewProps{Style: s},
		ui.StaticText(title, ui.Style{Font: ui.Font{Size: 20, Weight: ui.Semibold}, Color: colorText, TextAlign: ui.TextCenter}),
		ui.StaticText(body, ui.Style{Font: ui.Font{Size: 15}, Color: colorDim, TextAlign: ui.TextCenter}),
		ui.View(ui.ViewProps{
			OnPress: press,
			Label:   action,
			Role:    ui.RoleButton,
			Style: ui.Style{Margin: ui.Edges{Top: 10}, Padding: ui.Symmetric(11, 22), CornerRadius: 22, Background: colorGreen,
				Pressed: &ui.Style{Transform: ui.Transform{Scale: 0.97}}},
		}, ui.StaticText(action, ui.Style{Font: ui.Font{Size: 16, Weight: ui.Semibold}, Color: colorText})),
	)
}

// gridCell is a photo in the grid. Its thumbnail is a file the platform makes
// off the main thread; a row reused for another photo drops the old answer.
func gridCell(a signal.Accessor[photos.Asset], open func(Source, string)) ui.Node {
	thumb := signal.New("")
	id := ""
	signal.Effect(func() {
		want := a.Get().ID
		if want == id {
			return
		}
		id = want
		thumb.Set("")
		go func() {
			p, err := photos.Thumbnail(want, 360)
			ui.Post(func() {
				if err == nil && id == want {
					thumb.Set(p)
				}
			})
		}()
	})
	return ui.View(ui.ViewProps{
		OnPress: func() {
			as := signal.Untrack(a.Get)
			open(Source{AssetID: as.ID}, "photo-"+as.ID)
		},
		Label: t("Photo"),
		Style: ui.StyleFunc(func() ui.Style {
			return ui.Style{AspectRatio: 0.72, Background: colorRaised, Clip: true, Match: "photo-" + a.Get().ID}
		}),
	}, ui.Image(ui.ImageProps{Source: thumb, Mode: ui.ImageFill, Style: fill()}))
}
