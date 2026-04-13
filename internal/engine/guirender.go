package engine

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// RichSegment représente un segment de texte avec style pour le rendu GUI.
type RichSegment struct {
	Text   string
	Style  SegmentStyle
	Link   string // URL si c'est un lien
	LinkID int    // index du lien

	// Couleurs CSS calculées (data-fox-fg / data-fox-bg)
	Color      color.NRGBA
	HasColor   bool
	BGColor    color.NRGBA
	HasBGColor bool

	// Style CSS calculé (data-fox-bold / data-fox-italic / data-fox-mono)
	Bold bool
	Italic bool
	Mono bool

	// Taille de police CSS (data-fox-size, en px)
	FontSize float32
}

// SegmentStyle définit le style d'un segment.
type SegmentStyle int

const (
	StyleNormal SegmentStyle = iota
	StyleH1
	StyleH2
	StyleH3
	StyleH4
	StyleBold
	StyleItalic
	StyleCode
	StyleBlockquote
	StyleListItem
	StyleOrderedItem
	StyleLink
	StyleImage
	StyleSeparator
	StylePre
	StyleSmall
	StyleTableHeader
	StyleTableCell
)

// GUIBlock représente un bloc de contenu pour le rendu GUI.
type GUIBlock struct {
	Type     BlockStyleType
	Segments []RichSegment
	Children []GUIBlock // pour les sous-listes et tableaux
	Indent   int

	// Couleurs CSS du bloc (data-fox-fg / data-fox-bg)
	FGColor    color.NRGBA
	HasFGColor bool
	BGColor    color.NRGBA
	HasBGColor bool

	// Alignement texte CSS (data-fox-align : left/center/right/justify)
	Align string
}

// BlockStyleType définit le type de bloc.
type BlockStyleType int

const (
	BlockParagraph BlockStyleType = iota
	BlockHeading1
	BlockHeading2
	BlockHeading3
	BlockHeading4
	BlockList
	BlockOrderedList
	BlockQuote
	BlockCodeBlock
	BlockSep
	BlockTable
	BlockImage
)

// GUIRenderResult contient le rendu structuré pour le GUI.
type GUIRenderResult struct {
	Blocks []GUIBlock
	Links  []Link
}

// RenderForGUI produit un rendu structuré optimisé pour les interfaces graphiques.
func RenderForGUI(root *Element) *GUIRenderResult {
	gr := &guiRenderer{
		links: make([]Link, 0, 32),
	}
	gr.render(root)

	return &GUIRenderResult{
		Blocks: gr.blocks,
		Links:  gr.links,
	}
}

type guiRenderer struct {
	blocks    []GUIBlock
	links     []Link
	listDepth int
	olCounter []int
}

func (gr *guiRenderer) render(el *Element) {
	if guiSkipTags[el.Tag] {
		return
	}
	// Élément masqué par JS (display:none ou hidden=true)
	if el.Attrs != nil && el.Attrs["data-fox-hidden"] == "true" {
		return
	}

	switch el.Tag {
	case "h1":
		gr.addHeadingBlock(BlockHeading1, StyleH1, el)
	case "h2":
		gr.addHeadingBlock(BlockHeading2, StyleH2, el)
	case "h3":
		gr.addHeadingBlock(BlockHeading3, StyleH3, el)
	case "h4", "h5", "h6":
		gr.addHeadingBlock(BlockHeading4, StyleH4, el)
	case "hr":
		gr.blocks = append(gr.blocks, GUIBlock{Type: BlockSep})
	case "br":
		// ajouter un saut dans le bloc courant
	case "p":
		segs := gr.collectSegments(el)
		if len(segs) > 0 {
			block := GUIBlock{Type: BlockParagraph, Segments: segs}
			readBlockAttrs(el, &block)
			gr.blocks = append(gr.blocks, block)
		}
	case "ul":
		gr.renderList(el, false)
	case "ol":
		gr.renderList(el, true)
	case "blockquote":
		segs := gr.collectSegments(el)
		if len(segs) > 0 {
			gr.blocks = append(gr.blocks, GUIBlock{Type: BlockQuote, Segments: segs})
		}
	case "pre":
		text := guiCollectAllText(el)
		if text != "" {
			gr.blocks = append(gr.blocks, GUIBlock{
				Type:     BlockCodeBlock,
				Segments: []RichSegment{{Text: text, Style: StyleCode}},
			})
		}
	case "table":
		gr.renderTable(el)
	case "img":
		alt := el.Attrs["alt"]
		src := el.Attrs["src"]
		if alt != "" || src != "" {
			label := alt
			if label == "" {
				label = src
			}
			gr.blocks = append(gr.blocks, GUIBlock{
				Type:     BlockImage,
				Segments: []RichSegment{{Text: "[Image: " + label + "]", Style: StyleImage}},
			})
		}
	case "div", "section", "article", "main", "header", "footer", "nav",
		"aside", "figure", "figcaption", "details", "summary",
		"form", "fieldset":
		// Conteneurs : rendre les enfants
		for _, child := range el.Children {
			gr.render(child)
		}
	case "a":
		// Lien au niveau bloc (rare)
		segs := gr.collectSegments(el)
		if len(segs) > 0 {
			gr.blocks = append(gr.blocks, GUIBlock{Type: BlockParagraph, Segments: segs})
		}
	default:
		// Éléments inline ou inconnus au niveau bloc
		segs := gr.collectSegments(el)
		if len(segs) > 0 {
			gr.blocks = append(gr.blocks, GUIBlock{Type: BlockParagraph, Segments: segs})
		} else {
			for _, child := range el.Children {
				gr.render(child)
			}
		}
	}
}

