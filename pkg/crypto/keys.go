// Package crypto fournit les primitives cryptographiques de Fox Browser.
// XChaCha20-Poly1305 pour le chiffrement, Ed25519 pour les signatures,
// Argon2id pour la dérivation de clé depuis une passphrase.
package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

var ErrDecrypt = errors.New("crypto: échec du déchiffrement")

// GenerateKeyPair génère une paire de clés Ed25519.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// DeriveKey dérive une clé 32 octets depuis une passphrase via Argon2id.
func DeriveKey(passphrase []byte, salt []byte) []byte {
	return argon2.IDKey(passphrase, salt, 3, 64*1024, 4, 32)
}

// Encrypt chiffre des données avec XChaCha20-Poly1305.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt déchiffre des données chiffrées avec XChaCha20-Poly1305.
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrDecrypt
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, ErrDecrypt
	}

	return plaintext, nil
}
