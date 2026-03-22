// Package engine implémente le moteur de rendu HTML de Fox Browser.
package engine

import (
	"strings"

	"golang.org/x/net/html"
)

// Element représente un élément du DOM simplifié.
type Element struct {
	Tag      string
	Text     string
	Attrs    map[string]string
	Children []*Element
}

// Parse transforme du HTML brut en arbre DOM simplifié.
func Parse(htmlContent []byte) (*Element, error) {
	doc, err := html.Parse(strings.NewReader(string(htmlContent)))
	if err != nil {
		return nil, err
	}

	root := &Element{Tag: "root"}
	buildTree(doc, root)
	return root, nil
}

func buildTree(n *html.Node, parent *Element) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.ElementNode:
			el := &Element{
				Tag:   c.Data,
				Attrs: make(map[string]string),
			}
			for _, attr := range c.Attr {
				el.Attrs[attr.Key] = attr.Val
			}
			parent.Children = append(parent.Children, el)
			buildTree(c, el)

		case html.TextNode:
			text := strings.TrimSpace(c.Data)
			if text != "" {
				parent.Text += text
			}
		}
	}
}
