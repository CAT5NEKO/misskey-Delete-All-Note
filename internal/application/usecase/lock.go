package usecase

import (
	"fmt"
	"os"
)

func acquireLock(path string, force bool) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		fmt.Fprintf(f, "%d", os.Getpid())
		f.Close()
		return func() { os.Remove(path) }, nil
	}
	if os.IsExist(err) {
		if force {
			if removeErr := os.Remove(path); removeErr != nil {
				return nil, fmt.Errorf("failed to remove stale lock file: %w", removeErr)
			}
			f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
			if err != nil {
				return nil, fmt.Errorf("failed to acquire lock after force: %w", err)
			}
			fmt.Fprintf(f, "%d", os.Getpid())
			f.Close()
			return func() { os.Remove(path) }, nil
		}
		return nil, fmt.Errorf("another instance is already running (use --force to override)")
	}
	return nil, err
}
