package css

import (
	"sort"
	"strconv"
	"strings"

	"github.com/51TH-FireFox13/hexanaute/internal/engine"
)

// ResolveStyles applique les règles CSS à l'arbre DOM.
// Écrit les styles calculés dans les attributs data-fox-* de chaque élément.
// Gère l'héritage des propriétés héritables (color, font-*).
func ResolveStyles(root *engine.Element, sheets []*StyleSheet) {
	if root == nil || len(sheets) == 0 {
		return
	}
	// Collecter toutes les règles dans une seule liste
	allRules := make([]*Rule, 0, 64)
	for _, sheet := range sheets {
		if sheet != nil {
			allRules = append(allRules, sheet.Rules...)
		}
	}
	if len(allRules) == 0 {
		return
	}
	// Trier par spécificité croissante (la plus haute spécificité appliquée en dernier)
	sort.Slice(allRules, func(i, j int) bool {
		return allRules[i].Specificity < allRules[j].Specificity
	})

	// Walker l'arbre DOM en passant le contexte ancêtral
	resolveElement(root, allRules, nil, inheritedProps{})
}

// inheritedProps contient les propriétés CSS héritées du parent.
type inheritedProps struct {
	color    string
	fontSize string
	fontFam  string
}

func resolveElement(el *engine.Element, rules []*Rule, ancestors []AncestorInfo, inherited inheritedProps) {
	if el == nil {
		return
	}

	// Initialiser les attrs si besoin
	if el.Attrs == nil {
		el.Attrs = make(map[string]string)
	}

	// Ignorer les éléments non-visuels
	if skipStyleTags[el.Tag] {
		return
	}

	// Collecter les informations de l'élément courant
	id := el.Attrs["id"]
	classes := strings.Fields(el.Attrs["class"])

	// ── Phase 1 : Règles de style des feuilles ──
	// Collecter toutes les règles qui s'appliquent à cet élément
	type appliedRule struct {
		props       map[string]string
		specificity int
	}
	matching := make([]appliedRule, 0, 8)

	for _, rule := range rules {
		if MatchesSelector(el.Tag, id, classes, ancestors, rule.Selector) {
			matching = append(matching, appliedRule{rule.Properties, rule.Specificity})
		}
	}

	// ── Phase 2 : Construire le style computé ──
	computed := make(map[string]string)

	// Héritage depuis le parent
	if inherited.color != "" {
		computed["color"] = inherited.color
	}
	if inherited.fontSize != "" {
		computed["font-size"] = inherited.fontSize
	}
	if inherited.fontFam != "" {
		computed["font-family"] = inherited.fontFam
	}

	// Appliquer les règles CSS (par ordre de spécificité)
	for _, m := range matching {
		for k, v := range m.props {
			computed[k] = v
		}
	}

	// ── Phase 3 : Style inline (priorité maximale) ──
	if inlineStyle := el.Attrs["style"]; inlineStyle != "" {
		for k, v := range ParseInlineStyle(inlineStyle) {
			computed[k] = v
		}
	}

	// ── Phase 4 : Écrire les attributs data-fox-* ──
	applyComputedStyle(el, computed)

	// ── Phase 5 : Propager aux enfants ──
	nextAncestors := append(ancestors, AncestorInfo{Tag: el.Tag, ID: id, Classes: classes})
	nextInherited := inheritedProps{
		color:    getInheritable(computed, "color", inherited.color),
		fontSize: getInheritable(computed, "font-size", inherited.fontSize),
		fontFam:  getInheritable(computed, "font-family", inherited.fontFam),
	}

	for _, child := range el.Children {
		resolveElement(child, rules, nextAncestors, nextInherited)
	}
}