func (gr *guiRenderer) addHeadingBlock(blockType BlockStyleType, style SegmentStyle, el *Element) {
	segs := gr.collectSegments(el)
	if len(segs) == 0 {
		text := guiCollectAllText(el)
		if text != "" {
			seg := RichSegment{Text: text, Style: style}
			applyAttrsToSeg(el, &seg)
			segs = []RichSegment{seg}
		}
	}
	// Forcer le style heading sur tous les segments (sans écraser les couleurs CSS)
	for i := range segs {
		segs[i].Style = style
	}
	if len(segs) > 0 {
		block := GUIBlock{Type: blockType, Segments: segs}
		readBlockAttrs(el, &block)
		gr.blocks = append(gr.blocks, block)
	}
}

func (gr *guiRenderer) renderList(el *Element, ordered bool) {
	counter := 0
	for _, child := range el.Children {
		if child.Tag != "li" {
			continue
		}
		counter++
		segs := gr.collectSegments(child)

		var prefix string
		if ordered {
			prefix = fmt.Sprintf("%d. ", counter)
		} else {
			bullets := []string{"•", "◦", "▪"}
			idx := gr.listDepth
			if idx >= len(bullets) {
				idx = len(bullets) - 1
			}
			prefix = bullets[idx] + " "
		}

		if len(segs) > 0 {
			// Ajouter le préfixe au premier segment
			segs[0].Text = prefix + segs[0].Text
		} else {
			text := guiCollectAllText(child)
			if text == "" {
				text = " "
			}
			segs = []RichSegment{{Text: prefix + text, Style: StyleListItem}}
		}

		blockType := BlockList
		if ordered {
			blockType = BlockOrderedList
		}
		gr.blocks = append(gr.blocks, GUIBlock{
			Type: blockType, Segments: segs, Indent: gr.listDepth,
		})

		// Sous-listes
		gr.listDepth++
		for _, sub := range child.Children {
			if sub.Tag == "ul" || sub.Tag == "ol" {
				gr.renderList(sub, sub.Tag == "ol")
			}
		}
		gr.listDepth--
	}
}

func (gr *guiRenderer) renderTable(el *Element) {
	rows := collectTableRows(el)
	if len(rows) == 0 {
		return
	}

	tableBlock := GUIBlock{Type: BlockTable}
	for i, row := range rows {
		style := StyleTableCell
		if i == 0 {
			style = StyleTableHeader
		}
		var segs []RichSegment
		for j, cell := range row {
			if j > 0 {
				segs = append(segs, RichSegment{Text: " | ", Style: StyleNormal})
			}
			segs = append(segs, RichSegment{Text: cell, Style: style})
		}
		tableBlock.Children = append(tableBlock.Children, GUIBlock{
			Type: BlockParagraph, Segments: segs,
		})
	}
	gr.blocks = append(gr.blocks, tableBlock)
}

// collectSegments collecte les segments inline d'un élément.
func (gr *guiRenderer) collectSegments(el *Element) []RichSegment {
	segs := make([]RichSegment, 0)

	if el.Text != "" {
		style := tagToStyle(el.Tag)
		seg := RichSegment{Text: el.Text, Style: style}
		applyAttrsToSeg(el, &seg)
		segs = append(segs, seg)
	}

	for _, child := range el.Children {
		if guiSkipTags[child.Tag] {
			continue
		}
		// Sauter les sous-listes (gérées par renderList)
		if child.Tag == "ul" || child.Tag == "ol" {
			continue
		}
		// Sauter les éléments masqués
		if child.Attrs != nil && child.Attrs["data-fox-hidden"] == "true" {
			continue
		}
		if child.Tag == "a" {
			segs = append(segs, gr.linkSegment(child)...)
		} else if child.Tag == "img" {
			alt := child.Attrs["alt"]
			if alt != "" {
				segs = append(segs, RichSegment{Text: "[" + alt + "]", Style: StyleImage})
			}
		} else if child.Tag == "br" {
			segs = append(segs, RichSegment{Text: "\n", Style: StyleNormal})
		} else if child.Tag == "input" {
			segs = append(segs, gr.inputSegment(child))
		} else if child.Tag == "button" {
			text := guiCollectAllText(child)
			if text != "" {
				seg := RichSegment{Text: "[" + text + "]", Style: StyleBold}
				applyAttrsToSeg(child, &seg)
				segs = append(segs, seg)
			}
		} else {
			// Inline récursif (les attrs sont déjà propagés aux enfants par le CSS resolver)
			childSegs := gr.collectSegments(child)
			segs = append(segs, childSegs...)
		}
	}

	return segs
}

