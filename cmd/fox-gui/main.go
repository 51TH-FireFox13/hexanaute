package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"image/color"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/51TH-FireFox13/fox-browser/internal/aiguard"
	"github.com/51TH-FireFox13/fox-browser/internal/browser"
	"github.com/51TH-FireFox13/fox-browser/internal/css"
	"github.com/51TH-FireFox13/fox-browser/internal/engine"
	"github.com/51TH-FireFox13/fox-browser/internal/foxchain"
	"github.com/51TH-FireFox13/fox-browser/internal/foxota"
	"github.com/51TH-FireFox13/fox-browser/internal/jsengine"
	"github.com/51TH-FireFox13/fox-browser/internal/network"
)

const (
	version    = "0.1.0"
	versionNum = 100 // scheme: major*10000 + minor*100 + patch

	signingPubKeyHex = "ca69cb8216ba7b4f2a5198286fd62e106af21fdb0a24f16f148db567ba0a5e00"
	updateSource1    = "http://192.168.1.42:8090/manifest.json"
	updateSource2    = "http://192.168.1.42:8090/manifest.json"
)

// ══════════════════════════════════════
// TYPES
// ══════════════════════════════════════

// FoxTab représente l'état complet d'un onglet de navigation.
type FoxTab struct {
	id         int
	pageURL    string // URL actuelle
	title      string
	session    *browser.Session
	links      []engine.Link
	contentBox *fyne.Container
	scroll     *container.Scroll
	tabItem    *container.TabItem
	loading    bool
	cancel     context.CancelFunc
}

// FoxGUI est l'application principale.
type FoxGUI struct {
	app      fyne.App
	window   fyne.Window
	client   *network.Client
	guard    *aiguard.Guard
	jsEngine *jsengine.Engine
	profile  *foxchain.Profile

	// Barre de navigation (partagée entre onglets)
	urlEntry    *widget.Entry
	statusLabel *widget.Label
	guardLabel  *widget.Label
	backBtn     *widget.Button
	fwdBtn      *widget.Button
	favBtn      *widget.Button
	stopBtn     *widget.Button
	reloadBtn   *widget.Button

	// Panneau latéral (mis à jour selon l'onglet actif)
	linkList  *widget.List
	favList   *widget.List
	sidePanel *container.AppTabs

	// Gestion des onglets
	docTabs    *container.DocTabs
	foxTabs    []*FoxTab
	tabCounter int

	// Données profil (bookmarks, credentials)
	bookmarks   []foxchain.Bookmark
	credentials []foxchain.Credential
}

// ══════════════════════════════════════
// MAIN
// ══════════════════════════════════════

func main() {
	fox := &FoxGUI{
		client:   network.NewClient(),
		guard:    aiguard.NewGuard(0.7),
		jsEngine: jsengine.New(jsengine.DefaultConfig()),
		foxTabs:  make([]*FoxTab, 0, 8),
	}

	fox.app = app.NewWithID("fr.fox-browser")
	fox.app.Settings().SetTheme(&foxTheme{})

	fox.window = fox.app.NewWindow(fmt.Sprintf("Fox Browser v%s", version))
	fox.window.Resize(fyne.NewSize(1280, 800))

	fox.buildUI()
	fox.window.SetCloseIntercept(fox.onClose)

	fox.window.Show()
	fox.initFoxChain()

	fox.app.Run()
}

// ══════════════════════════════════════
// UI CONSTRUCTION
// ══════════════════════════════════════

