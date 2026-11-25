package main

import "testing"
import "unsafe"

func TestWin32StructLayout(t *testing.T) {
	if unsafe.Offsetof(drawItem{}.HwndItem) != 24 || unsafe.Sizeof(drawItem{}) != 64 {
		t.Fatalf("drawItem hwnd=%d size=%d", unsafe.Offsetof(drawItem{}.HwndItem), unsafe.Sizeof(drawItem{}))
	}
	if unsafe.Offsetof(winMsg{}.WParam) != 16 || unsafe.Sizeof(winMsg{}) != 48 {
		t.Fatalf("winMsg wparam=%d size=%d", unsafe.Offsetof(winMsg{}.WParam), unsafe.Sizeof(winMsg{}))
	}
	if unsafe.Sizeof(trackMouse{}) != 24 || unsafe.Sizeof(wndClassEx{}) != 80 {
		t.Fatalf("track=%d class=%d", unsafe.Sizeof(trackMouse{}), unsafe.Sizeof(wndClassEx{}))
	}
}
