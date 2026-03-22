// Package network gère les requêtes HTTP du navigateur Fox.
package network

import (
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net/http"
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
	TLSVersion  string
	Headers     http.Header
	Duration    time.Duration
}

// Client est le client HTTP de Fox Browser.
type Client struct {
	http *http.Client
}

// NewClient crée un nouveau client HTTP sécurisé.
func NewClient() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
				ForceAttemptHTTP2:   true,
			},
		},
	}
}

// Fetch récupère une page web.
func (c *Client) Fetch(url string) (*Response, error) {
	start := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("requête invalide: %w", err)
	}

	req.Header.Set("User-Agent", "FoxBrowser/0.0.1")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.5")

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

	// Fallback : détecter l'encodage depuis le contenu HTML si pas dans le header
	if enc == "" || strings.EqualFold(enc, "utf-8") {
		// Vérifier si c'est vraiment de l'UTF-8 valide, sinon essayer de détecter
		e, name, _ := charset.DetermineEncoding(body, contentType)
		if name != "utf-8" && e != nil {
			converted, _, err := transform.Bytes(e.NewDecoder(), body)
			if err == nil {
				body = converted
			}
		}
	}

	result := &Response{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        body,
		URL:         url,
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
