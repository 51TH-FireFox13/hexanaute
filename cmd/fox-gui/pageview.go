package main

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/51TH-FireFox13/fox-browser/internal/engine"
)

// buildPageView construit une vue Fyne structurée à partir des blocs de rendu GUI.
func buildPageView(result *engine.GUIRenderResult, onLinkClick func(url string)) fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0, len(result.Blocks))

	for _, block := range result.Blocks {
		obj := renderBlock(block, onLinkClick)
		if obj != nil {
			objects = append(objects, obj)
		}
	}

	if len(objects) == 0 {
		return widget.NewLabel("(page vide)")
	}

	return container.NewVBox(objects...)
}

func renderBlock(block engine.GUIBlock, onLinkClick func(string)) fyne.CanvasObject {
	switch block.Type {
	case engine.BlockHeading1:
		return renderHeading(block, 24, true)
	case engine.BlockHeading2:
		return renderHeading(block, 20, true)
	case engine.BlockHeading3:
		return renderHeading(block, 17, false)
	case engine.BlockHeading4:
		return renderHeading(block, 15, false)
	case engine.BlockSep:
		sep := canvas.NewLine(color.NRGBA{R: 80, G: 80, B: 85, A: 255})
		sep.StrokeWidth = 1
		return container.NewStack(container.NewPadded(sep))
	case engine.BlockParagraph:
		return renderParagraph(block, onLinkClick, 0)
	case engine.BlockList, engine.BlockOrderedList:
		return renderParagraph(block, onLinkClick, block.Indent)
	case engine.BlockQuote:
		return renderBlockquote(block, onLinkClick)
	case engine.BlockCodeBlock:
		return renderCodeBlock(block)
	case engine.BlockTable:
		return renderTableBlock(block)
	case engine.BlockImage:
		text := segmentsToText(block.Segments)
		label := widget.NewLabel(text)
		label.TextStyle = fyne.TextStyle{Italic: true}
		return label
	default:
		return renderParagraph(block, onLinkClick, 0)
	}
}

func renderHeading(block engine.GUIBlock, size float32, bold bool) fyne.CanvasObject {
	text := segmentsToText(block.Segments)
	if text == "" {
		return nil
	}

	// Couleur : CSS prioritaire, sinon orange Fox par défaut
	headingColor := color.NRGBA{R: 255, G: 180, B: 50, A: 255}
	if block.HasFGColor {
		headingColor = block.FGColor
	}

	heading := canvas.NewText(text, headingColor)
	heading.TextSize = size
	heading.TextStyle = fyne.TextStyle{Bold: bold}

	obj := fyne.CanvasObject(container.NewVBox(container.NewPadded(heading)))

	// Fond CSS sur le bloc heading
	if block.HasBGColor {
		bg := canvas.NewRectangle(block.BGColor)
		obj = container.NewStack(bg, obj)
	}

	return obj
}

func renderParagraph(block engine.GUIBlock, onLinkClick func(string), indent int) fyne.CanvasObject {
	if len(block.Segments) == 0 {
		return nil
	}

	// Construire un RichText avec segments stylisés
	richSegs := make([]widget.RichTextSegment, 0, len(block.Segments))

	for _, seg := range block.Segments {
		if seg.Text == "" {
			continue
		}

		style := segmentToRichStyle(seg)

		if seg.Link != "" && onLinkClick != nil {
			// Lien cliquable
			linkURL := seg.Link
			hyperlink := widget.NewHyperlink(seg.Text, nil)
			hyperlink.OnTapped = func() {
				onLinkClick(linkURL)
			}
			// Utiliser un TextSegment avec couleur pour les liens
			richSegs = append(richSegs, &widget.TextSegment{
				Text:  fmt.Sprintf("%s [%d]", seg.Text, seg.LinkID),
				Style: widget.RichTextStyle{
					Inline:    true,
					TextStyle: fyne.TextStyle{Bold: false, Italic: false},
					ColorName: "primary",
				},
			})
		} else {
			richSegs = append(richSegs, &widget.TextSegment{
				Text:  seg.Text,
				Style: style,
			})
		}
	}

	if len(richSegs) == 0 {
		return nil
	}

	rt := widget.NewRichText(richSegs...)
	rt.Wrapping = fyne.TextWrapWord

	var obj fyne.CanvasObject = rt

	if indent > 0 {
		obj = container.NewHBox(
			widget.NewLabel(strings.Repeat("  ", indent)),
			rt,
		)
	}

	// Alignement CSS
	switch block.Align {
	case "center":
		obj = container.NewCenter(obj)
	case "right":
		obj = container.NewHBox(widget.NewLabel(""), obj)
	}

	// Fond CSS du bloc
	if block.HasBGColor {
		bg := canvas.NewRectangle(block.BGColor)
		obj = container.NewStack(bg, container.NewPadded(obj))
	}

	return obj
}

