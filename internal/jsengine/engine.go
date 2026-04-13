// Package jsengine implémente le moteur JavaScript de HexaNaute.
//
// Basé sur Goja (ECMAScript 5.1 + ES2020 partiel) en Go pur.
// Aucune dépendance V8/Node/SpiderMonkey — souveraineté totale.
//
// Architecture de sécurité :
//   - Sandbox isolé par page (nouveau Runtime par exécution)
//   - Timeout strictement appliqué via vm.Interrupt()
//   - Aucun accès système (pas de require, pas de process, pas de fs)
//   - Mutations DOM tracées puis appliquées après exécution
//   - Scripts externes (src=...) non téléchargés en v0.2.0
package jsengine

import (
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"

	"github.com/51TH-FireFox13/hexanaute/internal/engine"
)

// Config configure le moteur JavaScript.
type Config struct {
	MaxDuration   time.Duration // temps max d'exécution par page
	MaxScripts    int           // nombre max de scripts exécutés par page
	MaxCallbacks  int           // nombre max de callbacks timers exécutés
	Debug         bool          // log console.log vers stdout
	AllowRedirect bool          // autoriser les redirects JS
}

// DefaultConfig retourne une config sécurisée par défaut.
func DefaultConfig() Config {
	return Config{
		MaxDuration:   3 * time.Second,
		MaxScripts:    30,
		MaxCallbacks:  20,
		Debug:         false,
		AllowRedirect: true,
	}
}

// DOMChange représente une mutation DOM produite par un script.
type DOMChange struct {
	Selector string // "#id", ".class", "tag" ou "tag[attr=val]"
	Property string // "textContent", "innerHTML", "hidden", "style.display", "class.add", "class.remove", "attr"
	AttrName string // nom d'attribut (pour Property="attr")
	Value    string
}

// ExecResult est le résultat de l'exécution JS d'une page.
type ExecResult struct {
	Redirect string      // URL si window.location.href a changé
	Title    string      // Nouveau titre si document.title a changé
	Changes  []DOMChange // Mutations DOM à appliquer
	Errors   []string    // Erreurs JS non fatales
	Executed int         // Scripts exécutés avec succès
	Blocked  int         // Scripts bloqués (timeout, erreur)
	TimedOut bool
}

// Engine est le moteur JavaScript souverain de HexaNaute.
type Engine struct {
	config Config
}

// New crée un moteur JS avec la config donnée.
func New(config Config) *Engine {
	return &Engine{config: config}
}

// Execute exécute une liste de scripts inline dans un sandbox isolé.
// root est l'arbre DOM après parsing HTML.
// Retourne les mutations à appliquer et les métadonnées.
func (e *Engine) Execute(scripts []string, baseURL string, root *engine.Element) *ExecResult {
	result := &ExecResult{
		Changes: make([]DOMChange, 0),
		Errors:  make([]string, 0),
	}

	// Filtrer les scripts vides
	filtered := make([]string, 0, len(scripts))
	for _, s := range scripts {
		if strings.TrimSpace(s) != "" {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return result
	}

	// Limiter le nombre de scripts
	max := e.config.MaxScripts
	if len(filtered) < max {
		max = len(filtered)
	}

	// Créer une VM Goja isolée pour cette page
	vm := goja.New()

	// Timeout global via interruption
	interruptCh := make(chan struct{})
	go func() {
		select {
		case <-time.After(e.config.MaxDuration):
			vm.Interrupt("fox: script timeout")
			result.TimedOut = true
		case <-interruptCh:
		}
	}()
	defer close(interruptCh)

	// Installer les APIs globales
	setupConsole(vm, e.config.Debug)
	setupNavigator(vm)
	setupStorage(vm)
	runTimers := setupTimers(vm, e.config.MaxCallbacks)
	setupWindow(vm, baseURL, result, e.config.AllowRedirect)
	setupDocument(vm, root, baseURL, result)
	setupCreepJSDefenses(vm) // anti-fingerprinting CreepJS (canvas, webgl, audio, observers, workers)

	// Exécuter les scripts en ordre
	for i := 0; i < max; i++ {
		if result.TimedOut {
			result.Blocked += max - i
			break
		}

		script := filtered[i]
		_, err := vm.RunString(script)
		if err != nil {
			if isInterrupt(err) {
				result.TimedOut = true
				result.Blocked++
				break
			}
			// Erreur JS non fatale : on log et on continue
			result.Errors = append(result.Errors, fmt.Sprintf("[js] script[%d]: %s", i, truncate(err.Error(), 200)))
			result.Blocked++
		} else {
			result.Executed++
		}
	}

	// Exécuter les callbacks timers différés (setTimeout, etc.)
	if !result.TimedOut {
		runTimers(vm)
	}

	return result
}

// ApplyChanges applique les mutations DOM au DOM Fox.
func ApplyChanges(root *engine.Element, changes []DOMChange) {
	for _, ch := range changes {
		el := findBySelector(root, ch.Selector)
		if el == nil {
			continue
		}
		applyChange(el, ch)
	}
}

func findBySelector(root *engine.Element, selector string) *engine.Element {
	return engine.QuerySelector(root, selector)
}

func applyChange(el *engine.Element, ch DOMChange) {
	switch ch.Property {
	case "textContent", "innerText":
		el.Text = ch.Value
		el.Children = el.Children[:0]
	case "innerHTML":
		frag := engine.ParseFragment(ch.Value)
		el.Text = frag.Text
		el.Children = frag.Children
	case "hidden":
		engine.SetHidden(el, ch.Value == "true")
	case "style.display":
		engine.SetHidden(el, ch.Value == "none" || ch.Value == "")
	case "style.visibility":
		engine.SetHidden(el, ch.Value == "hidden" || ch.Value == "collapse")
	case "class.add":
		engine.AddClass(el, ch.Value)
	case "class.remove":
		engine.RemoveClass(el, ch.Value)
	case "attr":
		if el.Attrs == nil {
			el.Attrs = make(map[string]string)
		}
		el.Attrs[ch.AttrName] = ch.Value
	}
}

func isInterrupt(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "interrupted") || strings.Contains(msg, "timeout")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
