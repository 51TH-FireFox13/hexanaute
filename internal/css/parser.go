// Package css implémente un moteur CSS basique pour Fox Browser.
// Objectif : souveraineté visuelle — appliquer les styles sans dépendance GAFAM.
//
// Supporte :
//   - Sélecteurs : tag, #id, .class, tag.class, *, descendant (a b), virgule (a, b)
//   - Propriétés : color, background-color, font-*, display, visibility, text-*
//   - Valeurs : hex, rgb(), rgba(), noms CSS, px/em/rem, keywords
//   - At-rules : @media, @keyframes, @charset, @import → ignorés
//
// Non supporté (v0.3.0) : pseudo-classes, ::before/::after, calc(), var()
package css

import (
	"strings"
)

// Rule représente une règle CSS (sélecteur + propriétés).
type Rule struct {
	Selector    string
	Properties  map[string]string
	Specificity int // pour l'ordre de cascade
}

// StyleSheet contient les règles parsées d'une feuille de style.
type StyleSheet struct {
	Rules []*Rule
}

// ParseSheet parse le contenu d'une balise <style>.
func ParseSheet(cssText string) *StyleSheet {
	cssText = removeComments(cssText)
	rules := make([]*Rule, 0, 32)

	i := 0
	for i < len(cssText) {
		// Sauter les espaces
		i = skipSpaces(cssText, i)
		if i >= len(cssText) {
			break
		}

		// At-rule (@media, @keyframes, @import, @charset...)
		if cssText[i] == '@' {
			i = skipAtRule(cssText, i)
			continue
		}

		// Trouver le début des déclarations '{'
		braceStart := strings.IndexByte(cssText[i:], '{')
		if braceStart < 0 {
			break
		}

		selectorText := strings.TrimSpace(cssText[i : i+braceStart])
		i += braceStart + 1

		// Trouver la fin des déclarations '}'
		// (en tenant compte des accolades imbriquées)
		braceEnd, depth := i, 1
		for braceEnd < len(cssText) && depth > 0 {
			if cssText[braceEnd] == '{' {
				depth++
			} else if cssText[braceEnd] == '}' {
				depth--
			}
			if depth > 0 {
				braceEnd++
			}
		}

		declarationsText := cssText[i:braceEnd]
		i = braceEnd + 1

		if selectorText == "" {
			continue
		}

		// Parser les propriétés
		props := ParseDeclarations(declarationsText)
		if len(props) == 0 {
			continue
		}

		// Séparer les sélecteurs par virgule
		for _, sel := range strings.Split(selectorText, ",") {
			sel = normalizeSelector(sel)
			if sel == "" {
				continue
			}
			rules = append(rules, &Rule{
				Selector:    sel,
				Properties:  props,
				Specificity: computeSpecificity(sel),
			})
		}
	}

	return &StyleSheet{Rules: rules}
}

// ParseDeclarations parse un bloc de déclarations CSS en map.
// Exemple : "color: red; font-size: 16px; display: none"
func ParseDeclarations(text string) map[string]string {
	props := make(map[string]string)
	for _, decl := range strings.Split(text, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		colon := strings.IndexByte(decl, ':')
		if colon < 0 {
			continue
		}
		prop := strings.TrimSpace(strings.ToLower(decl[:colon]))
		value := strings.TrimSpace(decl[colon+1:])
		// Retirer !important (avec ou sans espace avant : "red !important" ou "red!important")
		if idx := strings.Index(value, "!important"); idx >= 0 {
			value = strings.TrimSpace(value[:idx])
		}
		if prop != "" && value != "" {
			props[prop] = value
		}
	}
	return props
}

// ParseInlineStyle parse un attribut style="..." en map de propriétés.
func ParseInlineStyle(style string) map[string]string {
	return ParseDeclarations(style)
}

// ── Sélecteur matching ──────────────────────────────────────────────────────

// MatchesSelector vérifie si un élément correspond au sélecteur CSS.
// ancestors est la liste des ancêtres de l'élément (parent en dernier).
func MatchesSelector(tag string, id string, classes []string, ancestors []AncestorInfo, selector string) bool {
	// Sélecteur descendant : "div p" → plusieurs parties séparées par espace
	parts := splitDescendantSelector(selector)
	if len(parts) == 0 {
		return false
	}

	// La dernière partie doit correspondre à l'élément courant
	if !matchSimpleSelector(tag, id, classes, parts[len(parts)-1]) {
		return false
	}

	// Pour un sélecteur simple (1 partie), c'est suffisant
	if len(parts) == 1 {
		return true
	}

	// Vérifier les ancêtres dans l'ordre (de la droite vers la gauche)
	return matchAncestors(ancestors, parts[:len(parts)-1])
}

// AncestorInfo contient les infos d'un élément ancêtre.
type AncestorInfo struct {
	Tag     string
	ID      string
	Classes []string
}

