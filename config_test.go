package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigFromProjectFile(t *testing.T) {
	cfg, err := loadConfig(filepath.Join(".", "set-env.config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "Arbeitsplatz festlegen" {
		t.Fatalf("title = %q", cfg.Title)
	}
	if cfg.Variable != "IGELNAME" {
		t.Fatalf("variable = %q", cfg.Variable)
	}
	if cfg.Columns != 5 {
		t.Fatalf("columns = %d", cfg.Columns)
	}
	if len(cfg.Groups) != 2 {
		t.Fatalf("groups = %d", len(cfg.Groups))
	}
	if cfg.Groups[0].Name != "Labor" || cfg.Groups[0].Buttons[0].Anzeigename != "Labb2platz" || cfg.Groups[0].Buttons[0].Wert != "Igel1" {
		t.Fatalf("labor = %+v", cfg.Groups[0])
	}
	if cfg.Groups[1].Name != "Büro" || len(cfg.Groups[1].Buttons) == 0 {
		t.Fatalf("buero = %+v", cfg.Groups[1])
	}
	if cfg.Process.Kind != processRestart || len(cfg.Process.Executables) != 2 || cfg.Process.Executables[0] != "cts32.exe" || cfg.Process.Executables[1] != "labor32.exe" || !cfg.Process.StartIfMissing {
		t.Fatalf("process = %+v", cfg.Process)
	}
	if cfg.Process.CommandLine != `C:\Program Files (x86)\Inlog\Synchro\Lance32.exe` {
		t.Fatalf("command = %q", cfg.Process.CommandLine)
	}
	if cfg.Process.WorkingDirectory != `C:\Program Files (x86)\Inlog\Synchro` {
		t.Fatalf("workdir = %q", cfg.Process.WorkingDirectory)
	}
	if cfg.Process.Inhibit != "appli32.exe" {
		t.Fatalf("inhibit = %q", cfg.Process.Inhibit)
	}
	if cfg.ClearLabel != "Notebook/Workstation" {
		t.Fatalf("clear = %q", cfg.ClearLabel)
	}
}

func TestLoadConfigAcceptsAliasesAndRejectsBadVariable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "set-env.config.yaml")
	content := []byte("variablenname: ARBEITSPLATZ\nentries:\n  - name: Büro\n    value: BUERO\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Dark {
		t.Fatal("dark mode was not requested")
	}
	if cfg.Variable != "ARBEITSPLATZ" || len(cfg.Groups) != 1 || cfg.Groups[0].Name != "" || cfg.Groups[0].Buttons[0].Anzeigename != "Büro" || cfg.Groups[0].Buttons[0].Wert != "BUERO" {
		t.Fatalf("parsed %+v", cfg)
	}

	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("variable: IGEL NAME\nbuttons:\n  - anzeigename: A\n    wert: B\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(bad); err == nil {
		t.Fatal("expected invalid variable name")
	}
}

func TestDarkModeFromConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "set-env.config.yaml")
	content := []byte("darkmode: true\nbuttons:\n  - anzeigename: A\n    wert: \"1\"\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Dark {
		t.Fatal("expected dark mode")
	}
}

func TestUserSettingsPath(t *testing.T) {
	path, err := userSettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, filepath.Join("SET-ENV", "gruppen.yaml")) {
		t.Fatalf("path = %s", path)
	}
}

func TestUserSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gruppen.yaml")
	hidden, err := loadUserSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden) != 0 {
		t.Fatalf("missing file = %+v", hidden)
	}
	if err := saveUserSettings(path, map[string]bool{"Büro": true, "Labor": false}); err != nil {
		t.Fatal(err)
	}
	hidden, err = loadUserSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if !hidden["Büro"] || hidden["Labor"] {
		t.Fatalf("stored %+v", hidden)
	}
	if err := saveUserSettings(path, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	hidden, err = loadUserSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden) != 0 {
		t.Fatalf("cleared %+v", hidden)
	}
}

