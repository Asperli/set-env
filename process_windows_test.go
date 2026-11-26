package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestProcessActionNoneAndMissingProcess(t *testing.T) {
	if err := applyProcessAction(appConfig{}, "Igel1"); err != nil {
		t.Fatal(err)
	}
	cfg := appConfig{Process: processAction{Kind: processKill, Executable: "set-env-missing-probe.exe"}}
	if err := applyProcessAction(cfg, "Igel1"); err != nil {
		t.Fatal(err)
	}
}

func TestKillProbeProcess(t *testing.T) {
	probe, cleanup := startProbe(t, "/c", "ping", "-n", "30", "127.0.0.1")
	defer cleanup()

	cfg := appConfig{Process: processAction{Kind: processKill, Executable: filepath.Base(probe)}}
	if err := applyProcessAction(cfg, "Igel1"); err != nil {
		t.Fatal(err)
	}
	if running, err := processIDsByName(filepath.Base(probe)); err != nil {
		t.Fatal(err)
	} else if len(running) != 0 {
		t.Fatalf("still running: %v", running)
	}
}

func TestRestartPassesVariableWhenNotRunning(t *testing.T) {
	dir := t.TempDir()
	probe := copyProbeExecutable(t, dir)
	out := filepath.Join(dir, "wert.txt")
	command := fmt.Sprintf("%s /c echo %%SET_ENV_PROBE_VAR%% > %s", quoteCommand(probe), quoteCommand(out))
	cfg := appConfig{
		Variable: "SET_ENV_PROBE_VAR",
		Process: processAction{
			Kind:           processRestart,
			Executable:     filepath.Base(probe),
			CommandLine:    command,
			StartIfMissing: true,
		},
	}
	if err := applyProcessAction(cfg, "Igel9"); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	var text string
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(out)
		if err == nil {
			text = strings.TrimSpace(string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})))
			if text != "" {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if text != "Igel9" {
		t.Fatalf("written value = %q", text)
	}
}

func TestRestartSkipsWhenNotRunning(t *testing.T) {
	dir := t.TempDir()
	probe := copyProbeExecutable(t, dir)
	marker := filepath.Join(dir, "gestartet.txt")
	cfg := appConfig{
		Variable: "SET_ENV_PROBE_VAR",
		Process: processAction{
			Kind:        processRestart,
			Executable:  filepath.Base(probe),
			CommandLine: fmt.Sprintf("%s /c echo nein> %s", quoteCommand(probe), quoteCommand(marker)),
		},
	}
	if err := applyProcessAction(cfg, "Igel1"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("command started although the program was not running")
	}
}

func TestFirstCommandToken(t *testing.T) {
	if got := firstCommandToken(`"C:\Program Files\App\App.exe" /reset`); got != `C:\Program Files\App\App.exe` {
		t.Fatalf("quoted = %q", got)
	}
	if got := firstCommandToken(`C:\App\App.exe /reset`); got != `C:\App\App.exe` {
		t.Fatalf("plain = %q", got)
	}
}

func startProbe(t *testing.T, args ...string) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	probe := copyProbeExecutable(t, dir)
	cmd := exec.Command(probe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ids, err := processIDsByName(filepath.Base(probe))
		if err != nil {
			_ = cmd.Process.Kill()
			t.Fatal(err)
		}
		if len(ids) > 0 {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	return probe, func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }
}

func copyProbeExecutable(t *testing.T, dir string) string {
	t.Helper()
	src := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "set-env-probe.exe")
	if err := os.WriteFile(dst, data, 0o700); err != nil {
		t.Fatal(err)
	}
	return dst
}

func quoteCommand(path string) string {
	return `"` + strings.ReplaceAll(path, `"`, "") + `"`
}
