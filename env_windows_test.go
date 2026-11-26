package main

import "testing"

func TestSetUserEnvRoundTrip(t *testing.T) {
	const name = "BSZ_ENV_PROBE"
	before, err := getUserEnv(name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		keyErr := restoreUserEnv(name, before)
		if keyErr != nil {
			t.Error(keyErr)
		}
	})

	if err := setUserEnv(name, "PROBE-1"); err != nil {
		t.Fatal(err)
	}
	stored, err := getUserEnv(name)
	if err != nil {
		t.Fatal(err)
	}
	if stored != "PROBE-1" {
		t.Fatalf("stored %q", stored)
	}
	if err := clearUserEnv(name); err != nil {
		t.Fatal(err)
	}
	_, exists, err := readUserEnv(name)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("variable still exists")
	}
}

func restoreUserEnv(name, previous string) error {
	if previous == "" {
		return deleteUserEnv(name)
	}
	return setUserEnv(name, previous)
}
