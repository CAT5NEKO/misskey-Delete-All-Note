package usecase

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func acquireLock(path string, force bool) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		fmt.Fprintf(f, "%d\n", os.Getpid())
		f.Close()
		return func() { os.Remove(path) }, nil
	}
	if !os.IsExist(err) {
		return nil, err
	}

	if force || isLockStale(path) {
		if removeErr := os.Remove(path); removeErr != nil {
			return nil, fmt.Errorf("failed to remove stale lock file: %w", removeErr)
		}
		f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire lock after removing stale: %w", err)
		}
		fmt.Fprintf(f, "%d\n", os.Getpid())
		f.Close()
		return func() { os.Remove(path) }, nil
	}

	return nil, fmt.Errorf("another instance is already running (use --force to override)")
}

func isLockStale(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return true
	}
	if pid <= 0 {
		return true
	}
	if !processExists(pid) {
		return true
	}
	return false
}

func processExists(pid int) bool {
	if runtime.GOOS == "windows" {
		return true
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	defer p.Release()
	return p.Signal(syscall.Signal(0)) == nil
}
