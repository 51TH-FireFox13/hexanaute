package main

import (
	"fmt"
	"os"
	"runtime"
)

const (
	name    = "Fox Browser"
	version = "0.0.1"
	codename = "Renard"
)

func main() {
	fmt.Printf("%s v%s (%s)\n", name, version, codename)
	fmt.Printf("OS: %s | Arch: %s | Go: %s\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	fmt.Printf("CPUs: %d\n", runtime.NumCPU())

	if len(os.Args) > 1 && os.Args[1] == "version" {
		return
	}

	fmt.Println("\n[Fox] Initialisation...")
	fmt.Println("[FoxChain] Journal chiffré : non initialisé")
	fmt.Println("[AI Guard] NPU : détection en cours...")
	fmt.Println("[FoxOTA] Version à jour")
	fmt.Println("\n[Fox] Phase 0 — Mode terminal. En développement.")
}
