# Maskon

Mask what's private in a screenshot or photo before you share it. Built with
[RyUI](https://github.com/egoist/RyUI): Go, with native UIKit and Android Views.

- **Auto mask**: finds faces, QR codes and barcodes, phone numbers, emails,
  card and ID numbers, amounts, plates, IPs, links, labelled names and
  addresses, and covers them in one tap.
- **Tap** a line of text to cover it, or a mask to remove it.
- **Long press** text to pick single words (dictionary-based for Chinese).
- **Drag** to cover any area. **Pinch** to zoom, two fingers to move.
- Solid colors or a mosaic, undo and redo, save or share.
- On the device: OCR is Apple's Vision or ML Kit's bundled models, nothing is
  uploaded. Exports are written anew, so location and camera data are gone.

## Run

```sh
go tool ryui dev -ios -once .       # the iOS simulator
go tool ryui dev -android -once .   # an Android emulator
go test ./...
```

`go.mod` pins a commit of RyUI's `main`. To move to the latest:

```sh
go get github.com/egoist/ryui@main
```

`ryui dev` without `-once` runs the Go code on your Mac and reloads on save,
but on Android the photos' files are on the device: use `-once` there.

## Layout

- `detect`: what's private in a line of text.
- `mask`: the model: what was found, what's covered, undo states, and the
  arithmetic between the image and the screen.
- `analyze.go`: OCR, a slice at a time for long screenshots.
- `home.go`, `editor.go`, `onboarding.go`: the screens.

## Before shipping

- Set `id` in `ryui.json`, and `ios.team` and the Android keystore
  (`ryui build`, see RyUI's docs/shipping.md).
- Sharing into the app needs an App Group on device builds.
- Google Play asks an app reading the photo library to justify it in the
  Play Console. The `+` button's picker works without that permission.
- A mosaic can't be read back at the block size used here, but only a solid
  color removes the pixels outright, so solid is the default.

## License

[AGPL-3.0](LICENSE)
