package aiguard

import (
	"regexp"
	"strings"
)

// ScriptAnalyzer analyse le JavaScript pour détecter les menaces.
type ScriptAnalyzer struct {
	rules []Rule
}

// Rule est une règle de détection.
type Rule struct {
	Name     string
	Category ThreatCategory
	Score    float32
	Match    func(script string) bool
	// Requires : si défini, le pattern ne compte que si un autre signal est aussi présent
	Requires func(script string) bool
}

// NewScriptAnalyzer crée un analyseur de scripts avec les règles de détection.
func NewScriptAnalyzer() *ScriptAnalyzer {
	a := &ScriptAnalyzer{}
	a.rules = a.buildRules()
	return a
}

func (a *ScriptAnalyzer) buildRules() []Rule {
	return []Rule{
		// ══════════════════════════════════════
		// OBFUSCATION — code volontairement masqué
		// ══════════════════════════════════════
		{
			Name:     "eval(atob(...))",
			Category: ThreatObfuscated,
			Score:    0.5,
			Match:    regexMatch(`eval\s*\(\s*atob\s*\(`),
		},
		{
			Name:     "eval(unescape(...))",
			Category: ThreatObfuscated,
			Score:    0.4,
			Match:    regexMatch(`eval\s*\(\s*unescape\s*\(`),
		},
		{
			Name:     "eval + String.fromCharCode",
			Category: ThreatObfuscated,
			Score:    0.5,
			Match: func(s string) bool {
				return strings.Contains(s, "eval") && strings.Contains(s, "String.fromCharCode")
			},
		},
		{
			Name:     "document.write(unescape(...))",
			Category: ThreatObfuscated,
			Score:    0.4,
			Match:    regexMatch(`document\.write\s*\(\s*unescape\s*\(`),
		},
		{
			Name:     "long hex/unicode escape sequences",
			Category: ThreatObfuscated,
			Score:    0.3,
			Match:    regexMatch(`(\\x[0-9a-fA-F]{2}){15,}`),
		},
		{
			Name:     "long base64 string (>500 chars)",
			Category: ThreatObfuscated,
			Score:    0.2,
			Match:    regexMatch(`['"][A-Za-z0-9+/=]{500,}['"]`),
		},
		{
			Name:     "heavy char code construction",
			Category: ThreatObfuscated,
			Score:    0.4,
			Match: func(s string) bool {
				// Plus de 10 String.fromCharCode dans un même script
				return strings.Count(s, "String.fromCharCode") > 10
			},
		},
		{
			Name:     "Function constructor eval",
			Category: ThreatObfuscated,
			Score:    0.5,
			Match:    regexMatch(`new\s+Function\s*\([^)]*atob`),
		},

		// ══════════════════════════════════════
		// EXFILTRATION — vol de données
		// ══════════════════════════════════════
		{
			Name:     "cookie exfiltration via image",
			Category: ThreatExfiltration,
			Score:    0.7,
			Match: func(s string) bool {
				return strings.Contains(s, "document.cookie") &&
					(strings.Contains(s, "new Image") || strings.Contains(s, "img.src"))
			},
		},
		{
			Name:     "cookie sent via fetch/XHR",
			Category: ThreatExfiltration,
			Score:    0.6,
			Match: func(s string) bool {
				hasCookie := strings.Contains(s, "document.cookie")
				hasSend := strings.Contains(s, "fetch(") ||
					strings.Contains(s, ".send(") ||
					strings.Contains(s, "XMLHttpRequest")
				return hasCookie && hasSend
			},
		},
		{
			Name:     "localStorage exfiltration",
			Category: ThreatExfiltration,
			Score:    0.5,
			Match: func(s string) bool {
				hasStorage := strings.Contains(s, "localStorage") || strings.Contains(s, "sessionStorage")
				hasSend := strings.Contains(s, "fetch(") || strings.Contains(s, ".send(")
				return hasStorage && hasSend
			},
		},
		{
			Name:     "navigator/screen data collection",
			Category: ThreatFingerprint,
			Score:    0.15,
			Match: func(s string) bool {
				signals := 0
				fps := []string{
					"navigator.userAgent", "navigator.platform", "navigator.language",
					"screen.width", "screen.height", "screen.colorDepth",
					"navigator.plugins", "navigator.hardwareConcurrency",
					"canvas.toDataURL", "getContext('webgl')", "AudioContext",
				}
				for _, fp := range fps {
					if strings.Contains(s, fp) {
						signals++
					}
				}
				// 3+ signaux de fingerprinting = suspect
				return signals >= 3
			},
		},

		// ══════════════════════════════════════
		// KEYLOGGER — capture de frappe
		// ══════════════════════════════════════
		{
			Name:     "keylogger pattern",
			Category: ThreatKeylogger,
			Score:    0.6,
			Match: func(s string) bool {
				// keydown/keypress + envoi de données = keylogger
				hasKeyCapture := strings.Contains(s, "keydown") ||
					strings.Contains(s, "keypress") ||
					strings.Contains(s, "keyup")
				hasSend := strings.Contains(s, "fetch(") ||
					strings.Contains(s, ".send(") ||
					strings.Contains(s, "new Image") ||
					strings.Contains(s, "navigator.sendBeacon")
				return hasKeyCapture && hasSend
			},
		},
		// Note : keydown SEUL n'est plus un signal — c'est normal pour tout site interactif

		// ══════════════════════════════════════
		// CRYPTO-MINING
		// ══════════════════════════════════════
		{
			Name:     "CoinHive miner",
			Category: ThreatCryptoMiner,
			Score:    0.9,
			Match: func(s string) bool {
				return strings.Contains(s, "CoinHive") || strings.Contains(s, "coinhive.min.js")
			},
		},
		{
			Name:     "crypto mining patterns",
			Category: ThreatCryptoMiner,
			Score:    0.7,
			Match: func(s string) bool {
				miners := []string{
					"cryptonight", "stratum+tcp", "hashrate",
					"CryptoLoot", "deepMiner", "mineralt",
					"coinimp.com", "crypto-loot.com", "webminepool",
				}
				for _, m := range miners {
					if strings.Contains(strings.ToLower(s), strings.ToLower(m)) {
						return true
					}
				}
				return false
			},
		},
		{
			Name:     "WebWorker + WASM mining",
			Category: ThreatCryptoMiner,
			Score:    0.4,
			Match: func(s string) bool {
				hasWorker := strings.Contains(s, "new Worker") || strings.Contains(s, "SharedWorker")
				hasWasm := strings.Contains(s, "WebAssembly")
				hasLoop := strings.Contains(s, "while(true)") || strings.Contains(s, "for(;;)")
				return hasWorker && hasWasm && hasLoop
			},
		},

		// ══════════════════════════════════════
		// REDIRECT — redirections suspectes
		// ══════════════════════════════════════
		{
			Name:     "suspicious redirect",
			Category: ThreatRedirect,
			Score:    0.3,
			Match: func(s string) bool {
				hasRedirect := strings.Contains(s, "location.replace") ||
					strings.Contains(s, "location.assign")
				// Redirect + obfuscation = suspect
				hasObfusc := strings.Contains(s, "atob") ||
					strings.Contains(s, "unescape") ||
					strings.Contains(s, "String.fromCharCode")
				return hasRedirect && hasObfusc
			},
		},

		// ══════════════════════════════════════
		// FORM HIJACKING — vol de formulaires
		// ══════════════════════════════════════
		{
			Name:     "form action hijack",
			Category: ThreatFormHijack,
			Score:    0.6,
			Match: func(s string) bool {
				return (strings.Contains(s, ".action") || strings.Contains(s, "setAttribute('action'")) &&
					(strings.Contains(s, "http://") || strings.Contains(s, "https://"))
			},
		},
		{
			Name:     "input value interception",
			Category: ThreatFormHijack,
			Score:    0.5,
			Match: func(s string) bool {
				hasInput := strings.Contains(s, "querySelector('input[type=\"password\"]'") ||
					strings.Contains(s, "getElementsByName('password')") ||
					strings.Contains(s, "getElementById('password')")
				hasSend := strings.Contains(s, "fetch(") || strings.Contains(s, ".send(")
				return hasInput && hasSend
			},
		},
	}
}

// Analyze analyse un script et retourne les alertes.
func (a *ScriptAnalyzer) Analyze(script string, sourceURL string) []Alert {
	alerts := make([]Alert, 0)

	for _, rule := range a.rules {
		if rule.Match(script) {
			// Vérifier la condition requise si elle existe
			if rule.Requires != nil && !rule.Requires(script) {
				continue
			}
			alerts = append(alerts, Alert{
				Category: rule.Category,
				Pattern:  rule.Name,
				Score:    rule.Score,
			})
		}
	}

	// Bonus de contexte : plusieurs signaux du même type = amplification
	categoryCount := make(map[ThreatCategory]int)
	for i := range alerts {
		categoryCount[alerts[i].Category]++
	}
	for cat, count := range categoryCount {
		if count >= 3 {
			alerts = append(alerts, Alert{
				Category: cat,
				Pattern:  "multi-signal amplification",
				Score:    0.2,
			})
		}
	}

	return alerts
}

// regexMatch crée une fonction de match basée sur un regex.
func regexMatch(pattern string) func(string) bool {
	re := regexp.MustCompile(pattern)
	return func(s string) bool {
		return re.MatchString(s)
	}
}
