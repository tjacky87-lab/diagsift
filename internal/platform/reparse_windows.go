//go:build windows

package platform

import "syscall"

const fileAttributeReparsePoint = 0x400

func isReparsePoint(path string) bool {
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return true
	}
	attributes, err := syscall.GetFileAttributes(pointer)
	if err != nil {
		return false
	}
	return attributes&fileAttributeReparsePoint != 0
}
