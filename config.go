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

type processKind int

const (
	processNone processKind = iota
	processKill
	processRestart
)

type processAction struct {
	Kind             processKind
	Executables      []string
	CommandLine      string
	WorkingDirectory string
	StartIfMissing   bool
	Inhibit          string
}

type appConfig struct {
	Title      string
	Variable   string
	Columns    int
	Dark       bool
	Groups     []groupDef
	Process    processAction
	ClearLabel string
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
	Titel         string       `yaml:"titel"`
	Title         string       `yaml:"title"`
	Fenstertitel  string       `yaml:"fenstertitel"`
	Variable      string       `yaml:"variable"`
	Variablenname string       `yaml:"variablenname"`
	Variabel      string       `yaml:"variabel"`
	Spalten       *int         `yaml:"spalten"`
	Columns       *int         `yaml:"columns"`
	DarkMode      bool         `yaml:"darkmode"`
	Buttons       []rawButton  `yaml:"buttons"`
	Eintraege     []rawButton  `yaml:"eintraege"`
	EintraegeUml  []rawButton  `yaml:"einträge"`
	Entries       []rawButton  `yaml:"entries"`
	Gruppen       []rawGroup   `yaml:"gruppen"`
	Groups        []rawGroup   `yaml:"groups"`
	Applikation   rawAppAction `yaml:"applikation"`
	Application   rawAppAction `yaml:"application"`
	Fatclient     string       `yaml:"fatclient"`
	Clear         string       `yaml:"clear"`
}

type rawAppAction struct {
	Aktion             string   `yaml:"aktion"`
	Action             string   `yaml:"action"`
	Name               nameList `yaml:"name"`
	Programm           nameList `yaml:"programm"`
	Program            nameList `yaml:"program"`
	Executable         nameList `yaml:"executable"`
	Startbefehl        string   `yaml:"startbefehl"`
	Kommando           string   `yaml:"kommando"`
	Command            string   `yaml:"command"`
	CommandLine        string   `yaml:"commandline"`
	Arbeitsverzeichnis string   `yaml:"arbeitsverzeichnis"`
	WorkDir            string   `yaml:"workdir"`
	WorkingDirectory   string   `yaml:"workingdirectory"`
	Starten            *bool    `yaml:"starten"`
	StartIfMiss        *bool    `yaml:"startifmissing"`
	Inhibit            string   `yaml:"inhibit"`
	InhibitStart       string   `yaml:"inhibitstart"`
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
	process, err := processFromRaw(raw)
	if err != nil {
		return appConfig{}, err
	}
	clearLabel, err := clearLabelFromRaw(raw)
	if err != nil {
		return appConfig{}, err
	}
	return appConfig{Title: title, Variable: variable, Columns: columns, Dark: raw.DarkMode, Groups: groups, Process: process, ClearLabel: clearLabel}, nil
}

