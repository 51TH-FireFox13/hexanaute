package foxota

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// Manifest est le fichier de description d'une release, publié sur les sources.
type Manifest struct {
	Version    uint64          `json:"version"`
	VersionStr string          `json:"version_str"`
	Channel    string          `json:"channel"`
	Timestamp  int64           `json:"timestamp"`
	Changelog  string          `json:"changelog"`
	Binaries   []BinaryInfo    `json:"binaries"`
	Signature  string          `json:"signature"` // hex-encoded Ed25519 signature du JSON sans ce champ
}

// BinaryInfo décrit un binaire pour une plateforme.
type BinaryInfo struct {
	OS       string `json:"os"`       // "windows", "linux", "darwin"
	Arch     string `json:"arch"`     // "amd64", "arm64"
	Hash     string `json:"hash"`     // SHA-256 hex
	Size     int64  `json:"size"`
	Filename string `json:"filename"` // "fox-windows-amd64.exe"
	URL      string `json:"url"`      // URL de téléchargement
}

// CurrentPlatformBinary retourne le binaire pour la plateforme courante.
func (m *Manifest) CurrentPlatformBinary() (*BinaryInfo, error) {
	for _, b := range m.Binaries {
		if b.OS == runtime.GOOS && b.Arch == runtime.GOARCH {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("foxota: pas de binaire pour %s/%s", runtime.GOOS, runtime.GOARCH)
}

// SignManifest signe un manifest avec une clé privée Ed25519.
func SignManifest(m *Manifest, privKey ed25519.PrivateKey) error {
	m.Signature = "" // vider avant de calculer
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	sig := ed25519.Sign(privKey, hash[:])
	m.Signature = hex.EncodeToString(sig)
	return nil
}

// VerifyManifest vérifie la signature d'un manifest.
func VerifyManifest(m *Manifest, pubKey ed25519.PublicKey) bool {
	sigHex := m.Signature
	m.Signature = "" // vider pour recalculer le hash
	data, err := json.Marshal(m)
	if err != nil {
		return false
	}
	m.Signature = sigHex // restaurer

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	hash := sha256.Sum256(data)
	return ed25519.Verify(pubKey, hash[:], sig)
}

// CreateRelease crée un manifest pour une nouvelle release.
func CreateRelease(version uint64, versionStr, channel, changelog string) *Manifest {
	return &Manifest{
		Version:    version,
		VersionStr: versionStr,
		Channel:    channel,
		Timestamp:  time.Now().Unix(),
		Changelog:  changelog,
		Binaries:   make([]BinaryInfo, 0),
	}
}

// AddBinary ajoute un binaire au manifest.
func (m *Manifest) AddBinary(os, arch string, binaryData []byte, filename, url string) {
	hash := sha256.Sum256(binaryData)
	m.Binaries = append(m.Binaries, BinaryInfo{
		OS:       os,
		Arch:     arch,
		Hash:     hex.EncodeToString(hash[:]),
		Size:     int64(len(binaryData)),
		Filename: filename,
		URL:      url,
	})
}
