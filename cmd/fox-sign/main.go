// fox-sign — Outil de signature des releases Fox Browser.
//
// Usage :
//   fox-sign genkey                      Générer une paire de clés
//   fox-sign release <version> <channel> Créer un manifest signé
//   fox-sign verify <manifest.json>      Vérifier un manifest
//
// La clé privée ne doit JAMAIS être sur une machine connectée en prod.
// Idéalement sur une clé USB chiffrée ou un HSM.
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	foxcrypto "github.com/51TH-FireFox13/fox-browser/pkg/crypto"
	"github.com/51TH-FireFox13/fox-browser/internal/foxota"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "genkey":
		cmdGenKey()
	case "release":
		cmdRelease()
	case "verify":
		cmdVerify()
	case "hash":
		cmdHash()
	default:
		usage()
	}
}

func usage() {
	fmt.Println(`fox-sign — Outil de signature des releases Fox Browser

Usage :
  fox-sign genkey                          Générer une paire de clés Ed25519
  fox-sign release <ver_num> <ver_str> <channel>  Créer un manifest signé
  fox-sign verify <manifest.json>          Vérifier un manifest
  fox-sign hash <fichier>                  Calculer le SHA-256 d'un fichier

Exemples :
  fox-sign genkey
  fox-sign release 4 0.0.4 stable
  fox-sign verify manifest.json
  fox-sign hash fox.exe`)
}

func cmdGenKey() {
	pubKey, privKey, err := foxcrypto.GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur : %s\n", err)
		os.Exit(1)
	}

	pubHex := hex.EncodeToString(pubKey)
	privHex := hex.EncodeToString(privKey)

	// Sauvegarder
	os.WriteFile("fox-signing.pub", []byte(pubHex+"\n"), 0644)
	os.WriteFile("fox-signing.key", []byte(privHex+"\n"), 0600)

	fmt.Println("Clés générées :")
	fmt.Printf("  Publique  : %s\n", pubHex)
	fmt.Printf("  Fichiers  : fox-signing.pub (publique), fox-signing.key (PRIVÉE)\n")
	fmt.Println()
	fmt.Println("⚠  IMPORTANT : fox-signing.key est votre clé privée.")
	fmt.Println("   Ne la commitez JAMAIS. Stockez-la sur un support chiffré.")
	fmt.Println()
	fmt.Println("Intégrez la clé publique dans le binaire :")
	fmt.Printf("   const signingPubKey = \"%s\"\n", pubHex)
}

func cmdRelease() {
	if len(os.Args) < 5 {
		fmt.Println("Usage : fox-sign release <version_num> <version_str> <channel>")
		fmt.Println("Exemple : fox-sign release 4 0.0.4 stable")
		return
	}

	versionNum, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Version numérique invalide : %s\n", os.Args[2])
		os.Exit(1)
	}
	versionStr := os.Args[3]
	channel := os.Args[4]

	// Charger la clé privée
	privKey, err := loadPrivateKey("fox-signing.key")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur clé : %s\n", err)
		os.Exit(1)
	}

	// Demander le changelog
	fmt.Print("Changelog (une ligne) : ")
	var changelog string
	fmt.Scanln(&changelog)

	// Créer le manifest
	manifest := foxota.CreateRelease(versionNum, versionStr, channel, changelog)

	// Chercher les binaires dans le répertoire courant
	binaries := map[string]struct{ os, arch string }{
		"fox-linux-amd64":       {"linux", "amd64"},
		"fox-linux-arm64":       {"linux", "arm64"},
		"fox-windows-amd64.exe": {"windows", "amd64"},
		"fox-darwin-amd64":      {"darwin", "amd64"},
		"fox-darwin-arm64":      {"darwin", "arm64"},
	}

	found := 0
	for filename, platform := range binaries {
		data, err := os.ReadFile(filename)
		if err != nil {
			continue
		}
		manifest.AddBinary(platform.os, platform.arch, data, filename,
			fmt.Sprintf("https://releases.fox-browser.fr/v%s/%s", versionStr, filename))
		hash := sha256.Sum256(data)
		fmt.Printf("  ✓ %s (%d bytes, SHA-256: %s)\n", filename, len(data), hex.EncodeToString(hash[:8]))
		found++
	}

	if found == 0 {
		fmt.Println("Aucun binaire trouvé. Nommez-les fox-<os>-<arch>[.exe]")
		os.Exit(1)
	}

	// Signer
	if err := foxota.SignManifest(manifest, privKey); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur signature : %s\n", err)
		os.Exit(1)
	}

	// Sauvegarder
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur JSON : %s\n", err)
		os.Exit(1)
	}

	outFile := fmt.Sprintf("manifest-v%s.json", versionStr)
	os.WriteFile(outFile, data, 0644)

	fmt.Printf("\n✓ Manifest signé : %s (%d binaire(s))\n", outFile, found)
	fmt.Println("  Publiez ce fichier sur vos sources de mise à jour.")
}

func cmdVerify() {
	if len(os.Args) < 3 {
		fmt.Println("Usage : fox-sign verify <manifest.json>")
		return
	}

	data, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur : %s\n", err)
		os.Exit(1)
	}

	var manifest foxota.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		fmt.Fprintf(os.Stderr, "JSON invalide : %s\n", err)
		os.Exit(1)
	}

	// Charger la clé publique
	pubKey, err := loadPublicKey("fox-signing.pub")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur clé : %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Manifest v%s (canal: %s)\n", manifest.VersionStr, manifest.Channel)
	fmt.Printf("  Version  : %d\n", manifest.Version)
	fmt.Printf("  Binaires : %d\n", len(manifest.Binaries))

	for _, b := range manifest.Binaries {
		fmt.Printf("  - %s/%s : %s (%d bytes)\n", b.OS, b.Arch, b.Filename, b.Size)
	}

	if foxota.VerifyManifest(&manifest, pubKey) {
		fmt.Println("\n✓ Signature VALIDE")
	} else {
		fmt.Println("\n✗ Signature INVALIDE — ce manifest a été altéré !")
		os.Exit(1)
	}
}

func cmdHash() {
	if len(os.Args) < 3 {
		fmt.Println("Usage : fox-sign hash <fichier>")
		return
	}

	data, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur : %s\n", err)
		os.Exit(1)
	}

	hash := sha256.Sum256(data)
	fmt.Printf("SHA-256: %s\n", hex.EncodeToString(hash[:]))
	fmt.Printf("Fichier: %s (%d bytes)\n", os.Args[2], len(data))
}

func loadPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fichier %s introuvable (lancez 'fox-sign genkey' d'abord)", path)
	}
	keyHex := string(data)
	// Nettoyer
	for len(keyHex) > 0 && (keyHex[len(keyHex)-1] == '\n' || keyHex[len(keyHex)-1] == '\r') {
		keyHex = keyHex[:len(keyHex)-1]
	}
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("clé invalide")
	}
	return ed25519.PrivateKey(keyBytes), nil
}

func loadPublicKey(path string) (ed25519.PublicKey, error) {
	paths := []string{path, filepath.Join(".", path)}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		keyHex := string(data)
		for len(keyHex) > 0 && (keyHex[len(keyHex)-1] == '\n' || keyHex[len(keyHex)-1] == '\r') {
			keyHex = keyHex[:len(keyHex)-1]
		}
		keyBytes, err := hex.DecodeString(keyHex)
		if err != nil {
			continue
		}
		return ed25519.PublicKey(keyBytes), nil
	}
	return nil, fmt.Errorf("clé publique introuvable (%s)", path)
}
