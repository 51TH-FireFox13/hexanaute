package engine

import (
	"fmt"
	"strings"
)

// Link représente un lien trouvé dans la page.
type Link struct {
	Index int
	Text  string
	URL   string
}

// RenderResult contient le rendu texte et les liens extraits.
type RenderResult struct {
	Content string
	Links   []Link
}

// Render produit le rendu terminal d'un arbre DOM.
func Render(root *Element) *RenderResult {
	r := &renderer{
		links:    make([]Link, 0, 32),
		listDepth: 0,
	}
	r.render(root)

	content := r.buf.String()
	// Nettoyer les lignes vides multiples
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	return &RenderResult{
		Content: content,
		Links:   r.links,
	}
}

type renderer struct {
	buf       strings.Builder
	links     []Link
	listDepth int
	olCounter []int // compteurs pour listes ordonnées
	inPre     bool
	tableRows [][]string // collecte des cellules pour rendu tableau
	inTable   bool
	currentRow []string
}

// Tags à ignorer.
var skipTags = map[string]bool{
	"script": true, "style": true, "noscript": true,
	"meta": true, "link": true, "head": true,
	"svg": true, "iframe": true, "template": true,
}

// Tags bloc.
var blockTags = map[string]bool{
	"div": true, "p": true, "br": true, "hr": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"li": true, "tr": true, "blockquote": true, "pre": true,
	"section": true, "article": true, "header": true, "footer": true,
	"nav": true, "main": true, "aside": true, "figure": true,
	"figcaption": true, "details": true, "summary": true,
	"table": true, "thead": true, "tbody": true, "tfoot": true,
	"form": true, "fieldset": true, "legend": true,
	"dd": true, "dt": true, "dl": true,
}

func (r *renderer) render(el *Element) {
	if skipTags[el.Tag] {
		return
	}

	switch el.Tag {
	case "h1":
		r.buf.WriteString("\n══════════════════════════════════════════════\n")
		r.renderInlineContent(el)
		r.buf.WriteString("\n══════════════════════════════════════════════\n")
		return
	case "h2":
		r.buf.WriteString("\n── ")
		r.renderInlineContent(el)
		r.buf.WriteString(" ──\n")
		return
	case "h3":
		r.buf.WriteString("\n   ▸ ")
		r.renderInlineContent(el)
		r.buf.WriteString("\n")
		return
	case "h4":
		r.buf.WriteString("\n     ▹ ")
		r.renderInlineContent(el)
		r.buf.WriteString("\n")
		return
	case "h5", "h6":
		r.buf.WriteString("\n       ")
		r.renderInlineContent(el)
		r.buf.WriteString("\n")
		return
	case "hr":
		r.buf.WriteString("\n──────────────────────────────────────────────\n")
		return
	case "br":
		r.buf.WriteString("\n")
		return
	case "ul":
		r.listDepth++
		for _, child := range el.Children {
			r.render(child)
		}
		r.listDepth--
		if r.listDepth == 0 {
			r.buf.WriteString("\n")
		}
		return
	case "ol":
		r.listDepth++
		r.olCounter = append(r.olCounter, 0)
		for _, child := range el.Children {
			r.render(child)
		}
		r.olCounter = r.olCounter[:len(r.olCounter)-1]
		r.listDepth--
		if r.listDepth == 0 {
			r.buf.WriteString("\n")
		}
		return
	case "li":
		indent := strings.Repeat("  ", r.listDepth)
		if len(r.olCounter) > 0 {
			r.olCounter[len(r.olCounter)-1]++
			r.buf.WriteString(fmt.Sprintf("%s%d. ", indent, r.olCounter[len(r.olCounter)-1]))
		} else {
			bullets := []string{"•", "◦", "▪", "▫"}
			bullet := bullets[0]
			if r.listDepth > 0 && r.listDepth <= len(bullets) {
				bullet = bullets[r.listDepth-1]
			}
			r.buf.WriteString(indent + bullet + " ")
		}
		r.renderInlineContent(el)
		r.buf.WriteString("\n")
		// Rendre les sous-listes
		for _, child := range el.Children {
			if child.Tag == "ul" || child.Tag == "ol" {
				r.render(child)
			}
		}
		return
	case "table":
		r.renderTable(el)
		return
	case "blockquote":
		r.buf.WriteString("\n  │ ")
		r.renderInlineContent(el)
		r.buf.WriteString("\n")
		return
	case "pre":
		r.buf.WriteString("\n┌─────────────────────────────────────\n")
		r.inPre = true
		r.renderPreContent(el)
		r.inPre = false
		r.buf.WriteString("\n└─────────────────────────────────────\n")
		return
	case "a":
		r.renderLink(el)
		return
	case "img":
		alt := el.Attrs["alt"]
		if alt != "" {
			r.buf.WriteString("[img: " + alt + "] ")
		}
		return
	case "input":
		r.renderInput(el)
		return
	case "button":
		r.buf.WriteString("[")
		r.renderInlineContent(el)
		r.buf.WriteString("] ")
		return
	case "select":
		r.buf.WriteString("[▼ sélection] ")
		return
	case "textarea":
		r.buf.WriteString("[zone de texte] ")
		return
	case "dl":
		for _, child := range el.Children {
			r.render(child)
		}
		r.buf.WriteString("\n")
		return
	case "dt":
		r.buf.WriteString("\n  ")
		r.renderInlineContent(el)
		r.buf.WriteString("\n")
		return
	case "dd":
		r.buf.WriteString("    → ")
		r.renderInlineContent(el)
		r.buf.WriteString("\n")
		return
	}

	// Texte du nœud
	if el.Text != "" {
		r.writeStyledText(el.Tag, el.Text)
	}

	// Enfants
	for _, child := range el.Children {
		r.render(child)
	}

	// Fin de bloc
	if blockTags[el.Tag] {
		r.buf.WriteString("\n")
	}
}

