//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

func init() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")

	// Forcer la code page UTF-8
	setConsoleOutputCP := kernel32.NewProc("SetConsoleOutputCP")
	setConsoleCP := kernel32.NewProc("SetConsoleCP")
	setConsoleOutputCP.Call(65001)
	setConsoleCP.Call(65001)

	// Activer les séquences ANSI (couleurs) sur Windows 10+
	getStdHandle := kernel32.NewProc("GetStdHandle")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")

	handle, _, _ := getStdHandle.Call(uintptr(0xFFFFFFF5)) // STD_OUTPUT_HANDLE
	var mode uint32
	getConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode)))
	mode |= 0x0004 // ENABLE_VIRTUAL_TERMINAL_PROCESSING
	setConsoleMode.Call(handle, uintptr(mode))
}
