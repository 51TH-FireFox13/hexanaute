package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/51TH-FireFox13/fox-browser/internal/aiguard"
	"github.com/51TH-FireFox13/fox-browser/internal/browser"
	"github.com/51TH-FireFox13/fox-browser/internal/engine"
	"github.com/51TH-FireFox13/fox-browser/internal/foxchain"
	"github.com/51TH-FireFox13/fox-browser/internal/foxota"
	"github.com/51TH-FireFox13/fox-browser/internal/network"
	"github.com/51TH-FireFox13/fox-browser/internal/ui"
)

const (
	version    = "0.0.4"
	versionNum = 4 // numéro monotone croissant pour anti-downgrade

	// Clé publique de signature des releases (sera générée avec fox-sign genkey)
	// Pour l'instant, placeholder — sera remplacée par la vraie clé.
	signingPubKeyHex = ""
)

var profile *foxchain.Profile

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		printVersion()
		return
	}

	client := network.NewClient()
	guard := aiguard.NewGuard(0.7)
	session := browser.NewSession()

	ui.Banner(version)

	// Initialiser FoxChain
	initFoxChain()

	// URL en argument
	if len(os.Args) > 1 {
		url := normalizeURL(os.Args[1])
		navigate(client, guard, session, url)
	}

	// Boucle interactive
	for {
		page := session.Current()
		input := ui.PromptWithState(page != nil, session.CanBack(), session.CanForward(), session.Len())
		if input == "" {
			continue
		}

		switch {
		case input == "quitter" || input == "quit" || input == "exit" || input == "q":
			saveOnExit(session)
			fmt.Println("\n[Fox] À bientôt ! 🦊")
			return

		case input == "aide" || input == "help" || input == "?":
			ui.HelpFull()

		case input == "version":
			printVersion()

		case input == "retour" || input == "back" || input == "b":
			p := session.Back()
			if p == nil {
				fmt.Println("[Fox] Pas de page précédente.")
			} else {
				fmt.Printf("\n[Fox] ◀ Retour à %s\n", p.URL)
				ui.PageContent(p.Content)
				ui.ShowLinks(p.Links)
			}

		case input == "avancer" || input == "forward" || input == "f":
			p := session.Forward()
			if p == nil {
				fmt.Println("[Fox] Pas de page suivante.")
			} else {
				fmt.Printf("\n[Fox] ▶ Avancer à %s\n", p.URL)
				ui.PageContent(p.Content)
				ui.ShowLinks(p.Links)
			}

		case input == "liens" || input == "links" || input == "l":
			if page != nil && len(page.Links) > 0 {
				ui.ShowLinks(page.Links)
			} else {
				fmt.Println("[Fox] Aucun lien sur cette page.")
			}

		case input == "historique" || input == "history" || input == "h":
			ui.ShowHistory(session.History)

		case input == "recharger" || input == "reload" || input == "r":
			if page != nil {
				navigate(client, guard, session, page.URL)
			} else {
				fmt.Println("[Fox] Aucune page à recharger.")
			}

		// ── FoxChain : Favoris ──
		case input == "fav" || input == "favoris" || input == "bookmarks":
			cmdListBookmarks()

		case strings.HasPrefix(input, "fav+ ") || strings.HasPrefix(input, "+fav "):
			cmdAddBookmark(input, page)

		case strings.HasPrefix(input, "fav- "):
			cmdRemoveBookmark(input)

		// ── FoxChain : Mots de passe ──
		case input == "mdp" || input == "passwords":
			cmdListPasswords()

		case strings.HasPrefix(input, "mdp+ "):
			cmdAddPassword(input)

		case strings.HasPrefix(input, "mdp? "):
			cmdGetPassword(input)

		case strings.HasPrefix(input, "mdp- "):
			cmdRemovePassword(input)

		// ── FoxOTA ──
		case input == "update" || input == "maj":
			cmdCheckUpdate()

		// ── FoxChain : Stats / Intégrité ──
		case input == "chain" || input == "foxchain":
			cmdChainInfo()

		case input == "verify" || input == "intégrité":
			cmdVerifyChain()

		default:
			// Essayer d'interpréter comme un numéro de lien
			if num, err := strconv.Atoi(input); err == nil && page != nil {
				for _, link := range page.Links {
					if link.Index == num {
						resolved := session.ResolveURL(link.URL)
						fmt.Printf("\n[Fox] Lien [%d] → %s\n", num, resolved)
						navigate(client, guard, session, resolved)
						goto nextLoop
					}
				}
				fmt.Printf("[Fox] Lien [%d] introuvable.\n", num)
			} else {
				url := normalizeURL(input)
				navigate(client, guard, session, url)
			}
		}
	nextLoop:
	}
}

