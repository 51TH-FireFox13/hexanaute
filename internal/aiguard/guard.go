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
)

// AnalysisResult est le résultat de l'analyse IA d'un script.
type AnalysisResult struct {
	Score    float32        // 0.0 (sûr) à 1.0 (malveillant)
	Category ThreatCategory
	Details  string
	Blocked  bool
}

// Guard est le gardien IA du navigateur.
type Guard struct {
	threshold float32 // score au-dessus duquel on bloque
	npuAvail  bool    // NPU détecté et disponible
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
	}
}

// AnalyzeScript analyse un script JavaScript et retourne un score de risque.
// TODO: intégrer ONNX Runtime pour l'inférence réelle sur NPU.
func (g *Guard) AnalyzeScript(scriptContent []byte, sourceURL string) *AnalysisResult {
	// Phase 0 : analyse heuristique basique
	// Sera remplacé par l'inférence ONNX sur NPU en Phase 1
	result := &AnalysisResult{
		Score:    0.0,
		Category: ThreatNone,
	}

	// Heuristiques simples en attendant le modèle
	content := string(scriptContent)
	patterns := map[string]ThreatCategory{
		"eval(atob(":    ThreatObfuscated,
		"document.cookie": ThreatExfiltration,
		"CoinHive":      ThreatCryptoMiner,
		"keydown":       ThreatKeylogger, // faux positif possible, le modèle IA fera mieux
	}

	for pattern, category := range patterns {
		if containsPattern(content, pattern) {
			result.Score += 0.3
			result.Category = category
		}
	}

	if result.Score > 1.0 {
		result.Score = 1.0
	}

	result.Blocked = result.Score >= g.threshold
	return result
}

func containsPattern(content, pattern string) bool {
	for i := 0; i <= len(content)-len(pattern); i++ {
		if content[i:i+len(pattern)] == pattern {
			return true
		}
	}
	return false
}

// detectNPU tente de détecter un NPU disponible.
// TODO: implémenter la détection réelle via sysfs (Linux) ou DirectML (Windows).
func detectNPU() bool {
	// Stub — sera implémenté avec les bindings ONNX Runtime
	return false
}