func (f *FoxGUI) buildUI() {
	// ── Barre de navigation ──
	f.urlEntry = widget.NewEntry()
	f.urlEntry.SetPlaceHolder("Rechercher ou entrer une adresse web...")
	f.urlEntry.OnSubmitted = func(input string) { f.navigate(input) }

	f.backBtn = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		tab := f.currentFoxTab()
		if tab == nil {
			return
		}
		if p := tab.session.Back(); p != nil {
			f.urlEntry.SetText(p.URL)
			f.navigateInTab(tab, p.URL, false)
		}
	})
	f.fwdBtn = widget.NewButtonWithIcon("", theme.MailForwardIcon(), func() {
		tab := f.currentFoxTab()
		if tab == nil {
			return
		}
		if p := tab.session.Forward(); p != nil {
			f.urlEntry.SetText(p.URL)
			f.navigateInTab(tab, p.URL, false)
		}
	})
	f.reloadBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		tab := f.currentFoxTab()
		if tab == nil {
			return
		}
		if tab.pageURL != "" && !strings.HasPrefix(tab.pageURL, "about:") {
			f.navigateInTab(tab, tab.pageURL, false)
		}
	})
	f.stopBtn = widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		tab := f.currentFoxTab()
		if tab != nil && tab.cancel != nil {
			tab.cancel()
		}
	})
	f.favBtn = widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		f.addCurrentBookmark()
	})
	newTabBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		tab := f.createFoxTab("Nouvel onglet")
		f.docTabs.Append(tab.tabItem)
		f.docTabs.Select(tab.tabItem)
		f.showAboutPage(tab)
	})
	newTabBtn.Importance = widget.LowImportance

	f.backBtn.Disable()
	f.fwdBtn.Disable()
	f.stopBtn.Disable()

	navbar := container.NewBorder(nil, nil,
		container.NewHBox(f.backBtn, f.fwdBtn, f.reloadBtn, f.stopBtn),
		container.NewHBox(f.favBtn, newTabBtn),
		f.urlEntry,
	)

	// ── Barre d'état ──
	f.guardLabel = widget.NewLabel("[AI Guard] En attente")
	f.guardLabel.TextStyle = fyne.TextStyle{Bold: true}
	f.statusLabel = widget.NewLabel("[FoxChain] —")
	f.statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	statusBar := container.NewHBox(
		f.guardLabel,
		layout.NewSpacer(),
		f.statusLabel,
	)

	// ── Panneau latéral : Liens ──
	f.linkList = widget.NewList(
		func() int {
			tab := f.currentFoxTab()
			if tab == nil {
				return 0
			}
			return len(tab.links)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("lien")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			tab := f.currentFoxTab()
			if tab == nil || id >= len(tab.links) {
				return
			}
			lnk := tab.links[id]
			text := lnk.Text
			if len(text) > 45 {
				text = text[:42] + "..."
			}
			obj.(*widget.Label).SetText(fmt.Sprintf("[%d] %s", lnk.Index, text))
		},
	)
	f.linkList.OnSelected = func(id widget.ListItemID) {
		tab := f.currentFoxTab()
		if tab != nil && id < len(tab.links) {
			resolved := tab.session.ResolveURL(tab.links[id].URL)
			f.navigate(resolved)
		}
		f.linkList.UnselectAll()
	}

	// ── Panneau latéral : Favoris ──
	f.bookmarks = make([]foxchain.Bookmark, 0)
	f.favList = widget.NewList(
		func() int { return len(f.bookmarks) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("favori"),
				layout.NewSpacer(),
				widget.NewButtonWithIcon("", theme.DeleteIcon(), nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(f.bookmarks) {
				return
			}
			bm := f.bookmarks[id]
			hbox := obj.(*fyne.Container)
			label := hbox.Objects[0].(*widget.Label)
			title := bm.Title
			if len(title) > 35 {
				title = title[:32] + "..."
			}
			label.SetText(title)
			delBtn := hbox.Objects[2].(*widget.Button)
			bmURL := bm.URL
			delBtn.OnTapped = func() { f.removeBookmark(bmURL) }
		},
	)
	f.favList.OnSelected = func(id widget.ListItemID) {
		if id < len(f.bookmarks) {
			f.navigate(f.bookmarks[id].URL)
		}
		f.favList.UnselectAll()
	}

	// ── Panneau latéral : Mots de passe ──
	f.credentials = make([]foxchain.Credential, 0)
	pwdAddBtn := widget.NewButtonWithIcon("Ajouter", theme.ContentAddIcon(), func() {
		f.showAddPasswordDialog()
	})
	pwdList := widget.NewList(
		func() int { return len(f.credentials) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("site"),
				layout.NewSpacer(),
				widget.NewButtonWithIcon("", theme.VisibilityIcon(), nil),
				widget.NewButtonWithIcon("", theme.DeleteIcon(), nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(f.credentials) {
				return
			}
			cred := f.credentials[id]
			hbox := obj.(*fyne.Container)
			label := hbox.Objects[0].(*widget.Label)
			site := cred.Site
			if len(site) > 28 {
				site = site[:25] + "..."
			}
			label.SetText(fmt.Sprintf("%s (%s)", site, cred.Username))

			showBtn := hbox.Objects[2].(*widget.Button)
			credSite, credUser := cred.Site, cred.Username
			showBtn.OnTapped = func() { f.showPasswordReveal(credSite, credUser) }

			delBtn := hbox.Objects[3].(*widget.Button)
			delBtn.OnTapped = func() { f.removePassword(cred.Site) }
		},
	)
	pwdPanel := container.NewBorder(pwdAddBtn, nil, nil, nil, pwdList)

	f.sidePanel = container.NewAppTabs(
		container.NewTabItemWithIcon("Liens", theme.ListIcon(), f.linkList),
		container.NewTabItemWithIcon("Favoris", theme.ContentAddIcon(), f.favList),
		container.NewTabItemWithIcon("Mots de passe", theme.SettingsIcon(), pwdPanel),
	)

	// ── Menu ──
	f.window.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("Fox",
			fyne.NewMenuItem("Vérifier les mises à jour", f.checkUpdate),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Intégrité FoxChain", f.verifyChain),
			fyne.NewMenuItem("Infos FoxChain", f.showChainInfo),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Vider les cookies", func() {
				dialog.ShowConfirm("Cookies", "Vider tous les cookies de session ?", func(yes bool) {
					if yes {
						f.client.ClearCookies()
						dialog.ShowInformation("Cookies", "Cookies vidés.", f.window)
					}
				}, f.window)
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("À propos", f.showAbout),
		),
		fyne.NewMenu("Historique",
			fyne.NewMenuItem("Voir l'historique", f.showHistory),
		),
		fyne.NewMenu("Onglets",
			fyne.NewMenuItem("Nouvel onglet", func() {
				tab := f.createFoxTab("Nouvel onglet")
				f.docTabs.Append(tab.tabItem)
				f.docTabs.Select(tab.tabItem)
				f.showAboutPage(tab)
			}),
		),
	))

	// ── DocTabs ──
	f.docTabs = container.NewDocTabs()
	f.docTabs.OnSelected = f.onTabChanged
	f.docTabs.OnClosed = f.onTabClosed

	// Bouton "+" pour nouvel onglet
	f.docTabs.CreateTab = func() *container.TabItem {
		tab := f.createFoxTab("Nouvel onglet")
		f.showAboutPage(tab)
		return tab.tabItem
	}

	// ── Layout principal ──
	split := container.NewHSplit(f.docTabs, f.sidePanel)
	split.SetOffset(0.75)

	mainContent := container.NewBorder(
		container.NewVBox(navbar, widget.NewSeparator(), statusBar),
		nil, nil, nil,
		split,
	)

	f.window.SetContent(mainContent)

	// Créer le premier onglet
	firstTab := f.createFoxTab("Nouvel onglet")
	f.docTabs.Append(firstTab.tabItem)
	f.docTabs.Select(firstTab.tabItem)
	f.showAboutPage(firstTab)
}

