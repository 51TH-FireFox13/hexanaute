package foxota

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Apply remplace le binaire courant par le nouveau.
// Sur Windows, on ne peut pas remplacer un exe en cours d'exécution,
// donc on écrit à côté et on crée un script de remplacement.
func Apply(newBinary []byte) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("foxota: impossible de trouver le binaire courant: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("foxota: impossible de résoudre le chemin: %w", err)
	}

	dir := filepath.Dir(execPath)
	name := filepath.Base(execPath)

	if runtime.GOOS == "windows" {
		return applyWindows(dir, name, execPath, newBinary)
	}
	return applyUnix(execPath, newBinary)
}

func applyUnix(execPath string, newBinary []byte) error {
	// Sur Unix : écrire le nouveau binaire à côté, puis renommer (atomique)
	tmpPath := execPath + ".new"
	oldPath := execPath + ".old"

	// Écrire le nouveau binaire
	if err := os.WriteFile(tmpPath, newBinary, 0755); err != nil {
		return fmt.Errorf("foxota: échec écriture: %w", err)
	}

	// Renommer l'ancien
	if err := os.Rename(execPath, oldPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("foxota: échec backup: %w", err)
	}

	// Mettre le nouveau en place
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Rollback
		os.Rename(oldPath, execPath)
		return fmt.Errorf("foxota: échec remplacement: %w", err)
	}

	// Supprimer l'ancien
	os.Remove(oldPath)

	return nil
}

func applyWindows(dir, name, execPath string, newBinary []byte) error {
	// Sur Windows, on ne peut pas remplacer un .exe en cours d'exécution.
	// Stratégie : écrire le nouveau à côté + créer un script batch qui :
	// 1. Attend que le processus courant se termine
	// 2. Remplace l'exe
	// 3. Relance le navigateur

	newPath := filepath.Join(dir, name+".new")
	batPath := filepath.Join(dir, "fox-update.bat")

	// Écrire le nouveau binaire
	if err := os.WriteFile(newPath, newBinary, 0755); err != nil {
		return fmt.Errorf("foxota: échec écriture: %w", err)
	}

	// Créer le script de mise à jour
	bat := fmt.Sprintf(`@echo off
echo [FoxOTA] Mise à jour en cours...
timeout /t 2 /nobreak >nul
del "%s"
rename "%s" "%s"
echo [FoxOTA] Mise à jour terminée !
start "" "%s"
del "%%~f0"
`, execPath, newPath, name, execPath)

	if err := os.WriteFile(batPath, []byte(bat), 0755); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("foxota: échec script: %w", err)
	}

	fmt.Println("[FoxOTA] Mise à jour prête. Redémarrez HexaNaute pour appliquer.")
	return nil
}

// Rollback restaure l'ancien binaire si disponible.
func Rollback() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execPath, _ = filepath.EvalSymlinks(execPath)
	oldPath := execPath + ".old"

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("foxota: pas de backup disponible pour le rollback")
	}

	if err := os.Rename(execPath, execPath+".failed"); err != nil {
		return err
	}

	if err := os.Rename(oldPath, execPath); err != nil {
		os.Rename(execPath+".failed", execPath)
		return err
	}

	os.Remove(execPath + ".failed")
	return nil
}