func TestColumnsFromConfig(t *testing.T) {
	dir := t.TempDir()
	omitted := filepath.Join(dir, "omitted.yaml")
	if err := os.WriteFile(omitted, []byte("buttons:\n  - anzeigename: A\n    wert: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(omitted)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Columns != defaultColumns {
		t.Fatalf("default columns = %d", cfg.Columns)
	}

	three := filepath.Join(dir, "three.yaml")
	if err := os.WriteFile(three, []byte("spalten: 3\nbuttons:\n  - anzeigename: A\n    wert: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = loadConfig(three)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Columns != 3 {
		t.Fatalf("columns = %d", cfg.Columns)
	}
	if buttonRows(7, cfg.Columns) != 3 {
		t.Fatalf("rows = %d", buttonRows(7, cfg.Columns))
	}
}

func TestClearLabelFromConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "set-env.config.yaml")
	content := []byte("buttons:\n  - anzeigename: A\n    wert: \"1\"\nfatclient: Mobil\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClearLabel != "Mobil" {
		t.Fatalf("clear = %q", cfg.ClearLabel)
	}

	if err := os.WriteFile(path, []byte("buttons:\n  - anzeigename: A\n    wert: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClearLabel != defaultClearLabel {
		t.Fatalf("default = %q", cfg.ClearLabel)
	}
}

func TestProcessActionFromConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "set-env.config.yaml")
	content := []byte("buttons:\n  - anzeigename: A\n    wert: \"1\"\napplikation:\n  aktion: restart\n  name: 'C:\\Apps\\Client.EXE'\n  startbefehl: '\"C:\\Apps\\Client.EXE\" /neu'\n  starten: true\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Process.Kind != processRestart || len(cfg.Process.Executables) != 1 || cfg.Process.Executables[0] != "Client.EXE" || !cfg.Process.StartIfMissing {
		t.Fatalf("process = %+v", cfg.Process)
	}
	if cfg.Process.CommandLine != `"C:\Apps\Client.EXE" /neu` {
		t.Fatalf("command = %q", cfg.Process.CommandLine)
	}
	if cfg.Process.WorkingDirectory != "" {
		t.Fatalf("workdir = %q", cfg.Process.WorkingDirectory)
	}

	withDir := []byte("buttons:\n  - anzeigename: A\n    wert: \"1\"\napplikation:\n  aktion: restart\n  name: Client.exe\n  startbefehl: Client.exe\n  arbeitsverzeichnis: 'C:\\Apps\\Client'\n")
	if err := os.WriteFile(path, withDir, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Process.WorkingDirectory != `C:\Apps\Client` {
		t.Fatalf("workdir = %q", cfg.Process.WorkingDirectory)
	}
	if err := os.WriteFile(path, []byte("buttons:\n  - anzeigename: A\n    wert: \"1\"\napplikation:\n  aktion: restart\n  name: Client.exe\n  startbefehl: Client.exe\n  arbeitsverzeichnis: Apps\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil {
		t.Fatal("expected relative working directory to be rejected")
	}

	listed := []byte("buttons:\n  - anzeigename: A\n    wert: \"1\"\napplikation:\n  aktion: kill\n  name:\n    - One.exe\n    - 'C:\\Tools\\Two.EXE'\n")
	if err := os.WriteFile(path, listed, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Process.Executables) != 2 || cfg.Process.Executables[0] != "One.exe" || cfg.Process.Executables[1] != "Two.EXE" {
		t.Fatalf("list = %+v", cfg.Process.Executables)
	}

	killOnly := []byte("buttons:\n  - anzeigename: A\n    wert: \"1\"\napplikation:\n  action: kill\n  executable: tool.exe\n")
	if err := os.WriteFile(path, killOnly, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Process.Kind != processKill || len(cfg.Process.Executables) != 1 || cfg.Process.Executables[0] != "tool.exe" || cfg.Process.StartIfMissing {
		t.Fatalf("kill = %+v", cfg.Process)
	}

	for _, bad := range []string{
		"buttons:\n  - anzeigename: A\n    wert: \"1\"\napplikation:\n  aktion: stop\n  name: a.exe\n",
		"buttons:\n  - anzeigename: A\n    wert: \"1\"\napplikation:\n  aktion: kill\n",
		"buttons:\n  - anzeigename: A\n    wert: \"1\"\napplikation:\n  aktion: restart\n  name: a.exe\n",
		"buttons:\n  - anzeigename: A\n    wert: \"1\"\napplikation:\n  aktion: kill\n  name:\n    - One.exe\n    - one.EXE\n",
	} {
		if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := loadConfig(path); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestButtonGridSize(t *testing.T) {
	oneRow := preferredWindowSize(3, 5)
	twoRows := preferredWindowSize(6, 5)
	if oneRow.Width != twoRows.Width {
		t.Fatalf("width changed from %v to %v", oneRow.Width, twoRows.Width)
	}
	if twoRows.Height <= oneRow.Height {
		t.Fatalf("second row did not grow the window: %v vs %v", oneRow.Height, twoRows.Height)
	}
	width, height, _, _ := buttonMetrics(oneRow.Width, 5)
	if height*2 < width-1 || height*2 > width+1 {
		t.Fatalf("button ratio = %v x %v", width, height)
	}
}