// ── FoxChain Init ──

func foxchainDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".fox")
}

func initFoxChain() {
	dir := foxchainDir()

	if foxchain.Exists(dir) {
		// Profil existant
		fmt.Print("[FoxChain] Passphrase : ")
		pass := readPassword()
		p, err := foxchain.OpenProfile(dir, pass)
		if err != nil {
			fmt.Printf("[FoxChain] Erreur : %s\n", err)
			fmt.Println("[FoxChain] Navigation sans profil (données non persistées)")
			return
		}
		profile = p
		stats := profile.Vault().Stats()
		fmt.Printf("[FoxChain] Profil chargé — %d blocs, intégrité OK ✓\n", stats["total"])

		// Afficher les onglets sauvegardés si disponibles
		if state, err := profile.Vault().LoadTabState(); err == nil && len(state.Tabs) > 0 {
			fmt.Printf("[FoxChain] %d onglet(s) sauvegardé(s) depuis la dernière session\n", len(state.Tabs))
			for i, tab := range state.Tabs {
				fmt.Printf("  %d. %s\n", i+1, tab)
			}
		}
	} else {
		// Nouveau profil
		fmt.Println("[FoxChain] Aucun profil trouvé.")
		fmt.Print("[FoxChain] Créer un profil ? (o/n) : ")
		answer := readLine()
		if answer != "o" && answer != "oui" && answer != "y" && answer != "yes" {
			fmt.Println("[FoxChain] Navigation sans profil (données non persistées)")
			return
		}

		fmt.Print("[FoxChain] Choisissez une passphrase : ")
		pass := readPassword()
		if len(pass) < 8 {
			fmt.Println("[FoxChain] Passphrase trop courte (minimum 8 caractères)")
			return
		}

		fmt.Print("[FoxChain] Confirmez la passphrase : ")
		pass2 := readPassword()
		if pass != pass2 {
			fmt.Println("[FoxChain] Les passphrases ne correspondent pas")
			return
		}

		fmt.Println("[FoxChain] Génération des clés (Argon2id + Ed25519)...")
		p, err := foxchain.CreateProfile(dir, pass)
		if err != nil {
			fmt.Printf("[FoxChain] Erreur : %s\n", err)
			return
		}
		profile = p
		fmt.Printf("[FoxChain] Profil créé dans %s ✓\n", dir)
		fmt.Println("[FoxChain] Votre passphrase est la seule clé d'accès — ne la perdez pas !")
	}
}

// ── Commandes FoxChain ──

func cmdListBookmarks() {
	if profile == nil {
		fmt.Println("[FoxChain] Pas de profil actif.")
		return
	}

	bookmarks, err := profile.Vault().ListBookmarks()
	if err != nil {
		fmt.Printf("[FoxChain] Erreur : %s\n", err)
		return
	}

	if len(bookmarks) == 0 {
		fmt.Println("[FoxChain] Aucun favori. Utilisez 'fav+ <titre>' sur une page pour en ajouter.")
		return
	}

	fmt.Println("\n\033[36m── Favoris ──\033[0m")
	for i, bm := range bookmarks {
		t := time.Unix(bm.AddedAt, 0).Format("02/01/2006")
		fmt.Printf("  \033[36m[%d]\033[0m %s\n      \033[90m%s — ajouté le %s\033[0m\n", i+1, bm.Title, bm.URL, t)
	}
	fmt.Println()
}

