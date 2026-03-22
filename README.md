# Fox Browser

**Navigateur souverain, sécurisé par IA locale, écrit en Go.**

## Vision

Fox Browser est un navigateur web indépendant qui ne repose sur aucun moteur contrôlé par les GAFAM (pas de Blink, pas de WebKit, pas de Gecko). Il intègre une couche de sécurité par intelligence artificielle exécutée localement sur NPU.

## Principes

- **Souveraineté** — Aucune dépendance technologique étrangère, licence EUPL
- **Sécurité IA** — Analyse comportementale du JavaScript via modèle local sur NPU
- **Vie privée** — Données utilisateur dans un journal chaîné chiffré (FoxChain)
- **Portabilité** — Navigateur stateless, identité portable sur clé/token hardware
- **Multiplateforme** — Go compilé : Linux, Windows, macOS, ARM, RISC-V

## Architecture

```
cmd/fox/             Point d'entrée CLI
internal/
  browser/           Orchestration des onglets et sessions
  engine/            Moteur de rendu HTML/CSS
  jsengine/          Sandbox JavaScript (QuickJS via CGo)
  aiguard/           IA Guard — analyse de menaces sur NPU (ONNX)
  foxchain/          Journal chaîné chiffré (favoris, mdp, état)
  foxota/            Mise à jour OTA signée (Ed25519, multi-source)
  network/           Stack réseau (HTTP/2, HTTP/3, TLS 1.3)
  ui/                Interface utilisateur (terminal puis GUI)
pkg/
  crypto/            Primitives cryptographiques (XChaCha20, Ed25519, Argon2id)
  protocol/          Protocole OTA custom (FOXUP)
```

## Build

```bash
go build -o fox ./cmd/fox
```

## Cross-compilation

```bash
GOOS=windows GOARCH=amd64 go build -o fox.exe ./cmd/fox
GOOS=darwin GOARCH=arm64 go build -o fox-mac ./cmd/fox
GOOS=linux GOARCH=arm64 go build -o fox-arm ./cmd/fox
```

## Statut

Phase 0 — En développement.

## Licence

[EUPL v1.2](LICENSE) — Licence libre européenne.
