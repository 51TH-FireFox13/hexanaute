// Package network gère les requêtes HTTP du navigateur Fox.
package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// Response encapsule la réponse HTTP avec les métadonnées utiles.
type Response struct {
	StatusCode  int
	ContentType string
	Body        []byte
	URL         string
	FinalURL    string // URL finale après redirections
	TLSVersion  string
	Headers     http.Header
	Duration    time.Duration
}

// Client est le client HTTP de Fox Browser.
type Client struct {
	http *http.Client
	jar  *cookiejar.Jar
}

// NewClient crée un nouveau client HTTP sécurisé avec cookie jar.
func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		jar: jar,
		http: &http.Client{
			Jar:     jar,
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
				MaxIdleConns:       100,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: false,
				ForceAttemptHTTP2:  true,
			},
		},
	}
}

// Fetch récupère une page web (sans annulation).
func (c *Client) Fetch(rawURL string) (*Response, error) {
	return c.FetchWithContext(context.Background(), rawURL)
}

// FetchWithContext récupère une page avec support d'annulation via contexte.
func (c *Client) FetchWithContext(ctx context.Context, rawURL string) (*Response, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("requête invalide: %w", err)
	}

	req.Header.Set("User-Agent", "FoxBrowser/0.1.0 (Renard; souverain)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.5")
	req.Header.Set("DNT", "1")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("échec connexion: %w", err)
	}
	defer resp.Body.Close()

	// Détecter l'encodage et convertir en UTF-8
	reader := resp.Body
	contentType := resp.Header.Get("Content-Type")
	enc := detectEncoding(contentType)
	if enc != "" && !strings.EqualFold(enc, "utf-8") {
		e, _ := charset.Lookup(enc)
		if e != nil {
			reader = io.NopCloser(transform.NewReader(resp.Body, e.NewDecoder()))
		}
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("échec lecture: %w", err)
	}

	// Fallback : détecter l'encodage depuis le contenu HTML
	if enc == "" || strings.EqualFold(enc, "utf-8") {
		e, name, _ := charset.DetermineEncoding(body, contentType)
		if name != "utf-8" && e != nil {
			converted, _, convErr := transform.Bytes(e.NewDecoder(), body)
			if convErr == nil {
				body = converted
			}
		}
	}

	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	result := &Response{
		StatusCode:  resp.StatusCode,
		ContentType: contentType,
		Body:        body,
		URL:         rawURL,
		FinalURL:    finalURL,
		Headers:     resp.Header,
		Duration:    time.Since(start),
	}

	if resp.TLS != nil {
		switch resp.TLS.Version {
		case tls.VersionTLS13:
			result.TLSVersion = "TLS 1.3"
		case tls.VersionTLS12:
			result.TLSVersion = "TLS 1.2"
		default:
			result.TLSVersion = "TLS"
		}
	}

	return result, nil
}

// ClearCookies vide le cookie jar (ex : déconnexion globale).
func (c *Client) ClearCookies() {
	jar, _ := cookiejar.New(nil)
	c.jar = jar
	c.http.Jar = jar
}

// detectEncoding extrait le charset du Content-Type header.
func detectEncoding(contentType string) string {
	if contentType == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return params["charset"]
}
