package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrAlreadyRunning = errors.New("desktop instance is already running")

type InstanceLock struct {
	path string
	file *os.File
}

func AcquireInstanceLock(path string) (*InstanceLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrAlreadyRunning
		}
		return nil, err
	}

	return &InstanceLock{
		path: path,
		file: file,
	}, nil
}

func (l *InstanceLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}

	if err := l.file.Close(); err != nil {
		return err
	}

	err := os.Remove(l.path)
	l.file = nil
	return err
}
