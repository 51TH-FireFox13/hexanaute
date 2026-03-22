package foxchain

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	foxcrypto "github.com/51TH-FireFox13/fox-browser/pkg/crypto"
)

// ProfileMeta contient les métadonnées du profil (non chiffrées).
type ProfileMeta struct {
	Version   int    `json:"version"`
	Salt      []byte `json:"salt"`       // sel pour Argon2id
	PublicKey []byte `json:"public_key"` // clé publique Ed25519
	// La clé privée est chiffrée avec la clé dérivée de la passphrase
	EncryptedPrivateKey []byte `json:"encrypted_private_key"`
}

// Profile gère un profil utilisateur FoxChain.
type Profile struct {
	Dir   string
	Meta  ProfileMeta
	vault *Vault
}

// CreateProfile crée un nouveau profil avec une passphrase.
func CreateProfile(dir string, passphrase string) (*Profile, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("foxchain: impossible de créer %s: %w", dir, err)
	}

	// Générer le sel pour Argon2id
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// Dériver la clé de chiffrement
	encKey := foxcrypto.DeriveKey([]byte(passphrase), salt)

	// Générer les clés Ed25519
	pubKey, privKey, err := foxcrypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	// Chiffrer la clé privée
	encPrivKey, err := foxcrypto.Encrypt(encKey, privKey)
	if err != nil {
		return nil, err
	}

	meta := ProfileMeta{
		Version:             1,
		Salt:                salt,
		PublicKey:            pubKey,
		EncryptedPrivateKey: encPrivKey,
	}

	// Sauvegarder les métadonnées
	metaPath := filepath.Join(dir, "profile.json")
	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(metaPath, metaJSON, 0600); err != nil {
		return nil, err
	}

	// Créer la chaîne
	chain := NewChain(pubKey, privKey, encKey)
	chainPath := filepath.Join(dir, "foxchain.dat")
	if err := chain.Save(chainPath); err != nil {
		return nil, err
	}

	vault := NewVault(chain)

	return &Profile{
		Dir:   dir,
		Meta:  meta,
		vault: vault,
	}, nil
}

// OpenProfile ouvre un profil existant avec la passphrase.
func OpenProfile(dir string, passphrase string) (*Profile, error) {
	metaPath := filepath.Join(dir, "profile.json")
	metaJSON, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("foxchain: profil introuvable dans %s", dir)
	}

	var meta ProfileMeta
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		return nil, fmt.Errorf("foxchain: profil corrompu: %w", err)
	}

	// Dériver la clé de chiffrement
	encKey := foxcrypto.DeriveKey([]byte(passphrase), meta.Salt)

	// Déchiffrer la clé privée
	privKey, err := foxcrypto.Decrypt(encKey, meta.EncryptedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("foxchain: passphrase incorrecte")
	}

	pubKey := ed25519.PublicKey(meta.PublicKey)

	// Charger la chaîne
	chainPath := filepath.Join(dir, "foxchain.dat")
	blocks, err := LoadBlocks(chainPath)
	if err != nil {
		return nil, fmt.Errorf("foxchain: erreur lecture chaîne: %w", err)
	}

	chain := LoadChain(blocks, pubKey, privKey, encKey)

	// Vérifier l'intégrité
	if !chain.Verify() {
		return nil, fmt.Errorf("foxchain: intégrité de la chaîne compromise !")
	}

	vault := NewVault(chain)

	return &Profile{
		Dir:   dir,
		Meta:  meta,
		vault: vault,
	}, nil
}

// Vault retourne le vault du profil.
func (p *Profile) Vault() *Vault {
	return p.vault
}

// Save sauvegarde l'état actuel de la chaîne.
func (p *Profile) Save() error {
	chainPath := filepath.Join(p.Dir, "foxchain.dat")
	return p.vault.Chain().Save(chainPath)
}

// Exists vérifie si un profil existe dans un répertoire.
func Exists(dir string) bool {
	metaPath := filepath.Join(dir, "profile.json")
	_, err := os.Stat(metaPath)
	return err == nil
}
