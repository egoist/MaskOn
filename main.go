// Maskon masks what's private in a screenshot or photo before you share it:
// it finds codes, faces, phone numbers and more, and covers them in a tap.
// Everything happens on the device.
package main

import (
	"github.com/egoist/ryui/app"
	"github.com/egoist/ryui/signal"
	"github.com/egoist/ryui/store"
	"github.com/egoist/ryui/ui"
)

func main() { app.Run(App) }

func App() ui.Node {
	ui.SetColorScheme(ui.ColorSchemeDark)
	setLanguage(signal.Untrack(ui.UseEnvironment().Get).Locale)
	var onboarded bool
	store.Get("onboarded", &onboarded)
	// The bar's buttons are the home screen's, which sets what they do.
	var bar homeBar
	return ui.Stack(func() ui.Node { return Home(!onboarded, &bar) }, ui.ScreenOptions{
		Title: ui.Val("Maskon"),
		Actions: []ui.Action{
			{Title: t("Pick a photo"), Icon: ui.IconPlus, OnPress: func() { bar.pick() }},
			{Title: t("How it works"), Icon: ui.Icon("questionmark.circle"), OnPress: func() { bar.help() }},
		},
	})
}
