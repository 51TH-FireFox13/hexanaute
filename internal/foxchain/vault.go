package foxchain

import (
	"encoding/json"
	"fmt"
	"time"

	foxcrypto "github.com/51TH-FireFox13/fox-browser/pkg/crypto"
)

// Bookmark est un favori.
type Bookmark struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	AddedAt   int64  `json:"added_at"`
	DeletedAt int64  `json:"deleted_at,omitempty"`
}

// Credential est un identifiant stocké.
type Credential struct {
	Site      string `json:"site"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	AddedAt   int64  `json:"added_at"`
	DeletedAt int64  `json:"deleted_at,omitempty"`
}

// HistoryEntry est une entrée d'historique.
type HistoryEntry struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	VisitedAt int64  `json:"visited_at"`
}

// TabState représente l'état des onglets.
type TabState struct {
	Tabs    []string `json:"tabs"` // URLs des onglets ouverts
	Active  int      `json:"active"`
	SavedAt int64    `json:"saved_at"`
}

// DeleteMarker marque une entrée comme supprimée.
type DeleteMarker struct {
	TargetType BlockType `json:"target_type"`
	Key        string    `json:"key"` // URL pour bookmark/historique, site pour credential
}

// Vault est l'interface haut-niveau pour gérer les données FoxChain.
type Vault struct {
	chain *Chain
}

// NewVault crée un vault à partir d'une chaîne.
func NewVault(chain *Chain) *Vault {
	return &Vault{chain: chain}
}

// Chain retourne la chaîne sous-jacente.
func (v *Vault) Chain() *Chain {
	return v.chain
}

// ── Favoris ──

// AddBookmark ajoute un favori.
func (v *Vault) AddBookmark(url, title string) error {
	bm := Bookmark{
		URL:     url,
		Title:   title,
		AddedAt: time.Now().Unix(),
	}
	return v.appendEncrypted(BlockBookmark, bm)
}

// ListBookmarks retourne tous les favoris actifs (non supprimés).
func (v *Vault) ListBookmarks() ([]Bookmark, error) {
	deleted := v.getDeletedKeys(BlockBookmark)
	blocks := v.chain.BlocksByType(BlockBookmark)
	bookmarks := make([]Bookmark, 0, len(blocks))

	for _, b := range blocks {
		var bm Bookmark
		if err := v.decryptBlock(b, &bm); err != nil {
			continue
		}
		if _, ok := deleted[bm.URL]; !ok {
			bookmarks = append(bookmarks, bm)
		}
	}

	// Dédupliquer par URL (garder le plus récent)
	seen := make(map[string]bool)
	result := make([]Bookmark, 0, len(bookmarks))
	for i := len(bookmarks) - 1; i >= 0; i-- {
		if !seen[bookmarks[i].URL] {
			seen[bookmarks[i].URL] = true
			result = append([]Bookmark{bookmarks[i]}, result...)
		}
	}

	return result, nil
}

// RemoveBookmark marque un favori comme supprimé.
func (v *Vault) RemoveBookmark(url string) error {
	marker := DeleteMarker{TargetType: BlockBookmark, Key: url}
	return v.appendEncrypted(BlockDelete, marker)
}

// ── Mots de passe ──

// AddCredential ajoute un identifiant.
func (v *Vault) AddCredential(site, username, password string) error {
	cred := Credential{
		Site:     site,
		Username: username,
		Password: password,
		AddedAt:  time.Now().Unix(),
	}
	return v.appendEncrypted(BlockPassword, cred)
}

// ListCredentials retourne tous les identifiants actifs.
func (v *Vault) ListCredentials() ([]Credential, error) {
	deleted := v.getDeletedKeys(BlockPassword)
	blocks := v.chain.BlocksByType(BlockPassword)
	creds := make([]Credential, 0, len(blocks))

	for _, b := range blocks {
		var cred Credential
		if err := v.decryptBlock(b, &cred); err != nil {
			continue
		}
		if _, ok := deleted[cred.Site]; !ok {
			creds = append(creds, cred)
		}
	}

	// Dédupliquer par site+username (garder le plus récent)
	seen := make(map[string]bool)
	result := make([]Credential, 0, len(creds))
	for i := len(creds) - 1; i >= 0; i-- {
		key := creds[i].Site + "|" + creds[i].Username
		if !seen[key] {
			seen[key] = true
			result = append([]Credential{creds[i]}, result...)
		}
	}

	return result, nil
}

// GetCredential retourne l'identifiant pour un site donné.
func (v *Vault) GetCredential(site string) (*Credential, error) {
	creds, err := v.ListCredentials()
	if err != nil {
		return nil, err
	}
	for i := len(creds) - 1; i >= 0; i-- {
		if creds[i].Site == site {
			return &creds[i], nil
		}
	}
	return nil, fmt.Errorf("aucun identifiant pour %s", site)
}

// RemoveCredential marque un identifiant comme supprimé.
func (v *Vault) RemoveCredential(site string) error {
	marker := DeleteMarker{TargetType: BlockPassword, Key: site}
	return v.appendEncrypted(BlockDelete, marker)
}

// ── Historique ──

// AddHistory ajoute une entrée d'historique.
func (v *Vault) AddHistory(url, title string) error {
	entry := HistoryEntry{
		URL:       url,
		Title:     title,
		VisitedAt: time.Now().Unix(),
	}
	return v.appendEncrypted(BlockHistory, entry)
}

// ListHistory retourne les N dernières entrées d'historique.
func (v *Vault) ListHistory(limit int) ([]HistoryEntry, error) {
	blocks := v.chain.BlocksByType(BlockHistory)
	entries := make([]HistoryEntry, 0, len(blocks))

	for _, b := range blocks {
		var entry HistoryEntry
		if err := v.decryptBlock(b, &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	// Retourner les plus récents en premier
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	// Inverser l'ordre
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

// ── État des onglets ──

// SaveTabState sauvegarde l'état des onglets.
func (v *Vault) SaveTabState(urls []string, activeIndex int) error {
	state := TabState{
		Tabs:    urls,
		Active:  activeIndex,
		SavedAt: time.Now().Unix(),
	}
	return v.appendEncrypted(BlockTabState, state)
}

// LoadTabState charge le dernier état des onglets.
func (v *Vault) LoadTabState() (*TabState, error) {
	blocks := v.chain.BlocksByType(BlockTabState)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("aucun état d'onglet sauvegardé")
	}

	// Dernier bloc = état le plus récent
	var state TabState
	if err := v.decryptBlock(blocks[len(blocks)-1], &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// ── Méthodes internes ──

func (v *Vault) appendEncrypted(blockType BlockType, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("foxchain: erreur sérialisation: %w", err)
	}

	encrypted, err := foxcrypto.Encrypt(v.chain.EncKey(), jsonData)
	if err != nil {
		return fmt.Errorf("foxchain: erreur chiffrement: %w", err)
	}

	v.chain.Append(blockType, encrypted)
	return nil
}

func (v *Vault) decryptBlock(b Block, target any) error {
	plaintext, err := foxcrypto.Decrypt(v.chain.EncKey(), b.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(plaintext, target)
}

func (v *Vault) getDeletedKeys(targetType BlockType) map[string]bool {
	deleted := make(map[string]bool)
	blocks := v.chain.BlocksByType(BlockDelete)
	for _, b := range blocks {
		var marker DeleteMarker
		if err := v.decryptBlock(b, &marker); err != nil {
			continue
		}
		if marker.TargetType == targetType {
			deleted[marker.Key] = true
		}
	}
	return deleted
}

// Stats retourne des statistiques sur le vault.
func (v *Vault) Stats() map[string]int {
	stats := make(map[string]int)
	for _, b := range v.chain.Blocks {
		switch b.Type {
		case BlockGenesis:
			stats["genesis"]++
		case BlockBookmark:
			stats["favoris"]++
		case BlockPassword:
			stats["mots_de_passe"]++
		case BlockHistory:
			stats["historique"]++
		case BlockTabState:
			stats["états_onglets"]++
		case BlockDelete:
			stats["suppressions"]++
		}
	}
	stats["total"] = v.chain.Len()
	return stats
}
