package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// zoomTheme scales the default theme by the configured zoom level.
type zoomTheme struct {
	scale float32
}

func (t zoomTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(n, v)
}

func (t zoomTheme) Font(s fyne.TextStyle) fyne.Resource { return theme.DefaultTheme().Font(s) }

func (t zoomTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }

func (t zoomTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n) * t.scale
}

// Status of the generated password field, which sets its text colour.
type status int

const (
	statusInfo status = iota
	statusOK
	statusError
)

// foregroundTheme overrides the text colour of the objects it is applied to.
type foregroundTheme struct {
	status status
}

func (t *foregroundTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == theme.ColorNameForeground {
		switch {
		case t.status == statusOK && v == theme.VariantDark:
			return color.NRGBA{R: 0x5c, G: 0xd0, B: 0x5c, A: 0xff}
		case t.status == statusOK:
			return color.NRGBA{R: 0x00, G: 0x80, B: 0x00, A: 0xff}
		case t.status == statusError && v == theme.VariantDark:
			return color.NRGBA{R: 0xff, G: 0x72, B: 0x72, A: 0xff}
		case t.status == statusError:
			return color.NRGBA{R: 0xc0, G: 0x10, B: 0x10, A: 0xff}
		}
	}
	return fyne.CurrentApp().Settings().Theme().Color(n, v)
}

func (t *foregroundTheme) Font(s fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(s)
}

func (t *foregroundTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(n)
}

func (t *foregroundTheme) Size(n fyne.ThemeSizeName) float32 {
	return fyne.CurrentApp().Settings().Theme().Size(n)
}
