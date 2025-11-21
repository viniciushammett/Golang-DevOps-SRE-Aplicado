//go:build windows
// +build windows

package main

import (
	"syscall"
	"unsafe"
)

// Implementação de getDiskUsage para Windows usando GetDiskFreeSpaceEx.
func getDiskUsage(path string) (total, used, freeUser uint64, percentUsed float64, err error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes int64

	r1, _, e1 := syscall.Syscall6(
		procGetDiskFreeSpaceEx.Addr(),
		4,
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
		0, 0,
	)

	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
		return 0, 0, 0, 0, err
	}

	total = uint64(totalNumberOfBytes)
	freeTotal := uint64(totalNumberOfFreeBytes)
	freeUser = uint64(freeBytesAvailable)
	used = total - freeTotal

	if total == 0 {
		return total, used, freeUser, 0, nil
	}
	percentUsed = (float64(used) / float64(total)) * 100
	return total, used, freeUser, percentUsed, nil
}

var (
	modkernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceEx = modkernel32.NewProc("GetDiskFreeSpaceExW")
)
