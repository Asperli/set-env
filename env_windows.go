package main

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func getUserEnv(name string) (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	value, _, err := key.GetStringValue(name)
	if err == registry.ErrNotExist {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
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