// ══════════════════════════════════════
// GESTION DES ONGLETS
// ══════════════════════════════════════

// createFoxTab crée un nouvel onglet (sans l'ajouter au DocTabs).
func (f *FoxGUI) createFoxTab(title string) *FoxTab {
	f.tabCounter++

	contentBox := container.NewVBox()
	scroll := container.NewVScroll(contentBox)

	tabItem := container.NewTabItem(title, scroll)

	tab := &FoxTab{
		id:         f.tabCounter,
		title:      title,
		pageURL:    "",
		session:    browser.NewSession(),
		links:      make([]engine.Link, 0),
		contentBox: contentBox,
		scroll:     scroll,
		tabItem:    tabItem,
	}

	f.foxTabs = append(f.foxTabs, tab)
	return tab
}

// currentFoxTab retourne l'onglet actif.
func (f *FoxGUI) currentFoxTab() *FoxTab {
	if f.docTabs == nil || len(f.foxTabs) == 0 {
		return nil
	}
	selected := f.docTabs.Selected()
	if selected == nil {
		return nil
	}
	return f.findTab(selected)
}

// findTab retrouve le FoxTab correspondant à un TabItem Fyne.
func (f *FoxGUI) findTab(item *container.TabItem) *FoxTab {
	for _, t := range f.foxTabs {
		if t.tabItem == item {
			return t
		}
	}
	return nil
}

// onTabClosed est appelé quand l'utilisateur ferme un onglet.
func (f *FoxGUI) onTabClosed(item *container.TabItem) {
	for i, t := range f.foxTabs {
		if t.tabItem == item {
			// Annuler le chargement en cours
			if t.cancel != nil {
				t.cancel()
			}
			// Retirer de la liste
			f.foxTabs = append(f.foxTabs[:i], f.foxTabs[i+1:]...)
			return
		}
	}
}

// onTabChanged est appelé quand l'utilisateur change d'onglet.
func (f *FoxGUI) onTabChanged(item *container.TabItem) {
	tab := f.findTab(item)
	if tab == nil {
		return
	}
	f.urlEntry.SetText(tab.pageURL)
	f.linkList.Refresh()
	f.updateNavButtons(tab)
	title := tab.title
	if title == "" || title == "Nouvel onglet" {
		f.window.SetTitle(fmt.Sprintf("Fox Browser v%s", version))
	} else {
		f.window.SetTitle(fmt.Sprintf("Fox Browser — %s", title))
	}
}

// updateNavButtons met à jour l'état des boutons de navigation selon l'onglet.
func (f *FoxGUI) updateNavButtons(tab *FoxTab) {
	if tab == nil {
		f.backBtn.Disable()
		f.fwdBtn.Disable()
		return
	}
	if tab.session.CanBack() {
		f.backBtn.Enable()
	} else {
		f.backBtn.Disable()
	}
	if tab.session.CanForward() {
		f.fwdBtn.Enable()
	} else {
		f.fwdBtn.Disable()
	}
	if tab.loading {
		f.stopBtn.Enable()
	} else {
		f.stopBtn.Disable()
	}
}

// setTabTitle met à jour le titre d'un onglet.
func (f *FoxGUI) setTabTitle(tab *FoxTab, title string) {
	tab.title = title
	display := title
	if len(display) > 22 {
		display = display[:19] + "..."
	}
	tab.tabItem.Text = display
	f.docTabs.Refresh()
	if f.currentFoxTab() == tab {
		if title != "" {
			f.window.SetTitle(fmt.Sprintf("Fox Browser — %s", title))
		}
	}
}

// ══════════════════════════════════════
// FOXCHAIN
// ══════════════════════════════════════

func (f *FoxGUI) foxchainDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".fox")
}

func (f *FoxGUI) initFoxChain() {
	dir := f.foxchainDir()

	if foxchain.Exists(dir) {
		passEntry := widget.NewPasswordEntry()
		passEntry.SetPlaceHolder("Votre passphrase FoxChain")

		d := dialog.NewForm("FoxChain — Déverrouiller", "Ouvrir", "Sans profil",
			[]*widget.FormItem{
				widget.NewFormItem("Passphrase", passEntry),
			},
			func(ok bool) {
				if !ok {
					f.statusLabel.SetText("[FoxChain] Navigation sans profil")
					return
				}
				p, err := foxchain.OpenProfile(dir, passEntry.Text)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Passphrase incorrecte ou profil corrompu"), f.window)
					f.statusLabel.SetText("[FoxChain] Erreur")
					return
				}
				f.profile = p
				f.refreshBookmarks()
				f.refreshCredentials()
				stats := f.profile.Vault().Stats()
				f.statusLabel.SetText(fmt.Sprintf("[FoxChain] %d blocs — OK", stats["total"]))

				// Restaurer la dernière session
				if state, err := f.profile.Vault().LoadTabState(); err == nil && len(state.Tabs) > 0 {
					dialog.ShowConfirm("Restaurer la session",
						fmt.Sprintf("Reprendre %d onglet(s) de la dernière session ?", len(state.Tabs)),
						func(yes bool) {
							if yes {
								for i, tabURL := range state.Tabs {
									if i == 0 {
										// Naviguer dans le premier onglet existant
										tab := f.currentFoxTab()
										if tab != nil {
											f.navigateInTab(tab, tabURL, true)
										}
									} else if i < 5 { // limiter à 5 onglets restaurés
										tab := f.createFoxTab("Restauration...")
										f.docTabs.Append(tab.tabItem)
										f.navigateInTab(tab, tabURL, true)
									}
								}
							}
						}, f.window)
				}
			}, f.window)
		d.Resize(fyne.NewSize(400, 200))
		d.Show()
	} else {
		d := dialog.NewConfirm("FoxChain",
			"Aucun profil trouvé.\nCréer un profil chiffré pour sauvegarder favoris, mots de passe et historique ?",
			func(yes bool) {
				if yes {
					f.showCreateProfile()
				} else {
					f.statusLabel.SetText("[FoxChain] Navigation sans profil")
				}
			}, f.window)
		d.Show()
	}
}

