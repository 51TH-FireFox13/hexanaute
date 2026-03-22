// Package foxota implémente le système de mise à jour OTA sécurisé (FOXUP).
// Les mises à jour sont signées Ed25519 et vérifiées contre plusieurs
// sources indépendantes avant application. Anti-downgrade intégré.
package foxota

import (
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"
)

var (
	ErrInvalidSignature = errors.New("foxota: signature invalide")
	ErrHashMismatch     = errors.New("foxota: hash ne correspond pas")
	ErrNoConsensus      = errors.New("foxota: consensus multi-source non atteint")
	ErrDowngrade        = errors.New("foxota: tentative de downgrade détectée")
	ErrNoUpdate         = errors.New("foxota: pas de mise à jour disponible")
	ErrApplyFailed      = errors.New("foxota: échec de l'application")
)

// Update représente une mise à jour disponible.
type Update struct {
	Version    uint64 // numéro monotone croissant (anti-downgrade)
	VersionStr string // "0.0.4" pour affichage
	Channel    string // "stable", "beta", "security"
	BinaryHash [32]byte
	Binary     []byte
	Signature  []byte // Ed25519 sur BinaryHash
	Changelog  string
}

// SourceVerification représente le hash publié par une source indépendante.
type SourceVerification struct {
	SourceName string
	Hash       [32]byte
	Trusted    bool
	Error      error // si la source n'a pas pu être consultée
}

// Verifier vérifie les mises à jour avant application.
type Verifier struct {
	pubKey         ed25519.PublicKey
	currentVersion uint64
	minSources     int
}

// NewVerifier crée un vérificateur OTA.
func NewVerifier(pubKey ed25519.PublicKey, currentVersion uint64) *Verifier {
	return &Verifier{
		pubKey:         pubKey,
		currentVersion: currentVersion,
		minSources:     2, // minimum 2 sources concordantes
	}
}

// SetMinSources configure le nombre minimum de sources pour le consensus.
func (v *Verifier) SetMinSources(n int) {
	v.minSources = n
}

// VerifyResult contient le résultat détaillé de la vérification.
type VerifyResult struct {
	Valid          bool
	Error          error
	HashOK         bool
	SignatureOK    bool
	AntiDowngradeOK bool
	ConsensusOK    bool
	SourcesMatched int
	SourcesTotal   int
}

// Verify vérifie une mise à jour et retourne un résultat détaillé.
func (v *Verifier) Verify(update *Update, sources []SourceVerification) *VerifyResult {
	result := &VerifyResult{
		SourcesTotal: len(sources),
	}

	// 1. Anti-downgrade
	if update.Version <= v.currentVersion {
		result.Error = fmt.Errorf("%w: v%d <= v%d", ErrDowngrade, update.Version, v.currentVersion)
		return result
	}
	result.AntiDowngradeOK = true

	// 2. Vérifier le hash du binaire
	actualHash := sha256.Sum256(update.Binary)
	if actualHash != update.BinaryHash {
		result.Error = ErrHashMismatch
		return result
	}
	result.HashOK = true

	// 3. Vérifier la signature Ed25519
	if !ed25519.Verify(v.pubKey, update.BinaryHash[:], update.Signature) {
		result.Error = ErrInvalidSignature
		return result
	}
	result.SignatureOK = true

	// 4. Consensus multi-source
	matching := 0
	for _, src := range sources {
		if src.Error != nil {
			continue
		}
		if src.Hash == update.BinaryHash && src.Trusted {
			matching++
		}
	}
	result.SourcesMatched = matching

	if matching < v.minSources {
		result.Error = fmt.Errorf("%w: %d/%d sources confirment (minimum %d)",
			ErrNoConsensus, matching, len(sources), v.minSources)
		return result
	}
	result.ConsensusOK = true

	result.Valid = true
	return result
}
