//go:build !windows
// +build !windows

package main

import "syscall"

// Implementação de getDiskUsage para sistemas Unix-like.
func getDiskUsage(path string) (total, used, freeUser uint64, percentUsed float64, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return 0, 0, 0, 0, err
	}

	blockSize := uint64(st.Bsize)
	total = st.Blocks * blockSize

	freeUser = st.Bavail * blockSize

	freeTotal := st.Bfree * blockSize
	used = total - freeTotal

	if total == 0 {
		return total, used, freeUser, 0, nil
	}
	percentUsed = (float64(used) / float64(total)) * 100
	return total, used, freeUser, percentUsed, nil
}
