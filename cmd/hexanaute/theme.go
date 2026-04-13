package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// foxTheme est le thème personnalisé de HexaNaute.
type foxTheme struct{}

func (t *foxTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 255, G: 120, B: 0, A: 255} // Orange Fox
	case theme.ColorNameBackground:
		return color.NRGBA{R: 30, G: 30, B: 35, A: 255} // Fond sombre
	case theme.ColorNameForeground:
		return color.NRGBA{R: 230, G: 230, B: 235, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 45, G: 45, B: 50, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 50, G: 50, B: 55, A: 255}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 60, G: 60, B: 65, A: 255}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *foxTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *foxTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *foxTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNamePadding:
		return 6
	}
	return theme.DefaultTheme().Size(name)
}
