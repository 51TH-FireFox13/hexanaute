// fox-manifest — Génère un manifest signé avec URLs personnalisées.
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/51TH-FireFox13/fox-browser/internal/foxota"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: fox-manifest <key-file> [base-url]")
		return
	}

	keyFile := os.Args[1]
	baseURL := "http://192.168.1.42:8090"
	if len(os.Args) > 2 {
		baseURL = os.Args[2]
	}

	// Charger la clé
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur clé: %s\n", err)
		os.Exit(1)
	}
	keyHex := string(keyData)
	for len(keyHex) > 0 && (keyHex[len(keyHex)-1] == '\n' || keyHex[len(keyHex)-1] == '\r') {
		keyHex = keyHex[:len(keyHex)-1]
	}
	keyBytes, _ := hex.DecodeString(keyHex)
	privKey := ed25519.PrivateKey(keyBytes)

	manifest := foxota.CreateRelease(5, "0.0.5", "stable",
		"AI Guard amélioré, FoxChain complet, interface GUI enrichie")

	// Binaires
	bins := []struct {
		file, goos, arch string
	}{
		{"fox-windows-amd64.exe", "windows", "amd64"},
		{"fox-linux-amd64", "linux", "amd64"},
	}

	for _, b := range bins {
		data, err := os.ReadFile(b.file)
		if err != nil {
			fmt.Printf("  ⚠ %s non trouvé, ignoré\n", b.file)
			continue
		}
		url := fmt.Sprintf("%s/%s", baseURL, b.file)
		manifest.AddBinary(b.goos, b.arch, data, b.file, url)
		fmt.Printf("  ✓ %s → %s\n", b.file, url)
	}

	foxota.SignManifest(manifest, privKey)

	data, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile("manifest.json", data, 0644)
	fmt.Println("\n✓ manifest.json créé et signé")
}
