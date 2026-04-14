# HexaNaute

**Navigateur souverain, sécurisé par IA locale, écrit en Go.**

## Vision

HexaNaute est un navigateur web indépendant qui ne repose sur aucun moteur contrôlé par les GAFAM (pas de Blink, pas de WebKit, pas de Gecko). Il intègre une couche de sécurité par intelligence artificielle exécutée localement sur NPU.

## Principes

- **Souveraineté** — Aucune dépendance technologique étrangère, licence EUPL
- **Sécurité IA** — Analyse comportementale du JavaScript via modèle local sur NPU
- **Vie privée** — Données utilisateur dans un journal chaîné chiffré (HexaChain)
- **Portabilité** — Navigateur stateless, identité portable sur clé/token hardware
- **Multiplateforme** — Go compilé : Linux, Windows, macOS, ARM, RISC-V

## Architecture

```
cmd/hexanaute/       Point d'entrée CLI + interface graphique (Fyne)
internal/
  engine/            Moteur de rendu HTML → blocs GUI (GUIBlock)
  jsengine/          Sandbox JavaScript (QuickJS via CGo) + anti-fingerprinting
  network/           Stack réseau (HTTP/2, TLS 1.3, fetch d'images)
  aiguard/           IA Guard — analyse de menaces sur NPU (ONNX)
  foxchain/          Journal chaîné chiffré (favoris, mdp, état)
  foxota/            Mise à jour OTA signée (Ed25519, multi-source)
pkg/
  crypto/            Primitives cryptographiques (XChaCha20, Ed25519, Argon2id)
  protocol/          Protocole OTA custom
packaging/           Script NSIS (installeur Windows)
assets/              Icônes et ressources
```

## Build

```bash
go build -o hexanaute ./cmd/hexanaute
```

## Cross-compilation

```bash
GOOS=windows GOARCH=amd64 go build -o hexanaute.exe ./cmd/hexanaute
GOOS=darwin GOARCH=arm64 go build -o hexanaute-mac ./cmd/hexanaute
GOOS=linux GOARCH=arm64 go build -o hexanaute-arm ./cmd/hexanaute
```

## Installeur Windows

Le script NSIS se trouve dans `packaging/installer.nsi`.  
Générer l'installeur (nécessite NSIS installé) :

```bash
makensis packaging/installer.nsi
```

## Statut

**v0.5.0** — Rendu HTML interactif : texte, liens, formulaires, images.  
Interface graphique Fyne avec moteur de rendu propriétaire (sans WebKit/Blink).

## Licence

[EUPL v1.2](LICENSE) — Licence libre européenne.