func clearLabelFromRaw(raw rawConfig) (string, error) {
	label := firstText(raw.Fatclient, raw.Clear)
	if label == "" {
		return defaultClearLabel, nil
	}
	if strings.ContainsAny(label, "\r\n\x00") {
		return "", fmt.Errorf("der Text zum Löschen darf keine Zeilenumbrüche enthalten")
	}
	if len(label) > 120 {
		return "", fmt.Errorf("der Text zum Löschen ist länger als 120 Zeichen")
	}
	return label, nil
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

func processFromRaw(raw rawConfig) (processAction, error) {
	chosen := raw.Applikation
	if rawAppActionEmpty(chosen) {
		chosen = raw.Application
	}
	if rawAppActionEmpty(chosen) {
		return processAction{}, nil
	}

	kind, err := parseProcessKind(firstText(chosen.Aktion, chosen.Action))
	if err != nil {
		return processAction{}, err
	}
	startIfMissing := false
	if chosen.Starten != nil {
		startIfMissing = *chosen.Starten
	} else if chosen.StartIfMiss != nil {
		startIfMissing = *chosen.StartIfMiss
	}
	if kind == processNone && !startIfMissing {
		return processAction{}, nil
	}

	var executables []string
	if kind == processKill || kind == processRestart {
		executables, err = executableNames(firstNameList(chosen.Name, chosen.Programm, chosen.Program, chosen.Executable))
		if err != nil {
			return processAction{}, err
		}
	}
	command := firstText(chosen.Startbefehl, chosen.Kommando, chosen.Command, chosen.CommandLine)
	if kind == processRestart || startIfMissing {
		if command == "" {
			return processAction{}, fmt.Errorf("für einen Start fehlt der Startbefehl")
		}
		if strings.ContainsAny(command, "\r\n\x00") {
			return processAction{}, fmt.Errorf("der Startbefehl darf keine Zeilenumbrüche enthalten")
		}
		if len(command) > 32767 {
			return processAction{}, fmt.Errorf("der Startbefehl ist länger als 32767 Zeichen")
		}
	}
	workDir := ""
	if kind == processRestart || startIfMissing {
		var err error
		workDir, err = workingDirectory(firstText(chosen.Arbeitsverzeichnis, chosen.WorkDir, chosen.WorkingDirectory))
		if err != nil {
			return processAction{}, err
		}
	}
	inhibit := ""
	if text := firstText(chosen.Inhibit, chosen.InhibitStart); text != "" {
		name, err := executableName(text)
		if err != nil {
			return processAction{}, err
		}
		inhibit = name
	}
	return processAction{Kind: kind, Executables: executables, CommandLine: command, WorkingDirectory: workDir, StartIfMissing: startIfMissing, Inhibit: inhibit}, nil
}

func workingDirectory(value string) (string, error) {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" {
		return "", nil
	}
	if strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("das Arbeitsverzeichnis darf keine Zeilenumbrüche enthalten")
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("das Arbeitsverzeichnis %q muss ein vollständiger Pfad sein", value)
	}
	return filepath.Clean(value), nil
}

func rawAppActionEmpty(raw rawAppAction) bool {
	return len(firstNameList(raw.Name, raw.Programm, raw.Program, raw.Executable)) == 0 && firstText(raw.Aktion, raw.Action, raw.Startbefehl, raw.Kommando, raw.Command, raw.CommandLine, raw.Arbeitsverzeichnis, raw.WorkDir, raw.WorkingDirectory, raw.Inhibit, raw.InhibitStart) == "" && raw.Starten == nil && raw.StartIfMiss == nil
}

type nameList []string

func (n *nameList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var text string
		if err := value.Decode(&text); err != nil {
			return err
		}
		if strings.TrimSpace(text) != "" {
			*n = []string{text}
		}
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		*n = list
	default:
		return fmt.Errorf("der Programmname muss ein Text oder eine Liste sein")
	}
	return nil
}

func firstNameList(lists ...nameList) nameList {
	for _, list := range lists {
		if len(list) > 0 {
			return list
		}
	}
	return nil
}

func executableNames(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("der Programmname fehlt")
	}
	names := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		name, err := executableName(value)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(name)
		if seen[key] {
			return nil, fmt.Errorf("der Programmname %q ist mehrfach vorhanden", name)
		}
		seen[key] = true
		names = append(names, name)
	}
	return names, nil
}

func parseProcessKind(value string) (processKind, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none":
		return processNone, nil
	case "kill":
		return processKill, nil
	case "restart":
		return processRestart, nil
	default:
		return processNone, fmt.Errorf("aktion %q ist ungültig. Erlaubt sind none, kill und restart", value)
	}
}

func executableName(value string) (string, error) {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" {
		return "", fmt.Errorf("der Programmname fehlt")
	}
	name := filepath.Base(value)
	if name == "." || name == ".." || strings.ContainsAny(name, `\/:*?"<>|`+"\r\n\x00") {
		return "", fmt.Errorf("der Programmname %q ist ungültig", value)
	}
	return name, nil
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
