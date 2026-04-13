package engine

import (
	"strings"
)

// FindByID cherche récursivement un élément par son attribut id.
func FindByID(root *Element, id string) *Element {
	if root == nil || id == "" {
		return nil
	}
	if root.Attrs["id"] == id {
		return root
	}
	for _, child := range root.Children {
		if found := FindByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

// FindByClass cherche le premier élément ayant cette classe CSS.
func FindByClass(root *Element, class string) *Element {
	if root == nil || class == "" {
		return nil
	}
	if hasClass(root, class) {
		return root
	}
	for _, child := range root.Children {
		if found := FindByClass(child, class); found != nil {
			return found
		}
	}
	return nil
}

// FindByTag cherche le premier élément avec ce nom de tag.
func FindByTag(root *Element, tag string) *Element {
	if root == nil || tag == "" {
		return nil
	}
	if root.Tag == tag {
		return root
	}
	for _, child := range root.Children {
		if found := FindByTag(child, tag); found != nil {
			return found
		}
	}
	return nil
}

// FindAllByTag cherche tous les éléments avec ce nom de tag.
func FindAllByTag(root *Element, tag string) []*Element {
	results := make([]*Element, 0)
	walkElements(root, func(el *Element) bool {
		if el.Tag == tag {
			results = append(results, el)
		}
		return true
	})
	return results
}

// FindAllByClass cherche tous les éléments ayant cette classe.
func FindAllByClass(root *Element, class string) []*Element {
	results := make([]*Element, 0)
	walkElements(root, func(el *Element) bool {
		if hasClass(el, class) {
			results = append(results, el)
		}
		return true
	})
	return results
}

// QuerySelector implémente un sous-ensemble de CSS selector.
// Supporte : #id, .class, tagname, tagname[attr=val]
func QuerySelector(root *Element, selector string) *Element {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil
	}
	if strings.HasPrefix(selector, "#") {
		return FindByID(root, selector[1:])
	}
	if strings.HasPrefix(selector, ".") {
		return FindByClass(root, selector[1:])
	}
	// tag[attr=val] ou tag[attr]
	if idx := strings.Index(selector, "["); idx > 0 {
		tag := selector[:idx]
		attrPart := selector[idx+1 : len(selector)-1]
		var attr, val string
		if eqIdx := strings.Index(attrPart, "="); eqIdx >= 0 {
			attr = attrPart[:eqIdx]
			val = strings.Trim(attrPart[eqIdx+1:], `"'`)
		} else {
			attr = attrPart
		}
		var result *Element
		walkElements(root, func(el *Element) bool {
			if el.Tag == tag && el.Attrs != nil {
				v, ok := el.Attrs[attr]
				if ok && (val == "" || v == val) {
					result = el
					return false
				}
			}
			return true
		})
		return result
	}
	return FindByTag(root, selector)
}

// QuerySelectorAll retourne tous les éléments correspondant au sélecteur.
func QuerySelectorAll(root *Element, selector string) []*Element {
	selector = strings.TrimSpace(selector)
	results := make([]*Element, 0)
	if selector == "" {
		return results
	}
	if strings.HasPrefix(selector, "#") {
		if el := FindByID(root, selector[1:]); el != nil {
			return append(results, el)
		}
		return results
	}
	if strings.HasPrefix(selector, ".") {
		return FindAllByClass(root, selector[1:])
	}
	return FindAllByTag(root, selector)
}

// ParseFragment parse un fragment HTML et retourne ses enfants dans un élément div.
func ParseFragment(html string) *Element {
	if html == "" {
		return &Element{Tag: "div"}
	}
	root, err := Parse([]byte("<html><body><div id=\"__fox_frag__\">" + html + "</div></body></html>"))
	if err != nil {
		return &Element{Tag: "div", Text: html}
	}
	el := FindByID(root, "__fox_frag__")
	if el == nil {
		return &Element{Tag: "div"}
	}
	return el
}

// CollectText retourne tout le texte d'un sous-arbre.
func CollectText(el *Element) string {
	if el == nil {
		return ""
	}
	var sb strings.Builder
	if el.Text != "" {
		sb.WriteString(el.Text)
	}
	for _, child := range el.Children {
		if child.Tag == "script" || child.Tag == "style" {
			continue
		}
		t := CollectText(child)
		if t != "" {
			if sb.Len() > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(t)
		}
	}
	return strings.TrimSpace(sb.String())
}

// SetHidden marque ou démasque un élément (via data-fox-hidden).
func SetHidden(el *Element, hidden bool) {
	if el.Attrs == nil {
		el.Attrs = make(map[string]string)
	}
	if hidden {
		el.Attrs["data-fox-hidden"] = "true"
	} else {
		delete(el.Attrs, "data-fox-hidden")
	}
}

// AddClass ajoute une classe CSS à un élément.
func AddClass(el *Element, class string) {
	if el.Attrs == nil {
		el.Attrs = make(map[string]string)
	}
	existing := el.Attrs["class"]
	if !hasClass(el, class) {
		if existing == "" {
			el.Attrs["class"] = class
		} else {
			el.Attrs["class"] = existing + " " + class
		}
	}
}

// RemoveClass retire une classe CSS d'un élément.
func RemoveClass(el *Element, class string) {
	if el.Attrs == nil {
		return
	}
	parts := strings.Fields(el.Attrs["class"])
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != class {
			result = append(result, p)
		}
	}
	el.Attrs["class"] = strings.Join(result, " ")
}

// HasClass vérifie si un élément a une classe CSS.
func HasClass(el *Element, class string) bool {
	return hasClass(el, class)
}

// ── helpers internes ──

func hasClass(el *Element, class string) bool {
	if el.Attrs == nil {
		return false
	}
	classes := strings.Fields(el.Attrs["class"])
	for _, c := range classes {
		if c == class {
			return true
		}
	}
	return false
}

// walkElements parcourt l'arbre en profondeur, s'arrête si fn retourne false.
func walkElements(root *Element, fn func(*Element) bool) bool {
	if root == nil {
		return true
	}
	if !fn(root) {
		return false
	}
	for _, child := range root.Children {
		if !walkElements(child, fn) {
			return false
		}
	}
	return true
}