func (f *FoxGUI) showCreateProfile() {
	pass1 := widget.NewPasswordEntry()
	pass1.SetPlaceHolder("Choisissez une passphrase (min 8 car.)")
	pass2 := widget.NewPasswordEntry()
	pass2.SetPlaceHolder("Confirmez")

	d := dialog.NewForm("Créer un profil FoxChain", "Créer", "Annuler",
		[]*widget.FormItem{
			widget.NewFormItem("Passphrase", pass1),
			widget.NewFormItem("Confirmation", pass2),
		},
		func(ok bool) {
			if !ok {
				return
			}
			if len(pass1.Text) < 8 {
				dialog.ShowError(fmt.Errorf("Passphrase trop courte (min 8 caractères)"), f.window)
				return
			}
			if pass1.Text != pass2.Text {
				dialog.ShowError(fmt.Errorf("Les passphrases ne correspondent pas"), f.window)
				return
			}
			p, err := foxchain.CreateProfile(f.foxchainDir(), pass1.Text)
			if err != nil {
				dialog.ShowError(err, f.window)
				return
			}
			f.profile = p
			f.statusLabel.SetText("[FoxChain] Profil créé")
			dialog.ShowInformation("FoxChain",
				"Profil créé !\nVotre passphrase est la seule clé — ne la perdez pas.",
				f.window)
		}, f.window)
	d.Resize(fyne.NewSize(450, 250))
	d.Show()
}

// ══════════════════════════════════════
// FAVORIS
// ══════════════════════════════════════

func (f *FoxGUI) refreshBookmarks() {
	if f.profile == nil {
		return
	}
	bm, _ := f.profile.Vault().ListBookmarks()
	f.bookmarks = bm
	f.favList.Refresh()
}

func (f *FoxGUI) addCurrentBookmark() {
	tab := f.currentFoxTab()
	if tab == nil || tab.pageURL == "" || strings.HasPrefix(tab.pageURL, "about:") {
		dialog.ShowInformation("Favoris", "Naviguez d'abord vers une page.", f.window)
		return
	}
	if f.profile == nil {
		dialog.ShowInformation("FoxChain", "Pas de profil actif.\nCréez-en un via le menu Fox.", f.window)
		return
	}

	titleEntry := widget.NewEntry()
	titleEntry.SetText(tab.title)

	d := dialog.NewForm("Ajouter aux favoris", "Ajouter", "Annuler",
		[]*widget.FormItem{
			widget.NewFormItem("Titre", titleEntry),
			widget.NewFormItem("URL", widget.NewLabel(tab.pageURL)),
		},
		func(ok bool) {
			if !ok {
				return
			}
			f.profile.Vault().AddBookmark(tab.pageURL, titleEntry.Text)
			f.profile.Save()
			f.refreshBookmarks()
			f.sidePanel.SelectIndex(1)
		}, f.window)
	d.Resize(fyne.NewSize(500, 200))
	d.Show()
}

func (f *FoxGUI) removeBookmark(bmURL string) {
	if f.profile == nil {
		return
	}
	dialog.ShowConfirm("Supprimer le favori", "Supprimer ce favori ?", func(yes bool) {
		if yes {
			f.profile.Vault().RemoveBookmark(bmURL)
			f.profile.Save()
			f.refreshBookmarks()
		}
	}, f.window)
}

// ══════════════════════════════════════
// MOTS DE PASSE
// ══════════════════════════════════════

func (f *FoxGUI) refreshCredentials() {
	if f.profile == nil {
		return
	}
	creds, _ := f.profile.Vault().ListCredentials()
	f.credentials = creds
}

func (f *FoxGUI) showAddPasswordDialog() {
	if f.profile == nil {
		dialog.ShowInformation("FoxChain", "Pas de profil actif.", f.window)
		return
	}

	siteEntry := widget.NewEntry()
	siteEntry.SetPlaceHolder("exemple.com")
	userEntry := widget.NewEntry()
	userEntry.SetPlaceHolder("utilisateur")
	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("mot de passe")

	// Pré-remplir le site avec l'URL actuelle
	if tab := f.currentFoxTab(); tab != nil && tab.pageURL != "" {
		if parsed, err := url.Parse(tab.pageURL); err == nil {
			siteEntry.SetText(parsed.Hostname())
		}
	}

	d := dialog.NewForm("Ajouter un mot de passe", "Enregistrer", "Annuler",
		[]*widget.FormItem{
			widget.NewFormItem("Site", siteEntry),
			widget.NewFormItem("Utilisateur", userEntry),
			widget.NewFormItem("Mot de passe", passEntry),
		},
		func(ok bool) {
			if !ok {
				return
			}
			if siteEntry.Text == "" || userEntry.Text == "" || passEntry.Text == "" {
				dialog.ShowError(fmt.Errorf("Tous les champs sont obligatoires"), f.window)
				return
			}
			f.profile.Vault().AddCredential(siteEntry.Text, userEntry.Text, passEntry.Text)
			f.profile.Save()
			f.refreshCredentials()
		}, f.window)
	d.Resize(fyne.NewSize(450, 300))
	d.Show()
}

