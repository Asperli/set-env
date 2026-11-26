package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	processExitTimeout = 5000
	waitTimeout        = 0x00000102
)

func applyProcessAction(cfg appConfig, value string) error {
	action := cfg.Process
	if action.Kind == processNone && !action.StartIfMissing {
		return nil
	}

	var problems []string
	if action.Kind == processKill || action.Kind == processRestart {
		for _, name := range action.Executables {
			ids, err := processIDsByName(name)
			if err != nil {
				return fmt.Errorf("die laufenden Programme konnten nicht gelesen werden:\n%s", err)
			}
			for _, pid := range ids {
				if err := terminateProcess(pid); err != nil {
					problems = append(problems, fmt.Sprintf("%s (PID %d) konnte nicht beendet werden: %s", name, pid, err))
				}
			}
		}
	}

	shouldStart := false
	if action.StartIfMissing {
		blocked := false
		if action.Inhibit != "" {
			running, err := processIDsByName(action.Inhibit)
			if err != nil {
				return fmt.Errorf("die laufenden Programme konnten nicht gelesen werden:\n%s", err)
			}
			blocked = len(running) > 0
		}
		shouldStart = !blocked
	} else if action.Kind == processRestart {
		shouldStart = true
	}
	if shouldStart {
		if err := startCommand(action.CommandLine, action.WorkingDirectory, cfg.Variable, value); err != nil {
			problems = append(problems, err.Error())
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(problems, "\n"))
}

func processIDsByName(name string) ([]uint32, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, err
	}

	self := uint32(os.Getpid())
	var ids []uint32
	for {
		exe := windows.UTF16ToString(entry.ExeFile[:])
		if entry.ProcessID != 0 && entry.ProcessID != self && strings.EqualFold(exe, name) {
			ids = append(ids, entry.ProcessID)
		}
		err = windows.Process32Next(snapshot, &entry)
		if err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				break
			}
			return nil, err
		}
	}
	return ids, nil
}

func terminateProcess(pid uint32) error {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, pid)
	if err != nil {
		if isProcessGone(err) {
			return nil
		}
		return err
	}
	defer windows.CloseHandle(handle)

	if err := windows.TerminateProcess(handle, 1); err != nil {
		if isProcessGone(err) {
			return nil
		}
		return err
	}
	event, err := windows.WaitForSingleObject(handle, processExitTimeout)
	if err != nil {
		return err
	}
	if event != windows.WAIT_OBJECT_0 {
		if event == waitTimeout {
			return fmt.Errorf("der Prozess läuft nach dem Beenden noch")
		}
		return fmt.Errorf("das Warten auf das Prozessende ist fehlgeschlagen")
	}
	return nil
}

func isProcessGone(err error) bool {
	return errors.Is(err, windows.ERROR_INVALID_PARAMETER) || errors.Is(err, windows.ERROR_INVALID_HANDLE)
}

func startCommand(commandLine, workDir, variable, value string) error {
	command, err := windows.UTF16FromString(commandLine)
	if err != nil {
		return fmt.Errorf("der Startbefehl konnte nicht gelesen werden:\n%s", err)
	}
	env, err := environmentBlock(variable, value)
	if err != nil {
		return err
	}

	dir := workDir
	if dir == "" {
		dir = commandWorkingDirectory(commandLine)
	}
	var cwd []uint16
	var cwdPtr *uint16
	if dir != "" {
		info, statErr := os.Stat(dir)
		if statErr != nil || !info.IsDir() {
			return fmt.Errorf("das Arbeitsverzeichnis wurde nicht gefunden:\n%s", dir)
		}
		cwd, err = windows.UTF16FromString(dir)
		if err != nil {
			return fmt.Errorf("das Arbeitsverzeichnis konnte nicht gelesen werden:\n%s", err)
		}
		cwdPtr = &cwd[0]
	}

	var startup windows.StartupInfo
	startup.Cb = uint32(unsafe.Sizeof(startup))
	var info windows.ProcessInformation
	err = windows.CreateProcess(nil, &command[0], nil, nil, false, windows.CREATE_UNICODE_ENVIRONMENT, &env[0], cwdPtr, &startup, &info)
	if err != nil {
		return fmt.Errorf("die Applikation konnte nicht gestartet werden:\n%s", err)
	}
	_ = windows.CloseHandle(info.Process)
	_ = windows.CloseHandle(info.Thread)
	return nil
}

func environmentBlock(variable, value string) ([]uint16, error) {
	lines := make([]string, 0, len(os.Environ())+1)
	replaced := false
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(name, variable) {
			if value != "" {
				lines = append(lines, variable+"="+value)
				replaced = true
			}
			continue
		}
		lines = append(lines, entry)
	}
	if !replaced && value != "" {
		lines = append(lines, variable+"="+value)
	}

	buf := make([]uint16, 0, 4096)
	for _, line := range lines {
		encoded, err := windows.UTF16FromString(line)
		if err != nil {
			return nil, fmt.Errorf("die Umgebung für den Neustart konnte nicht erzeugt werden:\n%s", err)
		}
		buf = append(buf, encoded...)
	}
	buf = append(buf, 0)
	return buf, nil
}

func commandWorkingDirectory(commandLine string) string {
	token := firstCommandToken(commandLine)
	if token == "" || !strings.ContainsAny(token, `\/`) {
		return ""
	}
	dir := filepath.Dir(token)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ""
	}
	return dir
}

func firstCommandToken(command string) string {
	text := strings.TrimSpace(command)
	if text == "" {
		return ""
	}
	if text[0] == '"' {
		end := strings.Index(text[1:], `"`)
		if end < 0 {
			return strings.TrimSpace(text[1:])
		}
		return text[1 : 1+end]
	}
	if index := strings.IndexAny(text, " \t"); index >= 0 {
		return text[:index]
	}
	return text
}
