package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type userSettings struct {
	Ausgeblendet []string `yaml:"ausgeblendet"`
}

func userSettingsPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("der AppData-Ordner konnte nicht ermittelt werden: %w", err)
	}
	return filepath.Join(base, "SET-ENV", "gruppen.yaml"), nil
}

func loadUserSettings(path string) (map[string]bool, error) {
	data, err := readFirstSettingsFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, fmt.Errorf("die Benutzereinstellungen konnten nicht gelesen werden:\n%s", err)
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]bool{}, nil
	}

	var stored userSettings
	if err := yaml.Unmarshal(data, &stored); err != nil {
		return nil, fmt.Errorf("die Benutzereinstellungen sind kein gültiges YAML:\n%s", err)
	}
	hidden := make(map[string]bool, len(stored.Ausgeblendet))
	for _, name := range stored.Ausgeblendet {
		if name != "" {
			hidden[name] = true
		}
	}
	return hidden, nil
}

func readFirstSettingsFile(path string) ([]byte, error) {
	var lastErr error
	for _, candidate := range settingsCandidates(path) {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return data, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = os.ErrNotExist
	}
	return nil, lastErr
}

func settingsCandidates(path string) []string {
	candidates := []string{path}
	if strings.HasSuffix(path, ".yaml") {
		candidates = append(candidates, strings.TrimSuffix(path, ".yaml")+".json")
	}
	legacy := strings.Replace(path, `\SET-ENV\`, `\BSZ-ENV\`, 1)
	if legacy == path {
		return candidates
	}
	candidates = append(candidates, legacy)
	if strings.HasSuffix(legacy, ".yaml") {
		candidates = append(candidates, strings.TrimSuffix(legacy, ".yaml")+".json")
	}
	return candidates
}

func saveUserSettings(path string, hidden map[string]bool) error {
	names := make([]string, 0, len(hidden))
	for name, isHidden := range hidden {
		if isHidden && name != "" {
			names = append(names, name)
		}
	}
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(userSettings{Ausgeblendet: names}); err != nil {
		return err
	}
	_ = encoder.Close()
	data := buf.Bytes()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("der Ordner für die Benutzereinstellungen konnte nicht angelegt werden:\n%s", err)
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return fmt.Errorf("die Benutzereinstellungen konnten nicht gespeichert werden:\n%s", err)
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(path)
		if renameErr := os.Rename(temp, path); renameErr != nil {
			_ = os.Remove(temp)
			return fmt.Errorf("die Benutzereinstellungen konnten nicht gespeichert werden:\n%s", renameErr)
		}
	}
	return nil
}
