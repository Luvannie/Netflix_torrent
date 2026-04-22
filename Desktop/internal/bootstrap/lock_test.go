package bootstrap

import (
	"path/filepath"
	"testing"
)

func TestAcquireInstanceLockRejectsSecondOwner(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "netflixtorrent.lock")

	first, err := AcquireInstanceLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireInstanceLock() error = %v", err)
	}
	defer func() {
		if err := first.Release(); err != nil {
			t.Fatalf("Release() error = %v", err)
		}
	}()

	second, err := AcquireInstanceLock(lockPath)
	if err != ErrAlreadyRunning {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}
	if second != nil {
		t.Fatalf("expected nil second lock")
	}
}

func TestAcquireInstanceLockCanBeReacquiredAfterRelease(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "netflixtorrent.lock")

	lock, err := AcquireInstanceLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireInstanceLock() error = %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	reacquired, err := AcquireInstanceLock(lockPath)
	if err != nil {
		t.Fatalf("reacquire error = %v", err)
	}
	if err := reacquired.Release(); err != nil {
		t.Fatalf("second release error = %v", err)
	}
}
