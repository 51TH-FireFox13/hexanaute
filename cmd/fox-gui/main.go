package main

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/51TH-FireFox13/fox-browser/internal/aiguard"
	"github.com/51TH-FireFox13/fox-browser/internal/browser"
	"github.com/51TH-FireFox13/fox-browser/internal/engine"
	"github.com/51TH-FireFox13/fox-browser/internal/network"
)

const version = "0.0.3"

type FoxGUI struct {
	app     fyne.App
	window  fyne.Window
	client  *network.Client
	guard   *aiguard.Guard
	session *browser.Session

	// Widgets
	urlEntry    *widget.Entry
	statusLabel *widget.Label
	guardLabel  *widget.Label
	content     *widget.RichText
	linkList    *widget.List
	backBtn     *widget.Button
	fwdBtn      *widget.Button

	// State
	currentLinks []engine.Link
}

func main() {
	fox := &FoxGUI{
		client:  network.NewClient(),
		guard:   aiguard.NewGuard(0.7),
		session: browser.NewSession(),
	}

	fox.app = app.NewWithID("fr.fox-browser")
	fox.app.Settings().SetTheme(&foxTheme{})

	fox.window = fox.app.NewWindow(fmt.Sprintf("🦊 Fox Browser v%s", version))
	fox.window.Resize(fyne.NewSize(1024, 768))

	fox.buildUI()

	fox.window.ShowAndRun()
}

func (f *FoxGUI) buildUI() {
	// ── Barre d'URL ──
	f.urlEntry = widget.NewEntry()
	f.urlEntry.SetPlaceHolder("Entrez une URL...")
	f.urlEntry.OnSubmitted = func(url string) {
		f.navigate(url)
	}

	goBtn := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		f.navigate(f.urlEntry.Text)
	})

	f.backBtn = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		p := f.session.Back()
		if p != nil {
			f.urlEntry.SetText(p.URL)
			f.displayPage(p)
		}
	})

	f.fwdBtn = widget.NewButtonWithIcon("", theme.MailForwardIcon(), func() {
		p := f.session.Forward()
		if p != nil {
			f.urlEntry.SetText(p.URL)
			f.displayPage(p)
		}
	})

	reloadBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		p := f.session.Current()
		if p != nil {
			f.navigate(p.URL)
		}
	})

	f.backBtn.Disable()
	f.fwdBtn.Disable()

	navbar := container.NewBorder(
		nil, nil,
		container.NewHBox(f.backBtn, f.fwdBtn, reloadBtn),
		goBtn,
		f.urlEntry,
	)

	// ── Barre d'état ──
	f.statusLabel = widget.NewLabel("Prêt")
	f.statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	f.guardLabel = widget.NewLabel("[AI Guard] ✓ En attente")
	f.guardLabel.TextStyle = fyne.TextStyle{Bold: true}

	statusBar := container.NewHBox(
		f.guardLabel,
		layout.NewSpacer(),
		f.statusLabel,
	)

	// ── Contenu de la page ──
	f.content = widget.NewRichTextFromMarkdown("# 🦊 Fox Browser\n\nNavigateur souverain, sécurisé par IA.\n\nEntrez une URL ci-dessus pour commencer.")
	f.content.Wrapping = fyne.TextWrapWord

	contentScroll := container.NewVScroll(f.content)

	// ── Liste des liens ──
	f.currentLinks = make([]engine.Link, 0)
	f.linkList = widget.NewList(
		func() int { return len(f.currentLinks) },
		func() fyne.CanvasObject {
			return widget.NewLabel("lien")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(f.currentLinks) {
				link := f.currentLinks[id]
				text := link.Text
				if len(text) > 60 {
					text = text[:57] + "..."
				}
				obj.(*widget.Label).SetText(fmt.Sprintf("[%d] %s", link.Index, text))
			}
		},
	)
	f.linkList.OnSelected = func(id widget.ListItemID) {
		if id < len(f.currentLinks) {
			resolved := f.session.ResolveURL(f.currentLinks[id].URL)
			f.navigate(resolved)
		}
		f.linkList.UnselectAll()
	}

	linksPanel := container.NewBorder(
		widget.NewLabel("Liens"),
		nil, nil, nil,
		f.linkList,
	)
	linksPanel.Resize(fyne.NewSize(250, 0))

	// ── Split principal ──
	split := container.NewHSplit(contentScroll, linksPanel)
	split.SetOffset(0.75)

	// ── Layout final ──
	mainContent := container.NewBorder(
		container.NewVBox(navbar, widget.NewSeparator(), statusBar),
		nil,
		nil, nil,
		split,
	)

	f.window.SetContent(mainContent)
}

