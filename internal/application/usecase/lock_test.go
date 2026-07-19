package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func tempLockPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "misskeyNotedel-test.lock")
}

func TestAcquireLock_Success(t *testing.T) {
	path := tempLockPath(t)

	cleanup, err := acquireLock(path, false)
	if err != nil {
		t.Fatalf("acquireLock failed: %v", err)
	}

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatal("lock file should exist after acquisition")
	}

	cleanup()

	if _, statErr := os.Stat(path); statErr == nil {
		t.Fatal("lock file should be removed after cleanup")
	}
}

func TestAcquireLock_DoubleLockFails(t *testing.T) {
	path := tempLockPath(t)

	cleanup, err := acquireLock(path, false)
	if err != nil {
		t.Fatalf("first acquireLock failed: %v", err)
	}
	defer cleanup()

	_, err = acquireLock(path, false)
	if err == nil {
		t.Fatal("second acquireLock should fail without force")
	}
}

func TestAcquireLock_ForceOverrides(t *testing.T) {
	path := tempLockPath(t)

	cleanup1, err := acquireLock(path, false)
	if err != nil {
		t.Fatalf("first acquireLock failed: %v", err)
	}
	cleanup1()

	cleanup2, err := acquireLock(path, false)
	if err != nil {
		t.Fatalf("re-acquire after cleanup failed: %v", err)
	}
	defer cleanup2()

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatal("lock file should exist after re-acquisition")
	}
}

func TestAcquireLock_ForceRemovesStale(t *testing.T) {
	path := tempLockPath(t)

	f, createErr := os.Create(path)
	if createErr != nil {
		t.Fatalf("failed to create stale lock: %v", createErr)
	}
	f.Close()

	cleanup, err := acquireLock(path, true)
	if err != nil {
		t.Fatalf("acquireLock with force failed: %v", err)
	}
	defer cleanup()

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatal("lock file should exist after force acquisition")
	}
}

func TestAcquireLock_StaleLockAutoCleanup(t *testing.T) {
	path := tempLockPath(t)

	if err := os.WriteFile(path, []byte("99999\n"), 0600); err != nil {
		t.Fatalf("failed to write stale lock: %v", err)
	}

	cleanup, err := acquireLock(path, false)
	if err != nil {
		t.Fatalf("acquireLock should auto-clean stale lock: %v", err)
	}
	defer cleanup()

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatal("lock file should exist after stale cleanup")
	}
}

func TestAcquireLock_InvalidStaleLock(t *testing.T) {
	path := tempLockPath(t)

	if err := os.WriteFile(path, []byte("not-a-pid"), 0600); err != nil {
		t.Fatalf("failed to write invalid stale lock: %v", err)
	}

	cleanup, err := acquireLock(path, false)
	if err != nil {
		t.Fatalf("acquireLock should clean invalid lock: %v", err)
	}
	defer cleanup()
}

func TestAcquireLock_OwnPidLock(t *testing.T) {
	path := tempLockPath(t)

	if err := os.WriteFile(path, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0600); err != nil {
		t.Fatalf("failed to write own pid lock: %v", err)
	}

	_, err := acquireLock(path, false)
	if err == nil {
		t.Fatal("should reject own pid lock without force")
	}
}
