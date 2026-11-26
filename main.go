package main

import (
	"runtime"
)

const (
	defaultWindowTitle = "Arbeitsplatz festlegen"
	defaultClearLabel  = "Notebook/Workstation"
)

func main() {
	runtime.LockOSThread()

	path, err := configPath()
	if err != nil {
		showStartupError(defaultWindowTitle, err)
		return
	}
	cfg, err := loadConfig(path)
	if err != nil {
		showStartupError(defaultWindowTitle, err)
		return
	}
	current, envExists, err := readUserEnv(cfg.Variable)
	if err != nil {
		showStartupError(cfg.Title, err)
		return
	}
	settingsPath, err := userSettingsPath()
	if err != nil {
		showStartupError(cfg.Title, err)
		return
	}
	hidden, settingsErr := loadUserSettings(settingsPath)
	runWindow(cfg, current, envExists, settingsPath, hidden, settingsErr)
}