func renderBlockquote(block engine.GUIBlock, onLinkClick func(string)) fyne.CanvasObject {
	text := segmentsToText(block.Segments)
	if text == "" {
		return nil
	}

	bar := canvas.NewRectangle(color.NRGBA{R: 255, G: 120, B: 0, A: 200})
	bar.SetMinSize(fyne.NewSize(3, 0))

	label := widget.NewRichTextFromMarkdown("*" + text + "*")
	label.Wrapping = fyne.TextWrapWord

	return container.NewHBox(bar, container.NewPadded(label))
}

func renderCodeBlock(block engine.GUIBlock) fyne.CanvasObject {
	text := segmentsToText(block.Segments)
	if text == "" {
		return nil
	}

	bg := canvas.NewRectangle(color.NRGBA{R: 35, G: 35, B: 40, A: 255})
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Monospace: true}
	label.Wrapping = fyne.TextWrapWord

	return container.NewStack(bg, container.NewPadded(label))
}

func renderTableBlock(block engine.GUIBlock) fyne.CanvasObject {
	if len(block.Children) == 0 {
		return nil
	}

	rows := make([]fyne.CanvasObject, 0, len(block.Children))
	for i, row := range block.Children {
		text := segmentsToText(row.Segments)
		label := widget.NewLabel(text)
		if i == 0 {
			label.TextStyle = fyne.TextStyle{Bold: true}
		}
		label.Wrapping = fyne.TextWrapWord

		if i == 0 {
			bg := canvas.NewRectangle(color.NRGBA{R: 50, G: 50, B: 55, A: 255})
			rows = append(rows, container.NewStack(bg, container.NewPadded(label)))
		} else {
			rows = append(rows, container.NewPadded(label))
		}
	}

	bg := canvas.NewRectangle(color.NRGBA{R: 38, G: 38, B: 43, A: 255})
	return container.NewStack(bg, container.NewVBox(rows...))
}

func segmentToRichStyle(seg engine.RichSegment) widget.RichTextStyle {
	style := widget.RichTextStyle{
		Inline:    true,
		TextStyle: fyne.TextStyle{},
	}

	// Style sémantique HTML (tag-based)
	switch seg.Style {
	case engine.StyleBold:
		style.TextStyle.Bold = true
	case engine.StyleItalic:
		style.TextStyle.Italic = true
	case engine.StyleCode:
		style.TextStyle.Monospace = true
	case engine.StyleSmall:
		style.TextStyle.Italic = true
	case engine.StyleListItem, engine.StyleOrderedItem:
		// normal
	case engine.StyleTableHeader:
		style.TextStyle.Bold = true
	case engine.StyleLink:
		style.ColorName = "primary"
	case engine.StyleImage:
		style.TextStyle.Italic = true
	}

	// Overlay CSS calculé (priorité sur le style tag)
	if seg.Bold {
		style.TextStyle.Bold = true
	}
	if seg.Italic {
		style.TextStyle.Italic = true
	}
	if seg.Mono {
		style.TextStyle.Monospace = true
	}
	// Note: couleurs CSS inline non appliquées ici (widget.RichTextStyle sans Color).
	// Les couleurs de blocs (headings, p avec bg) sont gérées via canvas.NewText/Rectangle.

	return style
}

func segmentsToText(segs []engine.RichSegment) string {
	var buf strings.Builder
	for _, seg := range segs {
		buf.WriteString(seg.Text)
	}
	return strings.TrimSpace(buf.String())
}
