package main

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func getUserEnv(name string) (string, error) {
	value, _, err := readUserEnv(name)
	return value, err
}

func readUserEnv(name string) (string, bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return "", false, err
	}
	defer key.Close()

	value, _, err := key.GetStringValue(name)
	if err == registry.ErrNotExist {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func setUserEnv(name, value string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("die Benutzervariable %s konnte nicht geöffnet werden: %w", name, err)
	}
	defer key.Close()

	if strings.Contains(value, "%") {
		err = key.SetExpandStringValue(name, value)
	} else {
		err = key.SetStringValue(name, value)
	}
	if err != nil {
		return fmt.Errorf("die Benutzervariable %s konnte nicht gespeichert werden: %w", name, err)
	}

	stored, readErr := getUserEnv(name)
	if readErr != nil {
		return readErr
	}
	if stored != value {
		return fmt.Errorf("die Variable wurde nicht gespeichert. Erwartet %q, gelesen %q", value, stored)
	}

	broadcastEnvironmentChange()
	return nil
}

func clearUserEnv(name string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("die Benutzervariable %s konnte nicht geöffnet werden: %w", name, err)
	}
	defer key.Close()
	err = key.DeleteValue(name)
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("die Benutzervariable %s konnte nicht gelöscht werden: %w", name, err)
	}
	_, exists, readErr := readUserEnv(name)
	if readErr != nil {
		return readErr
	}
	if exists {
		return fmt.Errorf("die Variable %s ist noch vorhanden", name)
	}
	broadcastEnvironmentChange()
	return nil
}

func deleteUserEnv(name string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	err = key.DeleteValue(name)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

func broadcastEnvironmentChange() {
	environment, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}

	user32 := windows.NewLazySystemDLL("user32.dll")
	sendMessageTimeout := user32.NewProc("SendMessageTimeoutW")
	var result uintptr
	_, _, _ = sendMessageTimeout.Call(
		0xFFFF,
		0x001A,
		0,
		uintptr(unsafe.Pointer(environment)),
		0x0002,
		5000,
		uintptr(unsafe.Pointer(&result)),
	)
}