func (f *FoxGUI) showPasswordReveal(site, username string) {
	if f.profile == nil {
		return
	}
	cred, err := f.profile.Vault().GetCredential(site)
	if err != nil {
		dialog.ShowError(err, f.window)
		return
	}

	passEntry := widget.NewEntry()
	passEntry.SetText(cred.Password)
	passEntry.Disable()

	d := dialog.NewCustom(
		fmt.Sprintf("Mot de passe — %s", site),
		"Fermer",
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Site : %s", cred.Site)),
			widget.NewLabel(fmt.Sprintf("Utilisateur : %s", cred.Username)),
			widget.NewLabel("Mot de passe :"),
			passEntry,
		),
		f.window,
	)
	d.Resize(fyne.NewSize(400, 200))
	d.Show()
}

func (f *FoxGUI) removePassword(site string) {
	if f.profile == nil {
		return
	}
	dialog.ShowConfirm("Supprimer", fmt.Sprintf("Supprimer l'identifiant pour %s ?", site),
		func(yes bool) {
			if yes {
				f.profile.Vault().RemoveCredential(site)
				f.profile.Save()
				f.refreshCredentials()
			}
		}, f.window)
}

// ══════════════════════════════════════
// HISTORIQUE
// ══════════════════════════════════════

func (f *FoxGUI) showHistory() {
	if f.profile == nil {
		dialog.ShowInformation("Historique", "Pas de profil actif — l'historique n'est pas persisté.", f.window)
		return
	}

	entries, _ := f.profile.Vault().ListHistory(100)
	if len(entries) == 0 {
		dialog.ShowInformation("Historique", "Aucune entrée.", f.window)
		return
	}

	list := widget.NewList(
		func() int { return len(entries) },
		func() fyne.CanvasObject {
			return widget.NewLabel("historique")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(entries) {
				e := entries[id]
				t := time.Unix(e.VisitedAt, 0).Format("02/01 15:04")
				pageURL := e.URL
				if len(pageURL) > 65 {
					pageURL = pageURL[:62] + "..."
				}
				obj.(*widget.Label).SetText(fmt.Sprintf("[%s] %s", t, pageURL))
			}
		},
	)

	histWin := f.app.NewWindow("Historique de navigation")
	histWin.Resize(fyne.NewSize(700, 450))

	list.OnSelected = func(id widget.ListItemID) {
		if id < len(entries) {
			f.navigate(entries[id].URL)
			histWin.Close()
		}
	}

	histWin.SetContent(list)
	histWin.Show()
}

// ══════════════════════════════════════
// FOXOTA / MENU
// ══════════════════════════════════════

func (f *FoxGUI) checkUpdate() {
	f.guardLabel.SetText("[FoxOTA] Vérification...")

	go func() {
		pubKeyBytes, err := hex.DecodeString(signingPubKeyHex)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Clé de signature invalide"), f.window)
			return
		}

		checker := foxota.NewChecker(pubKeyBytes, versionNum, "stable")
		checker.AddSource("source-1", updateSource1)
		checker.AddSource("source-2", updateSource2)

		result := checker.Check()

		if result.Error != nil {
			if result.Error == foxota.ErrNoUpdate {
				f.guardLabel.SetText("[FoxOTA] À jour")
				dialog.ShowInformation("FoxOTA",
					fmt.Sprintf("Fox Browser v%s est à jour.", version),
					f.window)
			} else {
				f.guardLabel.SetText("[AI Guard] En attente")
				dialog.ShowError(fmt.Errorf("Erreur de vérification :\n%s", result.Error), f.window)
			}
			return
		}

		if result.Available {
			f.guardLabel.SetText(fmt.Sprintf("[FoxOTA] v%s disponible !", result.Manifest.VersionStr))

			info := fmt.Sprintf(
				"Nouvelle version disponible !\n\n"+
					"Version actuelle : v%s\n"+
					"Nouvelle version : v%s\n"+
					"Canal : %s\n",
				version, result.Manifest.VersionStr, result.Manifest.Channel)

			if result.Manifest.Changelog != "" {
				info += fmt.Sprintf("Changelog : %s\n", result.Manifest.Changelog)
			}
			if result.BinaryInfo != nil {
				info += fmt.Sprintf("Taille : %.1f Mo\n", float64(result.BinaryInfo.Size)/1024/1024)
			}
			info += fmt.Sprintf("\nConsensus multi-source : %v", result.Verified)

			dialog.ShowConfirm("Mise à jour disponible", info+"\n\nTélécharger et installer ?",
				func(yes bool) {
					if yes {
						f.downloadUpdate(checker, result)
					}
				}, f.window)
		}
	}()
}

func (f *FoxGUI) downloadUpdate(checker *foxota.Checker, checkResult *foxota.CheckResult) {
	f.guardLabel.SetText("[FoxOTA] Téléchargement...")

	go func() {
		update, vr, err := checker.Download(checkResult.BinaryInfo)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Échec du téléchargement :\n%s", err), f.window)
			f.guardLabel.SetText("[AI Guard] En attente")
			return
		}

		if vr != nil && !vr.HashOK {
			dialog.ShowError(fmt.Errorf("ALERTE SÉCURITÉ !\n\nHash invalide — mise à jour annulée."), f.window)
			f.guardLabel.SetText("[FoxOTA] HASH INVALIDE")
			return
		}

		f.guardLabel.SetText("[FoxOTA] Application...")
		if err := foxota.Apply(update.Binary); err != nil {
			dialog.ShowError(fmt.Errorf("Échec application :\n%s", err), f.window)
			f.guardLabel.SetText("[AI Guard] En attente")
			return
		}

		f.guardLabel.SetText("[FoxOTA] Mise à jour appliquée !")
		dialog.ShowInformation("FoxOTA",
			fmt.Sprintf("v%s installée !\nRedémarrez Fox Browser.", checkResult.Manifest.VersionStr),
			f.window)
	}()
}

