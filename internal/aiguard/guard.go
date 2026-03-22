// Package aiguard implémente la couche de sécurité IA du navigateur.
// Il analyse le JavaScript et le contenu web via un modèle local
// exécuté sur NPU (via ONNX Runtime) pour détecter les menaces.
package aiguard

import "fmt"

// ThreatCategory catégorise le type de menace détectée.
type ThreatCategory string

const (
	ThreatNone         ThreatCategory = "safe"
	ThreatPhishing     ThreatCategory = "phishing"
	ThreatCryptoMiner  ThreatCategory = "cryptominer"
	ThreatKeylogger    ThreatCategory = "keylogger"
	ThreatExfiltration ThreatCategory = "exfiltration"
	ThreatObfuscated   ThreatCategory = "obfuscated"
	ThreatRedirect     ThreatCategory = "redirect"
	ThreatFingerprint  ThreatCategory = "fingerprint"
	ThreatFormHijack   ThreatCategory = "formhijack"
)

// AnalysisResult est le résultat de l'analyse IA d'un script.
type AnalysisResult struct {
	Score    float32        // 0.0 (sûr) à 1.0 (malveillant)
	Category ThreatCategory
	Details  string
	Blocked  bool
	Alerts   []Alert
}

// Alert est une alerte individuelle détectée.
type Alert struct {
	Category ThreatCategory
	Pattern  string
	Score    float32
	Context  string
}

// Guard est le gardien IA du navigateur.
type Guard struct {
	threshold float32
	npuAvail  bool
	analyzer  *ScriptAnalyzer
	domGuard  *DOMAnalyzer
}

// NewGuard crée un nouveau gardien IA.
func NewGuard(threshold float32) *Guard {
	npu := detectNPU()
	if npu {
		fmt.Println("[AI Guard] NPU détecté — inférence matérielle activée")
	} else {
		fmt.Println("[AI Guard] Pas de NPU — fallback CPU")
	}

	return &Guard{
		threshold: threshold,
		npuAvail:  npu,
		analyzer:  NewScriptAnalyzer(),
		domGuard:  NewDOMAnalyzer(),
	}
}

// AnalyzePage analyse une page complète (HTML + scripts) et retourne un score de risque.
func (g *Guard) AnalyzePage(htmlContent []byte, sourceURL string) *AnalysisResult {
	result := &AnalysisResult{
		Score:    0.0,
		Category: ThreatNone,
		Alerts:   make([]Alert, 0),
	}

	// 1. Extraire et analyser les scripts inline
	scripts := ExtractScripts(htmlContent)
	for _, script := range scripts {
		alerts := g.analyzer.Analyze(script, sourceURL)
		result.Alerts = append(result.Alerts, alerts...)
	}

	// 2. Analyser le DOM pour le phishing et le formjacking
	domAlerts := g.domGuard.Analyze(htmlContent, sourceURL)
	result.Alerts = append(result.Alerts, domAlerts...)

	// 3. Calculer le score final pondéré
	if len(result.Alerts) > 0 {
		var maxScore float32
		maxCategory := ThreatNone

		for _, alert := range result.Alerts {
			result.Score += alert.Score
			if alert.Score > maxScore {
				maxScore = alert.Score
				maxCategory = alert.Category
			}
		}

		// Plafonner à 1.0
		if result.Score > 1.0 {
			result.Score = 1.0
		}

		result.Category = maxCategory

		// Construire le détail
		details := ""
		for _, alert := range result.Alerts {
			if details != "" {
				details += " | "
			}
			details += fmt.Sprintf("%s: %s", alert.Category, alert.Pattern)
		}
		result.Details = details
	}

	result.Blocked = result.Score >= g.threshold
	return result
}

// AnalyzeScript — rétrocompatibilité, redirige vers AnalyzePage.
func (g *Guard) AnalyzeScript(content []byte, sourceURL string) *AnalysisResult {
	return g.AnalyzePage(content, sourceURL)
}

// detectNPU tente de détecter un NPU disponible.
// TODO: implémenter la détection réelle via sysfs (Linux) ou DirectML (Windows).
func detectNPU() bool {
	return false
}
