// Package foxota implémente le système de mise à jour OTA sécurisé.
// Les mises à jour sont signées Ed25519 et vérifiées contre plusieurs
// sources indépendantes avant application.
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
)

// Update représente une mise à jour disponible.
type Update struct {
	Version    uint64 // numéro monotone croissant (anti-downgrade)
	Channel    string // "stable", "beta", "security"
	BinaryHash [32]byte
	Binary     []byte
	Signature  []byte // Ed25519 sur BinaryHash
}

// SourceVerification représente le hash publié par une source indépendante.
type SourceVerification struct {
	SourceName string
	Hash       [32]byte
	Trusted    bool
}

// Verifier vérifie les mises à jour avant application.
type Verifier struct {
	pubKey         ed25519.PublicKey // clé publique du développeur, en dur dans le binaire
	currentVersion uint64
	minSources     int // nombre minimum de sources pour consensus
}

// NewVerifier crée un vérificateur OTA.
func NewVerifier(pubKey ed25519.PublicKey, currentVersion uint64) *Verifier {
	return &Verifier{
		pubKey:         pubKey,
		currentVersion: currentVersion,
		minSources:     3,
	}
}

// Verify vérifie une mise à jour contre la signature et le consensus multi-source.
func (v *Verifier) Verify(update *Update, sources []SourceVerification) error {
	// 1. Anti-downgrade
	if update.Version <= v.currentVersion {
		return fmt.Errorf("%w: v%d <= v%d", ErrDowngrade, update.Version, v.currentVersion)
	}

	// 2. Vérifier le hash du binaire
	actualHash := sha256.Sum256(update.Binary)
	if actualHash != update.BinaryHash {
		return ErrHashMismatch
	}

	// 3. Vérifier la signature Ed25519
	if !ed25519.Verify(v.pubKey, update.BinaryHash[:], update.Signature) {
		return ErrInvalidSignature
	}

	// 4. Consensus multi-source
	matching := 0
	for _, src := range sources {
		if src.Hash == update.BinaryHash && src.Trusted {
			matching++
		}
	}
	if matching < v.minSources {
		return fmt.Errorf("%w: %d/%d sources confirment", ErrNoConsensus, matching, v.minSources)
	}

	return nil
}