func cmdAddBookmark(input string, page *browser.Page) {
	if profile == nil {
		fmt.Println("[FoxChain] Pas de profil actif.")
		return
	}

	// fav+ <titre> ou +fav <titre>
	title := strings.TrimPrefix(input, "fav+ ")
	title = strings.TrimPrefix(title, "+fav ")
	title = strings.TrimSpace(title)

	if page == nil {
		fmt.Println("[Fox] Naviguez d'abord vers une page.")
		return
	}

	if title == "" {
		title = page.URL
	}

	if err := profile.Vault().AddBookmark(page.URL, title); err != nil {
		fmt.Printf("[FoxChain] Erreur : %s\n", err)
		return
	}
	if err := profile.Save(); err != nil {
		fmt.Printf("[FoxChain] Erreur sauvegarde : %s\n", err)
		return
	}
	fmt.Printf("[FoxChain] Favori ajouté : %s ✓\n", title)
}

func cmdRemoveBookmark(input string) {
	if profile == nil {
		fmt.Println("[FoxChain] Pas de profil actif.")
		return
	}

	arg := strings.TrimSpace(strings.TrimPrefix(input, "fav- "))

	// Si c'est un numéro, trouver l'URL correspondante
	if num, err := strconv.Atoi(arg); err == nil {
		bookmarks, _ := profile.Vault().ListBookmarks()
		if num < 1 || num > len(bookmarks) {
			fmt.Printf("[FoxChain] Favori [%d] introuvable.\n", num)
			return
		}
		arg = bookmarks[num-1].URL
	}

	if err := profile.Vault().RemoveBookmark(arg); err != nil {
		fmt.Printf("[FoxChain] Erreur : %s\n", err)
		return
	}
	if err := profile.Save(); err != nil {
		fmt.Printf("[FoxChain] Erreur sauvegarde : %s\n", err)
		return
	}
	fmt.Println("[FoxChain] Favori supprimé ✓")
}

func cmdListPasswords() {
	if profile == nil {
		fmt.Println("[FoxChain] Pas de profil actif.")
		return
	}

	creds, err := profile.Vault().ListCredentials()
	if err != nil {
		fmt.Printf("[FoxChain] Erreur : %s\n", err)
		return
	}

	if len(creds) == 0 {
		fmt.Println("[FoxChain] Aucun mot de passe. Utilisez 'mdp+ <site> <user> <pass>' pour en ajouter.")
		return
	}

	fmt.Println("\n\033[36m── Mots de passe ──\033[0m")
	for i, cred := range creds {
		masked := strings.Repeat("•", len(cred.Password))
		t := time.Unix(cred.AddedAt, 0).Format("02/01/2006")
		fmt.Printf("  \033[36m[%d]\033[0m %s\n      \033[90muser: %s | pass: %s | %s\033[0m\n", i+1, cred.Site, cred.Username, masked, t)
	}
	fmt.Printf("\n  \033[90mUtilisez 'mdp? <site>' pour révéler un mot de passe\033[0m\n\n")
}

func cmdAddPassword(input string) {
	if profile == nil {
		fmt.Println("[FoxChain] Pas de profil actif.")
		return
	}

	parts := strings.Fields(strings.TrimPrefix(input, "mdp+ "))
	if len(parts) < 3 {
		fmt.Println("[FoxChain] Usage : mdp+ <site> <utilisateur> <mot_de_passe>")
		return
	}

	site, user, pass := parts[0], parts[1], strings.Join(parts[2:], " ")

	if err := profile.Vault().AddCredential(site, user, pass); err != nil {
		fmt.Printf("[FoxChain] Erreur : %s\n", err)
		return
	}
	if err := profile.Save(); err != nil {
		fmt.Printf("[FoxChain] Erreur sauvegarde : %s\n", err)
		return
	}
	fmt.Printf("[FoxChain] Identifiant ajouté pour %s ✓\n", site)
}

