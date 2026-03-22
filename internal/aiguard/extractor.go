package aiguard

import (
	"strings"

	"golang.org/x/net/html"
)

// ExtractScripts extrait tous les blocs <script> inline d'un document HTML.
func ExtractScripts(htmlContent []byte) []string {
	scripts := make([]string, 0, 8)

	tokenizer := html.NewTokenizer(strings.NewReader(string(htmlContent)))
	inScript := false

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return scripts
		case html.StartTagToken:
			tn, _ := tokenizer.TagName()
			if string(tn) == "script" {
				// Vérifier si c'est un script inline (pas de src externe)
				hasSource := false
				for {
					key, val, more := tokenizer.TagAttr()
					if string(key) == "src" && len(val) > 0 {
						hasSource = true
					}
					if !more {
						break
					}
				}
				if !hasSource {
					inScript = true
				}
			}
		case html.TextToken:
			if inScript {
				text := strings.TrimSpace(string(tokenizer.Text()))
				if len(text) > 0 {
					scripts = append(scripts, text)
				}
			}
		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			if string(tn) == "script" {
				inScript = false
			}
		}
	}
}