func (gr *guiRenderer) linkSegment(el *Element) []RichSegment {
	href := el.Attrs["href"]
	text := guiCollectAllText(el)
	if text == "" {
		for _, child := range el.Children {
			if child.Tag == "img" {
				text = child.Attrs["alt"]
			}
		}
	}
	if text == "" {
		text = href
	}
	if href == "" || href == "#" || strings.HasPrefix(href, "javascript:") {
		return []RichSegment{{Text: text, Style: StyleNormal}}
	}

	gr.links = append(gr.links, Link{Index: len(gr.links) + 1, Text: text, URL: href})
	return []RichSegment{
		{Text: text, Style: StyleLink, Link: href, LinkID: len(gr.links)},
	}
}

func (gr *guiRenderer) inputSegment(el *Element) RichSegment {
	inputType := el.Attrs["type"]
	placeholder := el.Attrs["placeholder"]
	value := el.Attrs["value"]

	switch inputType {
	case "submit", "button":
		label := value
		if label == "" {
			label = "Envoyer"
		}
		return RichSegment{Text: "[" + label + "]", Style: StyleBold}
	case "checkbox":
		return RichSegment{Text: "☐ ", Style: StyleNormal}
	case "radio":
		return RichSegment{Text: "○ ", Style: StyleNormal}
	case "hidden":
		return RichSegment{}
	default:
		label := placeholder
		if label == "" {
			label = inputType
		}
		return RichSegment{Text: "[" + label + "]", Style: StyleSmall}
	}
}

func tagToStyle(tag string) SegmentStyle {
	switch tag {
	case "strong", "b":
		return StyleBold
	case "em", "i":
		return StyleItalic
	case "code":
		return StyleCode
	case "small":
		return StyleSmall
	default:
		return StyleNormal
	}
}

func guiCollectAllText(el *Element) string {
	var buf strings.Builder
	if el.Text != "" {
		buf.WriteString(el.Text)
	}
	for _, child := range el.Children {
		if guiSkipTags[child.Tag] {
			continue
		}
		t := guiCollectAllText(child)
		if t != "" {
			if buf.Len() > 0 && !strings.HasSuffix(buf.String(), " ") {
				buf.WriteString(" ")
			}
			buf.WriteString(t)
		}
	}
	return strings.TrimSpace(buf.String())
}

var guiSkipTags = map[string]bool{
	"script": true, "style": true, "noscript": true,
	"meta": true, "link": true, "head": true,
	"svg": true, "iframe": true, "template": true,
}

// parseHexColor parse une couleur hexadécimale "#rrggbb" sans dépendre du package css.
func parseHexColor(s string) (color.NRGBA, bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.NRGBA{}, false
	}
	r, e1 := strconv.ParseUint(s[0:2], 16, 8)
	g, e2 := strconv.ParseUint(s[2:4], 16, 8)
	b, e3 := strconv.ParseUint(s[4:6], 16, 8)
	if e1 != nil || e2 != nil || e3 != nil {
		return color.NRGBA{}, false
	}
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}, true
}

// applyAttrsToSeg copie les propriétés data-fox-* d'un élément dans un RichSegment.
func applyAttrsToSeg(el *Element, seg *RichSegment) {
	if el.Attrs == nil {
		return
	}
	if fg, ok := el.Attrs["data-fox-fg"]; ok && fg != "" {
		if c, ok2 := parseHexColor(fg); ok2 {
			seg.Color = c
			seg.HasColor = true
		}
	}
	if el.Attrs["data-fox-bold"] == "1" {
		seg.Bold = true
	}
	if el.Attrs["data-fox-italic"] == "1" {
		seg.Italic = true
	}
	if el.Attrs["data-fox-mono"] == "1" {
		seg.Mono = true
	}
	if sz, ok := el.Attrs["data-fox-size"]; ok && sz != "" {
		if f, err := strconv.ParseFloat(sz, 32); err == nil {
			seg.FontSize = float32(f)
		}
	}
}

// readBlockAttrs copie les propriétés CSS bloc (couleur, alignement) d'un élément.
func readBlockAttrs(el *Element, block *GUIBlock) {
	if el.Attrs == nil {
		return
	}
	if fg, ok := el.Attrs["data-fox-fg"]; ok && fg != "" {
		if c, ok2 := parseHexColor(fg); ok2 {
			block.FGColor = c
			block.HasFGColor = true
		}
	}
	if bg, ok := el.Attrs["data-fox-bg"]; ok && bg != "" {
		if c, ok2 := parseHexColor(bg); ok2 {
			block.BGColor = c
			block.HasBGColor = true
		}
	}
	if align, ok := el.Attrs["data-fox-align"]; ok && align != "" {
		block.Align = align
	}
}
