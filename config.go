package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var variableNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type buttonDef struct {
	Anzeigename string
	Wert        string
}

type groupDef struct {
	Name    string
	Buttons []buttonDef
}

type appConfig struct {
	Title    string
	Variable string
	Columns  int
	Dark     bool
	Groups   []groupDef
}

func (c appConfig) hasNamedGroups() bool {
	for _, group := range c.Groups {
		if group.Name != "" {
			return true
		}
	}
	return false
}

type rawConfig struct {
	Titel         string      `yaml:"titel"`
	Title         string      `yaml:"title"`
	Fenstertitel  string      `yaml:"fenstertitel"`
	Variable      string      `yaml:"variable"`
	Variablenname string      `yaml:"variablenname"`
	Variabel      string      `yaml:"variabel"`
	Spalten       *int        `yaml:"spalten"`
	Columns       *int        `yaml:"columns"`
	DarkMode      bool        `yaml:"darkmode"`
	Buttons       []rawButton `yaml:"buttons"`
	Eintraege     []rawButton `yaml:"eintraege"`
	EintraegeUml  []rawButton `yaml:"einträge"`
	Entries       []rawButton `yaml:"entries"`
	Gruppen       []rawGroup  `yaml:"gruppen"`
	Groups        []rawGroup  `yaml:"groups"`
}

type rawGroup struct {
	Name         string      `yaml:"name"`
	Anzeigename  string      `yaml:"anzeigename"`
	Buttons      []rawButton `yaml:"buttons"`
	Eintraege    []rawButton `yaml:"eintraege"`
	EintraegeUml []rawButton `yaml:"einträge"`
	Entries      []rawButton `yaml:"entries"`
}

type rawButton struct {
	Anzeigename string `yaml:"anzeigename"`
	Name        string `yaml:"name"`
	Label       string `yaml:"label"`
	Text        string `yaml:"text"`
	Wert        string `yaml:"wert"`
	Value       string `yaml:"value"`
}

func configPath() (string, error) {
	dirs := make([]string, 0, 2)
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("der Ordner der Konfiguration konnte nicht ermittelt werden")
	}

	names := []string{"set-env.config.yaml", "set-env.config.yml"}
	for _, dir := range dirs {
		for _, name := range names {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}
	return filepath.Join(dirs[0], "set-env.config.yaml"), nil
}

func loadConfig(path string) (appConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return appConfig{}, fmt.Errorf("die Konfigurationsdatei wurde nicht gefunden:\n%s", path)
		}
		return appConfig{}, fmt.Errorf("die Konfigurationsdatei konnte nicht gelesen werden:\n%s", err)
	}

	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if len(bytes.TrimSpace(data)) == 0 {
		return appConfig{}, fmt.Errorf("die Konfigurationsdatei ist leer:\n%s", path)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return appConfig{}, fmt.Errorf("die Konfigurationsdatei ist kein gültiges YAML:\n%s", err)
	}

	title, err := titleFromRaw(raw)
	if err != nil {
		return appConfig{}, err
	}

	variable := firstText(raw.Variable, raw.Variablenname, raw.Variabel)
	if variable == "" {
		variable = "IGELNAME"
	}
	if !variableNamePattern.MatchString(variable) {
		return appConfig{}, fmt.Errorf("der Variablenname %q ist ungültig. Erlaubt sind Buchstaben, Ziffern und Unterstriche. Er muss mit einem Buchstaben oder Unterstrich beginnen", variable)
	}

	columns, err := columnsFromRaw(raw)
	if err != nil {
		return appConfig{}, err
	}
	groups, err := groupsFromRaw(raw)
	if err != nil {
		return appConfig{}, err
	}
	return appConfig{Title: title, Variable: variable, Columns: columns, Dark: raw.DarkMode, Groups: groups}, nil
}

func titleFromRaw(raw rawConfig) (string, error) {
	title := firstText(raw.Titel, raw.Title, raw.Fenstertitel)
	if title == "" {
		return defaultWindowTitle, nil
	}
	if strings.ContainsAny(title, "\r\n\x00") {
		return "", fmt.Errorf("der Fenstertitel darf keine Zeilenumbrüche enthalten")
	}
	if len(title) > 120 {
		return "", fmt.Errorf("der Fenstertitel ist länger als 120 Zeichen")
	}
	return title, nil
}

func columnsFromRaw(raw rawConfig) (int, error) {
	chosen := raw.Spalten
	if chosen == nil {
		chosen = raw.Columns
	}
	if chosen == nil {
		return defaultColumns, nil
	}
	if *chosen < 1 || *chosen > 20 {
		return 0, fmt.Errorf("spalten muss zwischen 1 und 20 liegen, gelesen %d", *chosen)
	}
	return *chosen, nil
}

func groupsFromRaw(raw rawConfig) ([]groupDef, error) {
	source := raw.Gruppen
	if len(source) == 0 {
		source = raw.Groups
	}
	if len(source) > 0 {
		return parseGroups(source)
	}

	buttons, err := parseButtons("", firstButtonList(raw.Buttons, raw.Eintraege, raw.EintraegeUml, raw.Entries))
	if err != nil {
		return nil, err
	}
	if len(buttons) == 0 {
		return nil, fmt.Errorf("in der Konfiguration fehlt die Liste 'gruppen' oder 'buttons'")
	}
	return []groupDef{{Buttons: buttons}}, nil
}

func parseGroups(source []rawGroup) ([]groupDef, error) {
	groups := make([]groupDef, 0, len(source))
	seen := map[string]bool{}
	for i, item := range source {
		name := firstText(item.Name, item.Anzeigename)
		if name == "" {
			return nil, fmt.Errorf("gruppe %d hat keinen Namen", i+1)
		}
		if seen[name] {
			return nil, fmt.Errorf("der Gruppenname %q ist mehrfach vorhanden", name)
		}
		seen[name] = true
		buttons, err := parseButtons(name, firstButtonList(item.Buttons, item.Eintraege, item.EintraegeUml, item.Entries))
		if err != nil {
			return nil, err
		}
		if len(buttons) == 0 {
			return nil, fmt.Errorf("gruppe %q enthält keine Arbeitsplätze", name)
		}
		groups = append(groups, groupDef{Name: name, Buttons: buttons})
	}
	return groups, nil
}

func firstButtonList(lists ...[]rawButton) []rawButton {
	for _, list := range lists {
		if len(list) > 0 {
			return list
		}
	}
	return nil
}

func parseButtons(group string, source []rawButton) ([]buttonDef, error) {
	buttons := make([]buttonDef, 0, len(source))
	for i, item := range source {
		name := firstText(item.Anzeigename, item.Name, item.Label, item.Text)
		value := firstText(item.Wert, item.Value)
		where := fmt.Sprintf("Button %d", i+1)
		if group != "" {
			where = fmt.Sprintf("Button %d in Gruppe %q", i+1, group)
		}
		if name == "" {
			return nil, fmt.Errorf("%s hat keinen Anzeigenamen", where)
		}
		if value == "" {
			return nil, fmt.Errorf("button %q hat keinen Wert", name)
		}
		if strings.ContainsAny(value, "\r\n\x00") {
			return nil, fmt.Errorf("der Wert von %q darf keine Zeilenumbrüche enthalten", name)
		}
		if len(value) > 32767 {
			return nil, fmt.Errorf("der Wert von %q ist länger als 32767 Zeichen", name)
		}
		buttons = append(buttons, buttonDef{Anzeigename: name, Wert: value})
	}
	return buttons, nil
}

func firstText(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