func (f *FoxGUI) verifyChain() {
	if f.profile == nil {
		dialog.ShowInformation("FoxChain", "Pas de profil actif.", f.window)
		return
	}
	chain := f.profile.Vault().Chain()
	if chain.Verify() {
		dialog.ShowInformation("Intégrité FoxChain",
			fmt.Sprintf("Chaîne vérifiée.\n%d blocs — intégrité OK", chain.Len()),
			f.window)
	} else {
		dialog.ShowError(fmt.Errorf("INTÉGRITÉ COMPROMISE\nLa chaîne a été altérée !"), f.window)
	}
}

func (f *FoxGUI) showChainInfo() {
	if f.profile == nil {
		dialog.ShowInformation("FoxChain", "Pas de profil actif.", f.window)
		return
	}
	stats := f.profile.Vault().Stats()
	info := fmt.Sprintf(
		"Répertoire : %s\n\nBlocs : %d\nFavoris : %d\nMots de passe : %d\nHistorique : %d\nÉtats onglets : %d",
		f.profile.Dir,
		stats["total"], stats["favoris"], stats["mots_de_passe"],
		stats["historique"], stats["états_onglets"])
	dialog.ShowInformation("Infos FoxChain", info, f.window)
}

func (f *FoxGUI) showAbout() {
	dialog.ShowInformation("À propos",
		fmt.Sprintf(
			"Fox Browser v%s (Renard)\n\n"+
				"Navigateur souverain\n"+
				"Sécurisé par IA locale (AI Guard)\n"+
				"Stockage chiffré (FoxChain)\n"+
				"Mises à jour P2P signées (FoxOTA)\n\n"+
				"Licence EUPL v1.2\n"+
				"github.com/51TH-FireFox13/fox-browser",
			version),
		f.window)
}

// ══════════════════════════════════════
// NAVIGATION
// ══════════════════════════════════════

// navigate navigue dans l'onglet actif.
func (f *FoxGUI) navigate(input string) {
	tab := f.currentFoxTab()
	if tab == nil {
		return
	}
	f.navigateInTab(tab, input, true)
}

