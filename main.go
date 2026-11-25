package main

import (
	"runtime"
)

const defaultWindowTitle = "Arbeitsplatz festlegen"

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
	current, err := getUserEnv(cfg.Variable)
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
	runWindow(cfg, current, settingsPath, hidden, settingsErr)
}