func cmdGetPassword(input string) {
	if profile == nil {
		fmt.Println("[FoxChain] Pas de profil actif.")
		return
	}

	site := strings.TrimSpace(strings.TrimPrefix(input, "mdp? "))

	// Si c'est un numéro
	if num, err := strconv.Atoi(site); err == nil {
		creds, _ := profile.Vault().ListCredentials()
		if num < 1 || num > len(creds) {
			fmt.Printf("[FoxChain] Entrée [%d] introuvable.\n", num)
			return
		}
		cred := creds[num-1]
		fmt.Printf("[FoxChain] %s — user: %s | pass: \033[33m%s\033[0m\n", cred.Site, cred.Username, cred.Password)
		return
	}

	cred, err := profile.Vault().GetCredential(site)
	if err != nil {
		fmt.Printf("[FoxChain] %s\n", err)
		return
	}
	fmt.Printf("[FoxChain] %s — user: %s | pass: \033[33m%s\033[0m\n", cred.Site, cred.Username, cred.Password)
}

func cmdRemovePassword(input string) {
	if profile == nil {
		fmt.Println("[FoxChain] Pas de profil actif.")
		return
	}

	arg := strings.TrimSpace(strings.TrimPrefix(input, "mdp- "))

	if num, err := strconv.Atoi(arg); err == nil {
		creds, _ := profile.Vault().ListCredentials()
		if num < 1 || num > len(creds) {
			fmt.Printf("[FoxChain] Entrée [%d] introuvable.\n", num)
			return
		}
		arg = creds[num-1].Site
	}

	if err := profile.Vault().RemoveCredential(arg); err != nil {
		fmt.Printf("[FoxChain] Erreur : %s\n", err)
		return
	}
	if err := profile.Save(); err != nil {
		fmt.Printf("[FoxChain] Erreur sauvegarde : %s\n", err)
		return
	}
	fmt.Println("[FoxChain] Identifiant supprimé ✓")
}

func cmdChainInfo() {
	if profile == nil {
		fmt.Println("[FoxChain] Pas de profil actif.")
		return
	}

	stats := profile.Vault().Stats()
	chain := profile.Vault().Chain()

	fmt.Println("\n\033[36m── FoxChain ──\033[0m")
	fmt.Printf("  Répertoire  : %s\n", profile.Dir)
	fmt.Printf("  Blocs       : %d\n", stats["total"])
	fmt.Printf("  Favoris     : %d\n", stats["favoris"])
	fmt.Printf("  Mots de passe : %d\n", stats["mots_de_passe"])
	fmt.Printf("  Historique  : %d\n", stats["historique"])
	fmt.Printf("  Onglets     : %d\n", stats["états_onglets"])
	fmt.Printf("  Intégrité   : ")
	if chain.Verify() {
		fmt.Println("\033[32m✓ chaîne valide\033[0m")
	} else {
		fmt.Println("\033[31m✗ CHAÎNE COMPROMISE\033[0m")
	}
	fmt.Println()
}

func cmdCheckUpdate() {
	fmt.Println("\n[FoxOTA] Vérification des mises à jour...")

	if signingPubKeyHex == "" {
		fmt.Println("[FoxOTA] Pas de clé de signature configurée.")
		fmt.Println("[FoxOTA] Générez une clé avec : fox-sign genkey")
		fmt.Println("[FoxOTA] Puis intégrez la clé publique dans le code source.")
		return
	}

	pubKeyBytes, err := hex.DecodeString(signingPubKeyHex)
	if err != nil {
		fmt.Printf("[FoxOTA] Clé publique invalide : %s\n", err)
		return
	}

	checker := foxota.NewChecker(pubKeyBytes, versionNum, "stable")

	// Sources de vérification (à configurer avec vos URLs)
	checker.AddSource("github", "https://raw.githubusercontent.com/51TH-FireFox13/fox-browser/main/releases/manifest.json")
	checker.AddSource("mirror-1", "https://fox-browser.fr/releases/manifest.json")

	result := checker.Check()

	if result.Error != nil {
		if result.Error == foxota.ErrNoUpdate {
			fmt.Printf("[FoxOTA] ✓ Fox Browser v%s est à jour.\n", version)
		} else {
			fmt.Printf("[FoxOTA] Erreur : %s\n", result.Error)
		}
		return
	}

	if result.Available {
		fmt.Printf("[FoxOTA] Nouvelle version disponible : v%s\n", result.Manifest.VersionStr)
		fmt.Printf("[FoxOTA] Canal : %s\n", result.Manifest.Channel)
		if result.Manifest.Changelog != "" {
			fmt.Printf("[FoxOTA] Changelog : %s\n", result.Manifest.Changelog)
		}
		if result.BinaryInfo != nil {
			fmt.Printf("[FoxOTA] Taille : %.1f Mo\n", float64(result.BinaryInfo.Size)/1024/1024)
		}
		fmt.Printf("[FoxOTA] Consensus multi-source : %v\n", result.Verified)
		fmt.Println("[FoxOTA] Tapez 'update!' pour télécharger et appliquer.")
	}
}