func (f *FoxGUI) navigate(rawURL string) {
	url := normalizeURL(rawURL)
	f.urlEntry.SetText(url)
	f.statusLabel.SetText("Chargement...")
	f.guardLabel.SetText("[AI Guard] Analyse en cours...")

	go func() {
		resp, err := f.client.Fetch(url)
		if err != nil {
			f.statusLabel.SetText("Erreur : " + err.Error())
			f.guardLabel.SetText("[AI Guard] —")
			return
		}

		// Status
		tlsInfo := resp.TLSVersion
		if tlsInfo == "" {
			tlsInfo = "pas de TLS"
		}
		lock := "🔓"
		if strings.HasPrefix(url, "https") {
			lock = "🔒"
		}
		f.statusLabel.SetText(fmt.Sprintf("%s HTTP %d | %s | %s",
			lock, resp.StatusCode, tlsInfo, resp.Duration.Round(time.Millisecond)))

		// AI Guard
		result := f.guard.AnalyzePage(resp.Body, url)
		f.updateGuardStatus(result)

		if result.Blocked {
			f.content.ParseMarkdown("# 🛡 Page bloquée\n\nL'AI Guard a détecté un contenu potentiellement malveillant.\n\n**Catégorie :** " + string(result.Category) + "\n\n**Détails :** " + result.Details)
			return
		}

		// Parser
		doc, err := engine.Parse(resp.Body)
		if err != nil {
			f.statusLabel.SetText("Erreur parsing : " + err.Error())
			return
		}

		// Rendre
		rendered := engine.Render(doc)

		// Sauvegarder dans la session
		pageLinks := make([]browser.PageLink, len(rendered.Links))
		for i, l := range rendered.Links {
			pageLinks[i] = browser.PageLink{
				Index: l.Index,
				Text:  l.Text,
				URL:   l.URL,
			}
		}
		f.session.Push(browser.Page{
			URL:     url,
			Content: rendered.Content,
			Links:   pageLinks,
		})

		page := f.session.Current()
		f.displayPage(page)

		// Mettre à jour les liens
		f.currentLinks = rendered.Links
		f.linkList.Refresh()

		// Mettre à jour les boutons de navigation
		if f.session.CanBack() {
			f.backBtn.Enable()
		} else {
			f.backBtn.Disable()
		}
		if f.session.CanForward() {
			f.fwdBtn.Enable()
		} else {
			f.fwdBtn.Disable()
		}
	}()
}

func (f *FoxGUI) displayPage(page *browser.Page) {
	if page == nil {
		return
	}

	// Convertir le contenu en pseudo-markdown pour RichText
	md := contentToMarkdown(page.Content)
	f.content.ParseMarkdown(md)
	f.urlEntry.SetText(page.URL)

	// Recharger les liens
	f.currentLinks = f.currentLinks[:0]
	for _, l := range page.Links {
		f.currentLinks = append(f.currentLinks, engine.Link{
			Index: l.Index,
			Text:  l.Text,
			URL:   l.URL,
		})
	}
	f.linkList.Refresh()
}

func (f *FoxGUI) updateGuardStatus(result *aiguard.AnalysisResult) {
	icon := "✓"
	label := "SÛR"

	if result.Score > 0.3 {
		icon = "⚠"
		label = "SUSPECT"
	}
	if result.Score > 0.7 {
		icon = "✗"
		label = "DANGEREUX"
	}
	if result.Blocked {
		icon = "🛡"
		label = "BLOQUÉ"
	}

	text := fmt.Sprintf("[AI Guard] %s %s (%.0f%%)", icon, label, result.Score*100)
	if result.Details != "" && result.Score > 0.1 {
		text += " — " + result.Details
	}
	f.guardLabel.SetText(text)
}

func contentToMarkdown(content string) string {
	// Nettoyer les codes ANSI
	content = stripANSI(content)

	// Remplacer les séquences de formatage terminal par du markdown
	content = strings.ReplaceAll(content, "══════════════════════════════════════════════\n", "")
	content = strings.ReplaceAll(content, "──────────────────────────────────────────────\n", "---\n")
	content = strings.ReplaceAll(content, "── ", "## ")
	content = strings.ReplaceAll(content, " ──", "")
	content = strings.ReplaceAll(content, "   ▸ ", "### ")
	content = strings.ReplaceAll(content, "     ▹ ", "#### ")

	// Nettoyer les lignes vides multiples
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	return content
}

func stripANSI(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			// Sauter la séquence ANSI
			j := i + 2
			for j < len(s) && !((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				j++
			}
			if j < len(s) {
				j++ // sauter la lettre finale
			}
			i = j
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

func normalizeURL(input string) string {
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return "https://" + input
	}
	return input
}
