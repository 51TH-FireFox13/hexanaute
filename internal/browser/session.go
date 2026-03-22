// Package browser gère les sessions de navigation (historique, onglets).
package browser

import (
	"net/url"
	"strings"
)

// Page représente une page visitée.
type Page struct {
	URL     string
	Title   string
	Content string
	Links   []PageLink
}

// PageLink est un lien dans une page avec son index.
type PageLink struct {
	Index int
	Text  string
	URL   string
}

// Session gère l'historique de navigation.
type Session struct {
	History []Page
	pos     int // position actuelle dans l'historique
}

// NewSession crée une nouvelle session de navigation.
func NewSession() *Session {
	return &Session{
		History: make([]Page, 0, 32),
		pos:     -1,
	}
}

// Push ajoute une page à l'historique.
func (s *Session) Push(page Page) {
	// Tronquer l'historique si on a fait des retours
	if s.pos >= 0 && s.pos < len(s.History)-1 {
		s.History = s.History[:s.pos+1]
	}
	s.History = append(s.History, page)
	s.pos = len(s.History) - 1
}

// Current retourne la page actuelle.
func (s *Session) Current() *Page {
	if s.pos < 0 || s.pos >= len(s.History) {
		return nil
	}
	return &s.History[s.pos]
}

// Back recule d'une page. Retourne nil si impossible.
func (s *Session) Back() *Page {
	if s.pos <= 0 {
		return nil
	}
	s.pos--
	return &s.History[s.pos]
}

// Forward avance d'une page. Retourne nil si impossible.
func (s *Session) Forward() *Page {
	if s.pos >= len(s.History)-1 {
		return nil
	}
	s.pos++
	return &s.History[s.pos]
}

// CanBack indique si on peut reculer.
func (s *Session) CanBack() bool {
	return s.pos > 0
}

// CanForward indique si on peut avancer.
func (s *Session) CanForward() bool {
	return s.pos < len(s.History)-1
}

// Len retourne le nombre de pages dans l'historique.
func (s *Session) Len() int {
	return len(s.History)
}

// ResolveURL résout une URL relative par rapport à la page courante.
func (s *Session) ResolveURL(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	current := s.Current()
	if current == nil {
		if !strings.HasPrefix(href, "http") {
			return "https://" + href
		}
		return href
	}

	base, err := url.Parse(current.URL)
	if err != nil {
		return href
	}

	ref, err := url.Parse(href)
	if err != nil {
		return href
	}

	return base.ResolveReference(ref).String()
}