func cmdVerifyChain() {
	if profile == nil {
		fmt.Println("[FoxChain] Pas de profil actif.")
		return
	}

	chain := profile.Vault().Chain()
	fmt.Print("[FoxChain] Vérification de l'intégrité... ")

	if chain.Verify() {
		fmt.Printf("\033[32m✓ %d blocs vérifiés, chaîne intègre\033[0m\n", chain.Len())
	} else {
		fmt.Println("\033[31m✗ INTÉGRITÉ COMPROMISE — la chaîne a été altérée !\033[0m")
	}
}

// ── Navigation ──

func navigate(client *network.Client, guard *aiguard.Guard, session *browser.Session, url string) {
	fmt.Printf("\n[Fox] Chargement de %s ...\n", url)

	resp, err := client.Fetch(url)
	if err != nil {
		fmt.Printf("[Fox] Erreur : %s\n", err)
		return
	}

	tlsInfo := resp.TLSVersion
	if tlsInfo == "" {
		tlsInfo = "pas de TLS"
	}
	ui.StatusBar(url, resp.StatusCode, tlsInfo, resp.Duration.Round(1e6).String())

	result := guard.AnalyzePage(resp.Body, url)
	ui.AIGuardStatus(result.Score, string(result.Category), result.Blocked, result.Details)

	if result.Blocked {
		fmt.Println("[AI Guard] 🛡 Page bloquée par l'IA — contenu potentiellement malveillant.")
		return
	}

	doc, err := engine.Parse(resp.Body)
	if err != nil {
		fmt.Printf("[Fox] Erreur de parsing : %s\n", err)
		return
	}

	rendered := engine.Render(doc)

	pageLinks := make([]browser.PageLink, len(rendered.Links))
	for i, l := range rendered.Links {
		pageLinks[i] = browser.PageLink{
			Index: l.Index,
			Text:  l.Text,
			URL:   l.URL,
		}
	}

	session.Push(browser.Page{
		URL:     url,
		Content: rendered.Content,
		Links:   pageLinks,
	})

	// Sauvegarder dans l'historique FoxChain
	if profile != nil {
		profile.Vault().AddHistory(url, "")
		profile.Save()
	}

	ui.PageContent(rendered.Content)

	if len(rendered.Links) > 0 {
		fmt.Printf("\033[90m  %d liens trouvés — tapez un numéro pour suivre un lien, 'liens' pour la liste\033[0m\n", len(rendered.Links))
	}
}

func saveOnExit(session *browser.Session) {
	if profile == nil {
		return
	}

	// Sauvegarder les URLs de l'historique de session comme état des onglets
	var urls []string
	for _, p := range session.History {
		urls = append(urls, p.URL)
	}
	if len(urls) > 0 {
		profile.Vault().SaveTabState(urls, len(urls)-1)
		profile.Save()
		fmt.Printf("[FoxChain] %d onglet(s) sauvegardé(s) ✓\n", len(urls))
	}
}

func normalizeURL(input string) string {
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return "https://" + input
	}
	return input
}

func printVersion() {
	fmt.Printf("Fox Browser v%s (Renard)\n", version)
	fmt.Printf("OS: %s | Arch: %s | Go: %s\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	fmt.Printf("CPUs: %d\n", runtime.NumCPU())
}

func readLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func readPassword() string {
	// En mode terminal, on pourrait masquer l'input
	// Pour l'instant, lecture simple
	return readLine()
}