func matchSimpleSelector(tag, id string, classes []string, sel string) bool {
	sel = strings.TrimSpace(sel)
	if sel == "" || sel == "*" {
		return true
	}

	// Décomposer le sélecteur composé : "div.class#id"
	// On gère : tag, #id, .class, tag.class, tag#id, .class1.class2
	remaining := sel

	// Extraire le tag (commence par lettre ou est *)
	expectedTag := ""
	if remaining != "" && remaining[0] != '#' && remaining[0] != '.' {
		end := 0
		for end < len(remaining) && remaining[end] != '#' && remaining[end] != '.' && remaining[end] != '[' && remaining[end] != ':' {
			end++
		}
		expectedTag = remaining[:end]
		remaining = remaining[end:]
	}

	// Extraire les ID
	expectedIDs := make([]string, 0)
	for strings.HasPrefix(remaining, "#") {
		remaining = remaining[1:]
		end := 0
		for end < len(remaining) && remaining[end] != '#' && remaining[end] != '.' && remaining[end] != '[' && remaining[end] != ':' {
			end++
		}
		expectedIDs = append(expectedIDs, remaining[:end])
		remaining = remaining[end:]
	}

	// Extraire les classes
	expectedClasses := make([]string, 0)
	for strings.HasPrefix(remaining, ".") {
		remaining = remaining[1:]
		end := 0
		for end < len(remaining) && remaining[end] != '#' && remaining[end] != '.' && remaining[end] != '[' && remaining[end] != ':' {
			end++
		}
		expectedClasses = append(expectedClasses, remaining[:end])
		remaining = remaining[end:]
	}

	// Vérifications
	if expectedTag != "" && expectedTag != "*" && tag != expectedTag {
		return false
	}
	for _, eid := range expectedIDs {
		if id != eid {
			return false
		}
	}
	for _, ec := range expectedClasses {
		found := false
		for _, c := range classes {
			if c == ec {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func matchAncestors(ancestors []AncestorInfo, parts []string) bool {
	if len(parts) == 0 {
		return true
	}
	// Chercher si au moins un ancêtre correspond à la première partie
	// puis récursivement pour les parties suivantes
	for i, anc := range ancestors {
		if matchSimpleSelector(anc.Tag, anc.ID, anc.Classes, parts[len(parts)-1]) {
			if len(parts) == 1 {
				return true
			}
			// Vérifier le reste dans les ancêtres restants
			if matchAncestors(ancestors[:i], parts[:len(parts)-1]) {
				return true
			}
		}
	}
	return false
}

// splitDescendantSelector divise "div p span" en ["div", "p", "span"].
// Ignore les combinateurs >, +, ~ (traités comme descendant simple).
func splitDescendantSelector(sel string) []string {
	// Simplification : traiter >, +, ~ comme des espaces
	sel = strings.ReplaceAll(sel, ">", " ")
	sel = strings.ReplaceAll(sel, "+", " ")
	sel = strings.ReplaceAll(sel, "~", " ")
	// Retirer les pseudo-classes (:hover, :focus, ::before, etc.)
	for {
		colonIdx := strings.IndexByte(sel, ':')
		if colonIdx < 0 {
			break
		}
		// Trouver la fin du pseudo-sélecteur
		end := colonIdx + 1
		for end < len(sel) && sel[end] != ' ' && sel[end] != '#' && sel[end] != '.' {
			end++
		}
		sel = sel[:colonIdx] + sel[end:]
	}
	parts := strings.Fields(sel)
	filtered := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// computeSpecificity calcule la spécificité d'un sélecteur CSS (a=ID, b=class, c=tag).
func computeSpecificity(sel string) int {
	parts := splitDescendantSelector(sel)
	a, b, c := 0, 0, 0
	for _, part := range parts {
		remaining := part
		// Tag
		if remaining != "" && remaining[0] != '#' && remaining[0] != '.' {
			end := 0
			for end < len(remaining) && remaining[end] != '#' && remaining[end] != '.' {
				end++
			}
			if remaining[:end] != "" && remaining[:end] != "*" {
				c++
			}
			remaining = remaining[end:]
		}
		// IDs
		for strings.HasPrefix(remaining, "#") {
			a++
			remaining = remaining[1:]
			for len(remaining) > 0 && remaining[0] != '#' && remaining[0] != '.' {
				remaining = remaining[1:]
			}
		}
		// Classes
		for strings.HasPrefix(remaining, ".") {
			b++
			remaining = remaining[1:]
			for len(remaining) > 0 && remaining[0] != '#' && remaining[0] != '.' {
				remaining = remaining[1:]
			}
		}
	}
	return a*100 + b*10 + c
}

// ── Utilitaires ──────────────────────────────────────────────────────────────

func removeComments(css string) string {
	for {
		start := strings.Index(css, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(css[start+2:], "*/")
		if end < 0 {
			css = css[:start]
			break
		}
		css = css[:start] + " " + css[start+2+end+2:]
	}
	return css
}

func skipSpaces(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return i
}

func skipAtRule(css string, start int) int {
	i := start
	// Chercher jusqu'à ';' ou '{...}'
	for i < len(css) {
		if css[i] == ';' {
			return i + 1
		}
		if css[i] == '{' {
			depth := 1
			i++
			for i < len(css) && depth > 0 {
				if css[i] == '{' {
					depth++
				} else if css[i] == '}' {
					depth--
				}
				i++
			}
			return i
		}
		i++
	}
	return i
}

func normalizeSelector(sel string) string {
	// Normaliser les espaces
	sel = strings.TrimSpace(sel)
	// Remplacer les espaces multiples par un seul
	for strings.Contains(sel, "  ") {
		sel = strings.ReplaceAll(sel, "  ", " ")
	}
	return sel
}
