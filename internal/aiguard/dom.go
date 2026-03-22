package aiguard

import (
	"strings"

	"golang.org/x/net/html"
)

// DOMAnalyzer analyse la structure HTML pour détecter les menaces.
type DOMAnalyzer struct{}

// NewDOMAnalyzer crée un analyseur DOM.
func NewDOMAnalyzer() *DOMAnalyzer {
	return &DOMAnalyzer{}
}

// Analyze analyse le DOM d'une page pour détecter le phishing et les attaques visuelles.
func (d *DOMAnalyzer) Analyze(htmlContent []byte, sourceURL string) []Alert {
	alerts := make([]Alert, 0)

	info := d.extractDOMInfo(htmlContent)

	// 1. Faux formulaire de login : input password + action externe
	if info.hasPasswordField {
		if info.formAction != "" && isExternalURL(info.formAction, sourceURL) {
			alerts = append(alerts, Alert{
				Category: ThreatPhishing,
				Pattern:  "password form posts to external URL",
				Score:    0.6,
				Context:  "action=" + info.formAction,
			})
		}
	}

	// 2. iframe caché (clickjacking)
	if info.hiddenIframes > 0 {
		alerts = append(alerts, Alert{
			Category: ThreatPhishing,
			Pattern:  "hidden iframe detected",
			Score:    0.3,
		})
	}

	// 3. Nombre excessif d'inputs cachés + formulaire vers site externe = suspect
	if info.hiddenInputs > 10 && info.formAction != "" && isExternalURL(info.formAction, sourceURL) {
		alerts = append(alerts, Alert{
			Category: ThreatFormHijack,
			Pattern:  "excessive hidden inputs with external form",
			Score:    0.3,
		})
	}

	// 4. Scripts externes depuis des domaines suspects
	for _, src := range info.externalScripts {
		if isSuspiciousDomain(src) {
			alerts = append(alerts, Alert{
				Category: ThreatObfuscated,
				Pattern:  "script from suspicious domain",
				Score:    0.4,
				Context:  src,
			})
		}
	}

	// 5. Meta refresh redirect
	if info.metaRefresh != "" {
		alerts = append(alerts, Alert{
			Category: ThreatRedirect,
			Pattern:  "meta refresh redirect",
			Score:    0.2,
			Context:  info.metaRefresh,
		})
	}

	// 6. Data URI dans des éléments sensibles
	if info.dataURIs > 0 {
		alerts = append(alerts, Alert{
			Category: ThreatObfuscated,
			Pattern:  "data URI in page elements",
			Score:    0.15,
		})
	}

	return alerts
}

type domInfo struct {
	hasPasswordField bool
	formAction       string
	hiddenIframes    int
	hiddenInputs     int
	externalScripts  []string
	metaRefresh      string
	dataURIs         int
}

func (d *DOMAnalyzer) extractDOMInfo(htmlContent []byte) domInfo {
	info := domInfo{
		externalScripts: make([]string, 0),
	}

	tokenizer := html.NewTokenizer(strings.NewReader(string(htmlContent)))

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return info

		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := tokenizer.TagName()
			tag := string(tn)

			if !hasAttr {
				continue
			}

			attrs := extractAttrs(tokenizer)

			switch tag {
			case "input":
				inputType := strings.ToLower(attrs["type"])
				if inputType == "password" {
					info.hasPasswordField = true
				}
				if inputType == "hidden" {
					info.hiddenInputs++
				}

			case "form":
				if action, ok := attrs["action"]; ok {
					info.formAction = action
				}

			case "iframe":
				style := strings.ToLower(attrs["style"])
				width := attrs["width"]
				height := attrs["height"]
				if strings.Contains(style, "display:none") ||
					strings.Contains(style, "visibility:hidden") ||
					strings.Contains(style, "opacity:0") ||
					width == "0" || height == "0" {
					info.hiddenIframes++
				}

			case "script":
				if src, ok := attrs["src"]; ok {
					info.externalScripts = append(info.externalScripts, src)
				}

			case "meta":
				httpEquiv := strings.ToLower(attrs["http-equiv"])
				if httpEquiv == "refresh" {
					info.metaRefresh = attrs["content"]
				}

			case "a", "img":
				for _, v := range attrs {
					if strings.HasPrefix(v, "data:") {
						info.dataURIs++
					}
				}
			}
		}
	}
}

func extractAttrs(tokenizer *html.Tokenizer) map[string]string {
	attrs := make(map[string]string)
	for {
		key, val, more := tokenizer.TagAttr()
		if len(key) > 0 {
			attrs[string(key)] = string(val)
		}
		if !more {
			break
		}
	}
	return attrs
}

// isExternalURL vérifie si une URL pointe vers un domaine différent.
func isExternalURL(href, pageURL string) bool {
	if strings.HasPrefix(href, "/") || strings.HasPrefix(href, "#") || href == "" {
		return false
	}
	if !strings.Contains(href, "://") {
		return false
	}

	pageDomain := extractDomain(pageURL)
	hrefDomain := extractDomain(href)

	if pageDomain == "" || hrefDomain == "" {
		return false
	}

	return !strings.HasSuffix(hrefDomain, pageDomain) && !strings.HasSuffix(pageDomain, hrefDomain)
}

func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	parts := strings.SplitN(url, "/", 2)
	if len(parts) > 0 {
		domain := strings.SplitN(parts[0], ":", 2)[0]
		return strings.ToLower(domain)
	}
	return ""
}

// isSuspiciousDomain vérifie si un domaine de script est dans une liste suspecte.
func isSuspiciousDomain(src string) bool {
	suspicious := []string{
		"coinhive.com", "coin-hive.com", "crypto-loot.com",
		"coinimp.com", "webminepool.com", "ppoi.org",
		"authedmine.com", "minero.cc", "cloudcoins.co",
		".tk/", ".ml/", ".ga/", ".cf/",
		"evil.", "malware.", "exploit.",
	}

	srcLower := strings.ToLower(src)
	for _, s := range suspicious {
		if strings.Contains(srcLower, s) {
			return true
		}
	}
	return false
}