// navigateInTab effectue la navigation dans un onglet spécifique.
// addHistory : false pour back/forward/reload (ne pas repousser dans session).
func (f *FoxGUI) navigateInTab(tab *FoxTab, input string, addHistory bool) {
	if input == "" {
		return
	}

	pageURL := f.processInput(input)

	// Annuler un chargement en cours sur cet onglet
	if tab.cancel != nil {
		tab.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	tab.cancel = cancel
	tab.loading = true
	tab.pageURL = pageURL

	// Mettre à jour l'URL bar si c'est l'onglet actif
	if f.currentFoxTab() == tab {
		f.urlEntry.SetText(pageURL)
		f.stopBtn.Enable()
	}

	f.setTabTitle(tab, "Chargement...")
	f.guardLabel.SetText("[AI Guard] Analyse...")

	// Pages about: (pas de réseau)
	if strings.HasPrefix(pageURL, "about:") {
		tab.loading = false
		tab.cancel = nil
		f.showAboutPage(tab)
		if f.currentFoxTab() == tab {
			f.stopBtn.Disable()
		}
		return
	}

	go func() {
		defer func() {
			tab.loading = false
			tab.cancel = nil
			if f.currentFoxTab() == tab {
				f.stopBtn.Disable()
			}
		}()

		resp, err := f.client.FetchWithContext(ctx, pageURL)
		if err != nil {
			if ctx.Err() != nil {
				f.guardLabel.SetText("[AI Guard] Arrêté")
				f.setTabTitle(tab, "Arrêté")
			} else {
				f.guardLabel.SetText("[AI Guard] Erreur réseau")
				f.setTabTitle(tab, "Erreur")
				if f.currentFoxTab() == tab {
					dialog.ShowError(fmt.Errorf("Impossible de charger :\n%s", err), f.window)
				}
			}
			return
		}

		// Utiliser l'URL finale (après redirections)
		if resp.FinalURL != "" && resp.FinalURL != pageURL {
			tab.pageURL = resp.FinalURL
			if f.currentFoxTab() == tab {
				f.urlEntry.SetText(resp.FinalURL)
			}
		}

		// Statut HTTP
		tlsInfo := resp.TLSVersion
		if tlsInfo == "" {
			tlsInfo = "HTTP"
		}
		proto := "HTTP"
		if strings.HasPrefix(tab.pageURL, "https") {
			proto = "HTTPS"
		}
		statusText := fmt.Sprintf("%s %d | %s | %s", proto, resp.StatusCode, tlsInfo, resp.Duration.Round(time.Millisecond))
		if f.profile != nil {
			stats := f.profile.Vault().Stats()
			statusText = fmt.Sprintf("[FoxChain] %d blocs | %s", stats["total"], statusText)
		}
		if f.currentFoxTab() == tab {
			f.statusLabel.SetText(statusText)
		}

		// AI Guard
		guardResult := f.guard.AnalyzePage(resp.Body, tab.pageURL)
		if f.currentFoxTab() == tab {
			f.updateGuardStatus(guardResult)
		}

		// Parser le HTML
		doc, err := engine.Parse(resp.Body)
		if err != nil {
			f.setTabTitle(tab, "Erreur parsing")
			return
		}

		if guardResult.Blocked {
			blockedView := buildBlockedPage(guardResult)
			tab.contentBox.Objects = []fyne.CanvasObject{blockedView}
			tab.contentBox.Refresh()
			tab.scroll.ScrollToTop()
			f.setTabTitle(tab, "Page bloquée")
			return
		}

		// Extraire le titre HTML initial
		pageTitle := extractTitle(doc)
		if pageTitle == "" {
			if parsed, err2 := url.Parse(tab.pageURL); err2 == nil {
				pageTitle = parsed.Hostname()
			} else {
				pageTitle = tab.pageURL
			}
		}

		// ── CSS Engine ──
		// Résoudre les styles CSS avant JS (les mutations JS peuvent en dépendre)
		{
			cssTexts := css.ExtractStyleSheets(doc)
			cssSheets := make([]*css.StyleSheet, 0, len(cssTexts))
			for _, text := range cssTexts {
				cssSheets = append(cssSheets, css.ParseSheet(text))
			}
			css.ResolveStyles(doc, cssSheets)
			if f.currentFoxTab() == tab && len(cssSheets) > 0 {
				cssSummary := fmt.Sprintf(" | CSS: %d feuilles", len(cssSheets))
				f.statusLabel.SetText(f.statusLabel.Text + cssSummary)
			}
		}

		// ── JavaScript Engine ──
		// Exécuter les scripts inline seulement si la page n'est pas bloquée
		if !guardResult.Blocked {
			scripts := aiguard.ExtractScripts(resp.Body)
			if len(scripts) > 0 {
				jsResult := f.jsEngine.Execute(scripts, tab.pageURL, doc)

				// Redirect JS (window.location.href = '...')
				if jsResult.Redirect != "" && addHistory {
					f.navigateInTab(tab, jsResult.Redirect, true)
					return
				}

				// Titre modifié par document.title = '...'
				if jsResult.Title != "" {
					pageTitle = jsResult.Title
				}

				// Appliquer les mutations DOM avant le rendu final
				if len(jsResult.Changes) > 0 {
					jsengine.ApplyChanges(doc, jsResult.Changes)
				}

				// Afficher les infos JS dans la barre de statut
				if f.currentFoxTab() == tab && jsResult.Executed > 0 {
					extra := fmt.Sprintf(" | JS: %d scripts", jsResult.Executed)
					if len(jsResult.Changes) > 0 {
						extra += fmt.Sprintf(", %d mutations", len(jsResult.Changes))
					}
					f.statusLabel.SetText(f.statusLabel.Text + extra)
				}
			}
		}

		// Rendu structuré GUI (après mutations JS)
		guiResult := engine.RenderForGUI(doc)

		// Mettre à jour le contenu de l'onglet
		pageView := buildPageView(guiResult, func(linkURL string) {
			resolved := tab.session.ResolveURL(linkURL)
			f.navigateInTab(tab, resolved, true)
		})

		tab.contentBox.Objects = []fyne.CanvasObject{pageView}
		tab.contentBox.Refresh()
		tab.scroll.ScrollToTop()
		tab.links = guiResult.Links
		f.setTabTitle(tab, pageTitle)

		if f.currentFoxTab() == tab {
			f.linkList.Refresh()
		}

		// Historique de session (back/forward)
		if addHistory {
			pageLinks := make([]browser.PageLink, len(guiResult.Links))
			for i, l := range guiResult.Links {
				pageLinks[i] = browser.PageLink{Index: l.Index, Text: l.Text, URL: l.URL}
			}
			tab.session.Push(browser.Page{URL: tab.pageURL, Title: pageTitle, Links: pageLinks})
		}

		if f.currentFoxTab() == tab {
			f.updateNavButtons(tab)
		}

		// Sauvegarder dans l'historique FoxChain
		if f.profile != nil {
			f.profile.Vault().AddHistory(tab.pageURL, pageTitle)
			f.profile.Save()
		}
	}()
}

// processInput détermine l'URL finale à partir de l'entrée utilisateur.
// - URL complète (http:// ou https://) → retournée telle quelle
// - about: → retournée telle quelle
// - Ressemble à un domaine → préfixe https://
// - Sinon → recherche DuckDuckGo
func (f *FoxGUI) processInput(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return "about:fox"
	}
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return input
	}
	if strings.HasPrefix(input, "about:") {
		return input
	}
	// Ressemble à un domaine : pas d'espaces, contient un point ou est localhost
	if !strings.Contains(input, " ") &&
		(strings.Contains(input, ".") || strings.HasPrefix(input, "localhost")) {
		return "https://" + input
	}
	// Requête de recherche → DuckDuckGo (respect de la vie privée)
	return "https://duckduckgo.com/?q=" + url.QueryEscape(input) + "&kp=-1&kl=fr-fr"
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
		label = "DANGER"
	}
	if result.Blocked {
		icon = "■"
		label = "BLOQUÉ"
	}

	text := fmt.Sprintf("[AI Guard] %s %s (%.0f%%)", icon, label, result.Score*100)
	if result.Details != "" && result.Score > 0.1 {
		if len(result.Details) > 60 {
			text += " — " + result.Details[:57] + "..."
		} else {
			text += " — " + result.Details
		}
	}
	f.guardLabel.SetText(text)
}

// ══════════════════════════════════════
// PAGE ABOUT / NOUVEAU ONGLET
// ══════════════════════════════════════

// showAboutPage affiche la page d'accueil Fox dans un onglet.
func (f *FoxGUI) showAboutPage(tab *FoxTab) {
	tab.pageURL = "about:fox"
	tab.title = "Fox Browser"

	content := f.buildNewTabContent()
	tab.contentBox.Objects = []fyne.CanvasObject{content}
	tab.contentBox.Refresh()

	f.setTabTitle(tab, "Accueil")
	if f.currentFoxTab() == tab {
		f.urlEntry.SetText("about:fox")
		f.backBtn.Disable()
		f.fwdBtn.Disable()
		f.stopBtn.Disable()
		f.guardLabel.SetText("[AI Guard] En attente")
	}
}

