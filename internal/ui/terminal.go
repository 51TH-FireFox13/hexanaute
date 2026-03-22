// Package ui gère l'affichage du navigateur Fox en mode terminal.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/51TH-FireFox13/fox-browser/internal/browser"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorBlue   = "\033[34m"
)

var scanner = bufio.NewScanner(os.Stdin)

// StatusBar affiche la barre d'état du navigateur.
func StatusBar(url string, status int, tls string, duration string) {
	lock := "🔓"
	statusColor := colorGreen
	if strings.HasPrefix(url, "https") {
		lock = "🔒"
	}
	if status >= 400 {
		statusColor = colorRed
	}

	fmt.Printf("\n%s┌─── Fox Browser ──────────────────────────────────────┐%s\n", colorCyan, colorReset)
	fmt.Printf("%s│%s %s %s%s%s", colorCyan, colorReset, lock, colorBold, url, colorReset)
	padding := 51 - len(url) - 4
	if padding < 0 {
		padding = 0
	}
	fmt.Printf("%*s%s│%s\n", padding, "", colorCyan, colorReset)
	fmt.Printf("%s│%s %sHTTP %d%s | %s | %s", colorCyan, colorReset, statusColor, status, colorReset, tls, duration)
	infoLen := len(fmt.Sprintf("HTTP %d | %s | %s", status, tls, duration))
	padding2 := 51 - infoLen - 1
	if padding2 < 0 {
		padding2 = 0
	}
	fmt.Printf("%*s%s│%s\n", padding2, "", colorCyan, colorReset)
	fmt.Printf("%s└──────────────────────────────────────────────────────┘%s\n\n", colorCyan, colorReset)
}

// AIGuardStatus affiche le résultat de l'analyse IA.
func AIGuardStatus(score float32, category string, blocked bool, details string) {
	color := colorGreen
	icon := "✓"
	label := "SÛR"

	if score > 0.3 {
		color = colorYellow
		icon = "⚠"
		label = "SUSPECT"
	}
	if score > 0.7 {
		color = colorRed
		icon = "✗"
		label = "DANGEREUX"
	}
	if blocked {
		color = colorRed
		icon = "🛡"
		label = "BLOQUÉ"
	}

	fmt.Printf("%s[AI Guard] %s %s (score: %.1f%%, catégorie: %s)%s\n",
		color, icon, label, score*100, category, colorReset)

	if details != "" && score > 0.1 {
		fmt.Printf("%s           détails: %s%s\n", colorGray, details, colorReset)
	}
	fmt.Println()
}

// PageContent affiche le contenu de la page.
func PageContent(content string) {
	fmt.Println(content)
}

// PromptWithState affiche le prompt avec indicateurs de navigation.
func PromptWithState(hasPage, canBack, canForward bool, histLen int) string {
	nav := ""
	if hasPage {
		if canBack {
			nav += "◀"
		}
		if canForward {
			nav += "▶"
		}
		if histLen > 0 {
			nav += fmt.Sprintf(" [%d]", histLen)
		}
	}

	if nav != "" {
		fmt.Printf("\n%s%s fox>%s ", colorGray, nav, colorReset)
	} else {
		fmt.Printf("\n%sfox>%s ", colorCyan, colorReset)
	}

	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// ShowLinks affiche la liste des liens numérotés.
func ShowLinks(links []browser.PageLink) {
	if len(links) == 0 {
		return
	}
	fmt.Printf("\n%s── Liens ──%s\n", colorCyan, colorReset)
	for _, l := range links {
		text := l.Text
		if len(text) > 50 {
			text = text[:47] + "..."
		}
		url := l.URL
		if len(url) > 60 {
			url = url[:57] + "..."
		}
		fmt.Printf("  %s[%d]%s %s %s→ %s%s\n",
			colorCyan, l.Index, colorReset,
			text,
			colorGray, url, colorReset)
	}
	fmt.Println()
}

// ShowHistory affiche l'historique de navigation.
func ShowHistory(history []browser.Page) {
	if len(history) == 0 {
		fmt.Println("[Fox] Historique vide.")
		return
	}
	fmt.Printf("\n%s── Historique ──%s\n", colorCyan, colorReset)
	for i, p := range history {
		url := p.URL
		if len(url) > 60 {
			url = url[:57] + "..."
		}
		marker := "  "
		if i == len(history)-1 {
			marker = "▸ "
		}
		fmt.Printf("  %s%s%d.%s %s\n", marker, colorGray, i+1, colorReset, url)
	}
	fmt.Println()
}

// Banner affiche la bannière de démarrage.
func Banner(version string) {
	fmt.Printf(`
%s    ╔═══════════════════════════════════╗
    ║   🦊  FOX BROWSER  v%s      ║
    ║   Navigateur Souverain            ║
    ╚═══════════════════════════════════╝%s
`, colorCyan, version, colorReset)
	fmt.Printf("    %sMoteur: FoxEngine | Sécurité: AI Guard%s\n", colorGray, colorReset)
	fmt.Printf("    %sTapez une URL ou '?' pour les commandes%s\n\n", colorGray, colorReset)
}

// Help affiche l'aide basique (rétrocompat).
func Help() {
	HelpFull()
}

// HelpFull affiche l'aide complète.
func HelpFull() {
	fmt.Printf(`
%sNavigation:%s
  %s<url>%s           Naviguer vers une URL
  %s<numéro>%s        Suivre un lien numéroté
  %sb%s / %sretour%s      Page précédente
  %sf%s / %savancer%s     Page suivante
  %sr%s / %srecharger%s   Recharger la page

%sInformations:%s
  %sl%s / %sliens%s        Lister les liens de la page
  %sh%s / %shistorique%s   Historique de navigation
  %sversion%s         Informations système
  %s?%s / %saide%s         Cette aide

%sFoxChain (données chiffrées):%s
  %sfav%s             Lister les favoris
  %sfav+ <titre>%s    Ajouter la page aux favoris
  %sfav- <n>%s        Supprimer un favori
  %smdp%s             Lister les mots de passe
  %smdp+ <s> <u> <p>%s  Ajouter (site user pass)
  %smdp? <site|n>%s   Révéler un mot de passe
  %smdp- <site|n>%s   Supprimer un identifiant
  %schain%s           Infos FoxChain
  %sverify%s           Vérifier l'intégrité
  %supdate%s           Vérifier les mises à jour (OTA signé)

%sQuitter:%s
  %sq%s / %squitter%s      Fermer Fox Browser
`, colorBold, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset, colorCyan, colorReset,
		colorBold, colorReset,
		colorCyan, colorReset, colorCyan, colorReset)
}