func (r *renderer) renderLink(el *Element) {
	href := el.Attrs["href"]
	text := collectText(el)
	if text == "" {
		// Essayer alt d'une image enfant
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
		r.buf.WriteString(text)
		return
	}

	r.links = append(r.links, Link{
		Index: len(r.links) + 1,
		Text:  text,
		URL:   href,
	})
	r.buf.WriteString(fmt.Sprintf("\033[36m%s\033[0m \033[90m[%d]\033[0m", text, len(r.links)))
}

func (r *renderer) renderInput(el *Element) {
	inputType := el.Attrs["type"]
	placeholder := el.Attrs["placeholder"]
	value := el.Attrs["value"]

	switch inputType {
	case "submit", "button":
		label := value
		if label == "" {
			label = "Envoyer"
		}
		r.buf.WriteString("[" + label + "] ")
	case "checkbox":
		r.buf.WriteString("☐ ")
	case "radio":
		r.buf.WriteString("○ ")
	case "hidden":
		// Ne rien afficher
	default:
		label := placeholder
		if label == "" {
			label = inputType
		}
		r.buf.WriteString("[_" + label + "_] ")
	}
}

func (r *renderer) writeStyledText(tag, text string) {
	switch tag {
	case "strong", "b":
		r.buf.WriteString("\033[1m" + text + "\033[0m")
	case "em", "i":
		r.buf.WriteString("\033[3m" + text + "\033[0m")
	case "code":
		r.buf.WriteString("\033[33m`" + text + "`\033[0m")
	case "mark":
		r.buf.WriteString("\033[43m" + text + "\033[0m")
	case "del", "s":
		r.buf.WriteString("\033[9m" + text + "\033[0m")
	case "small":
		r.buf.WriteString("\033[2m" + text + "\033[0m")
	default:
		r.buf.WriteString(text)
	}
}

// renderInlineContent rend le texte et les enfants inline d'un élément.
func (r *renderer) renderInlineContent(el *Element) {
	if el.Text != "" {
		r.writeStyledText(el.Tag, el.Text)
	}
	for _, child := range el.Children {
		if child.Tag == "ul" || child.Tag == "ol" {
			continue // les sous-listes sont rendues après
		}
		if skipTags[child.Tag] {
			continue
		}
		if child.Tag == "a" {
			r.renderLink(child)
		} else if child.Tag == "img" {
			alt := child.Attrs["alt"]
			if alt != "" {
				r.buf.WriteString("[img: " + alt + "] ")
			}
		} else if child.Tag == "br" {
			r.buf.WriteString("\n")
		} else {
			if child.Text != "" {
				r.writeStyledText(child.Tag, child.Text)
			}
			for _, sub := range child.Children {
				r.renderInlineContent(sub)
			}
		}
	}
}

func (r *renderer) renderPreContent(el *Element) {
	if el.Text != "" {
		lines := strings.Split(el.Text, "\n")
		for _, line := range lines {
			r.buf.WriteString("│ " + line + "\n")
		}
	}
	for _, child := range el.Children {
		r.renderPreContent(child)
	}
}

// renderTable collecte et rend un tableau formaté.
func (r *renderer) renderTable(el *Element) {
	rows := collectTableRows(el)
	if len(rows) == 0 {
		return
	}

	// Calculer la largeur de chaque colonne
	colWidths := make([]int, 0)
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(colWidths) {
				colWidths = append(colWidths, 0)
			}
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Limiter la largeur max par colonne
	for i := range colWidths {
		if colWidths[i] > 40 {
			colWidths[i] = 40
		}
	}

	r.buf.WriteString("\n")

	// Ligne de séparation
	sep := "┼"
	topSep := "┌"
	botSep := "└"
	for i, w := range colWidths {
		topSep += strings.Repeat("─", w+2)
		sep += strings.Repeat("─", w+2)
		botSep += strings.Repeat("─", w+2)
		if i < len(colWidths)-1 {
			topSep += "┬"
			sep += "┼"
			botSep += "┴"
		}
	}
	topSep += "┐"
	sep += "┤"
	botSep += "┘"

	r.buf.WriteString(topSep + "\n")

	for i, row := range rows {
		r.buf.WriteString("│")
		for j, w := range colWidths {
			cell := ""
			if j < len(row) {
				cell = row[j]
				if len(cell) > w {
					cell = cell[:w-1] + "…"
				}
			}
			r.buf.WriteString(fmt.Sprintf(" %-*s │", w, cell))
		}
		r.buf.WriteString("\n")

		// Séparation après le header
		if i == 0 && len(rows) > 1 {
			r.buf.WriteString("├" + sep[1:] + "\n")
		}
	}

	r.buf.WriteString(botSep + "\n")
}

func collectTableRows(el *Element) [][]string {
	var rows [][]string
	for _, child := range el.Children {
		switch child.Tag {
		case "thead", "tbody", "tfoot":
			rows = append(rows, collectTableRows(child)...)
		case "tr":
			var cells []string
			for _, td := range child.Children {
				if td.Tag == "td" || td.Tag == "th" {
					cells = append(cells, strings.TrimSpace(collectText(td)))
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
			}
		}
	}
	return rows
}

// collectText récupère tout le texte visible d'un élément.
func collectText(el *Element) string {
	var buf strings.Builder
	if el.Text != "" {
		buf.WriteString(el.Text)
	}
	for _, child := range el.Children {
		if skipTags[child.Tag] {
			continue
		}
		t := collectText(child)
		if t != "" {
			if buf.Len() > 0 {
				buf.WriteString(" ")
			}
			buf.WriteString(t)
		}
	}
	return strings.TrimSpace(buf.String())
}