// buildNewTabContent construit le contenu de la page d'accueil.
func (f *FoxGUI) buildNewTabContent() fyne.CanvasObject {
	// Logo
	logo := canvas.NewText("Fox Browser", color.NRGBA{R: 255, G: 140, B: 0, A: 255})
	logo.TextSize = 36
	logo.TextStyle = fyne.TextStyle{Bold: true}
	logo.Alignment = fyne.TextAlignCenter

	tagline := canvas.NewText("Naviguez libre. Naviguez souverain.", color.NRGBA{R: 180, G: 180, B: 190, A: 255})
	tagline.TextSize = 15
	tagline.Alignment = fyne.TextAlignCenter

	// Barre de recherche rapide
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Rechercher ou entrer une adresse...")
	searchBtn := widget.NewButtonWithIcon("Aller", theme.NavigateNextIcon(), func() {
		f.navigate(searchEntry.Text)
	})
	searchEntry.OnSubmitted = func(q string) { f.navigate(q) }

	searchBox := container.NewBorder(nil, nil, nil, searchBtn, searchEntry)

	// Indicateurs de statut
	badges := container.NewHBox(
		layout.NewSpacer(),
		buildBadge("AI Guard", color.NRGBA{R: 50, G: 150, B: 50, A: 220}),
		buildBadge("FoxChain", color.NRGBA{R: 80, G: 80, B: 200, A: 220}),
		buildBadge("FoxOTA P2P", color.NRGBA{R: 180, G: 80, B: 20, A: 220}),
		buildBadge("EUPL v1.2", color.NRGBA{R: 100, G: 100, B: 100, A: 220}),
		layout.NewSpacer(),
	)

	top := container.NewVBox(
		widget.NewSeparator(),
		container.NewCenter(logo),
		container.NewCenter(tagline),
		widget.NewSeparator(),
		container.NewPadded(searchBox),
		widget.NewSeparator(),
		container.NewPadded(badges),
		widget.NewSeparator(),
	)

	// Favoris rapides (si profil chargé)
	var bottomContent fyne.CanvasObject
	if len(f.bookmarks) > 0 {
		bTitle := widget.NewLabelWithStyle("Favoris", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		max := 12
		if len(f.bookmarks) < max {
			max = len(f.bookmarks)
		}
		btns := make([]fyne.CanvasObject, max)
		for i := 0; i < max; i++ {
			bm := f.bookmarks[i]
			label := bm.Title
			if len(label) > 18 {
				label = label[:15] + "..."
			}
			bmURL := bm.URL
			btns[i] = widget.NewButton(label, func() { f.navigate(bmURL) })
		}
		grid := container.NewGridWithColumns(4, btns...)
		bottomContent = container.NewVBox(
			container.NewPadded(bTitle),
			container.NewPadded(grid),
		)
	} else {
		hint := widget.NewLabelWithStyle(
			"Ajoutez des favoris avec le bouton ★ dans la barre d'adresse",
			fyne.TextAlignCenter,
			fyne.TextStyle{Italic: true},
		)
		bottomContent = container.NewCenter(hint)
	}

	versionLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("v%s", version),
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	return container.NewVBox(
		top,
		bottomContent,
		layout.NewSpacer(),
		widget.NewSeparator(),
		container.NewCenter(versionLabel),
	)
}

// buildBadge crée un badge coloré pour la page d'accueil.
func buildBadge(text string, bg color.Color) fyne.CanvasObject {
	rect := canvas.NewRectangle(bg)
	rect.CornerRadius = 4
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewStack(rect, container.NewPadded(label))
}

// buildBlockedPage construit la vue d'une page bloquée par AI Guard.
func buildBlockedPage(result *aiguard.AnalysisResult) fyne.CanvasObject {
	title := canvas.NewText("Page bloquée par AI Guard", color.NRGBA{R: 220, G: 50, B: 50, A: 255})
	title.TextSize = 22
	title.TextStyle = fyne.TextStyle{Bold: true}

	cat := widget.NewLabel(fmt.Sprintf("Catégorie : %s", string(result.Category)))
	cat.TextStyle = fyne.TextStyle{Bold: true}

	details := widget.NewLabel(result.Details)
	details.Wrapping = fyne.TextWrapWord

	score := widget.NewLabel(fmt.Sprintf("Score de menace : %.0f%%", result.Score*100))

	bg := canvas.NewRectangle(color.NRGBA{R: 40, G: 10, B: 10, A: 255})
	content := container.NewVBox(
		container.NewCenter(title),
		widget.NewSeparator(),
		cat, details, score,
	)
	return container.NewStack(bg, container.NewPadded(content))
}

// ══════════════════════════════════════
// FERMETURE
// ══════════════════════════════════════

func (f *FoxGUI) onClose() {
	// Annuler tous les chargements en cours
	for _, tab := range f.foxTabs {
		if tab.cancel != nil {
			tab.cancel()
		}
	}

	// Sauvegarder l'état des onglets dans FoxChain
	if f.profile != nil {
		var urls []string
		for _, tab := range f.foxTabs {
			if tab.pageURL != "" && !strings.HasPrefix(tab.pageURL, "about:") {
				urls = append(urls, tab.pageURL)
			}
		}
		if len(urls) > 0 {
			f.profile.Vault().SaveTabState(urls, len(urls)-1)
			f.profile.Save()
		}
	}

	f.app.Quit()
}

// ══════════════════════════════════════
// UTILITAIRES
// ══════════════════════════════════════

// extractTitle extrait le contenu de la balise <title> du DOM.
func extractTitle(el *engine.Element) string {
	if el == nil {
		return ""
	}
	if el.Tag == "title" && el.Text != "" {
		return strings.TrimSpace(el.Text)
	}
	for _, child := range el.Children {
		if t := extractTitle(child); t != "" {
			return t
		}
	}
	return ""
}