// applyComputedStyle écrit les propriétés calculées dans les attributs data-fox-*.
func applyComputedStyle(el *engine.Element, computed map[string]string) {
	for prop, value := range computed {
		switch prop {
		case "display":
			v := strings.ToLower(strings.TrimSpace(value))
			if v == "none" {
				el.Attrs["data-fox-hidden"] = "true"
			} else if el.Attrs["data-fox-hidden"] == "true" && v != "" && v != "none" {
				// display explicitement non-none → démasquer
				delete(el.Attrs, "data-fox-hidden")
			}

		case "visibility":
			v := strings.ToLower(strings.TrimSpace(value))
			if v == "hidden" || v == "collapse" {
				el.Attrs["data-fox-hidden"] = "true"
			}

		case "color":
			if c, ok := ParseColor(value); ok && c.A > 0 {
				el.Attrs["data-fox-fg"] = ColorToHex(c)
			}

		case "background-color", "background":
			// Pour background shorthand, prendre juste la couleur
			bgVal := value
			if prop == "background" {
				// Chercher une couleur dans la valeur
				for _, part := range strings.Fields(bgVal) {
					if _, ok := ParseColor(part); ok {
						bgVal = part
						break
					}
				}
			}
			if c, ok := ParseColor(bgVal); ok && c.A > 0 {
				el.Attrs["data-fox-bg"] = ColorToHex(c)
			}

		case "font-weight":
			v := strings.TrimSpace(value)
			if v == "bold" || v == "bolder" {
				el.Attrs["data-fox-bold"] = "1"
			} else if n, err := strconv.Atoi(v); err == nil && n >= 600 {
				el.Attrs["data-fox-bold"] = "1"
			} else if v == "normal" || v == "400" {
				delete(el.Attrs, "data-fox-bold")
			}

		case "font-style":
			v := strings.TrimSpace(value)
			if v == "italic" || v == "oblique" {
				el.Attrs["data-fox-italic"] = "1"
			} else if v == "normal" {
				delete(el.Attrs, "data-fox-italic")
			}

		case "font-size":
			if px := parseFontSize(value); px > 0 {
				el.Attrs["data-fox-size"] = strconv.FormatFloat(float64(px), 'f', 1, 32)
			}

		case "font-family":
			lower := strings.ToLower(value)
			if strings.Contains(lower, "monospace") || strings.Contains(lower, "courier") ||
				strings.Contains(lower, "consolas") || strings.Contains(lower, "monaco") ||
				strings.Contains(lower, "menlo") || strings.Contains(lower, "jetbrains") {
				el.Attrs["data-fox-mono"] = "1"
			}

		case "text-align":
			v := strings.TrimSpace(value)
			if v == "center" || v == "right" || v == "left" || v == "justify" {
				el.Attrs["data-fox-align"] = v
			}

		case "text-decoration":
			lower := strings.ToLower(value)
			if strings.Contains(lower, "underline") {
				el.Attrs["data-fox-decor"] = "underline"
			} else if strings.Contains(lower, "line-through") {
				el.Attrs["data-fox-decor"] = "line-through"
			}

		case "opacity":
			if f, err := strconv.ParseFloat(strings.TrimSpace(value), 32); err == nil {
				if f < 0.1 {
					el.Attrs["data-fox-hidden"] = "true"
				}
			}
		}
	}
}

// parseFontSize convertit une valeur CSS font-size en pixels (float32).
func parseFontSize(value string) float32 {
	value = strings.TrimSpace(value)

	// Mots-clés
	keywords := map[string]float32{
		"xx-small": 9, "x-small": 11, "small": 13, "medium": 16,
		"large": 18, "x-large": 24, "xx-large": 32, "xxx-large": 48,
		"smaller": 0, "larger": 0, // relatif → on ignore
	}
	if px, ok := keywords[strings.ToLower(value)]; ok {
		return px
	}

	// px
	if strings.HasSuffix(value, "px") {
		if f, err := strconv.ParseFloat(value[:len(value)-2], 32); err == nil {
			return float32(f)
		}
	}

	// pt (1pt ≈ 1.333px)
	if strings.HasSuffix(value, "pt") {
		if f, err := strconv.ParseFloat(value[:len(value)-2], 32); err == nil {
			return float32(f * 1.333)
		}
	}

	// em / rem (base 16px)
	if strings.HasSuffix(value, "rem") {
		if f, err := strconv.ParseFloat(value[:len(value)-3], 32); err == nil {
			return float32(f * 16)
		}
	}
	if strings.HasSuffix(value, "em") {
		if f, err := strconv.ParseFloat(value[:len(value)-2], 32); err == nil {
			return float32(f * 16)
		}
	}

	// % (base 16px)
	if strings.HasSuffix(value, "%") {
		if f, err := strconv.ParseFloat(value[:len(value)-1], 32); err == nil {
			return float32(f * 16 / 100)
		}
	}

	// Nombre seul → pixels
	if f, err := strconv.ParseFloat(value, 32); err == nil {
		return float32(f)
	}

	return 0
}

// getInheritable retourne la valeur héritée si la propriété est définie.
func getInheritable(computed map[string]string, prop, fallback string) string {
	if v, ok := computed[prop]; ok {
		return v
	}
	return fallback
}

// ExtractStyleSheets extrait le contenu des balises <style> dans l'arbre DOM.
func ExtractStyleSheets(root *engine.Element) []string {
	sheets := make([]string, 0, 4)
	engine.WalkElements(root, func(el *engine.Element) bool {
		if el.Tag == "style" && el.Text != "" {
			sheets = append(sheets, el.Text)
		}
		return true
	})
	return sheets
}

var skipStyleTags = map[string]bool{
	"script": true, "style": true, "meta": true, "link": true,
	"head": true, "svg": true, "iframe": true, "template": true,
	"#text": true,
}
