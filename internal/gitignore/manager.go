package gitignore

import (
	"errors"
	"os"
	"path/filepath"
)

type CurrentDirectoryManager struct {
	path string
}

func NewCurrentDirectoryManager() *CurrentDirectoryManager {
	return &CurrentDirectoryManager{path: ".gitignore"}
}

func NewManagerAt(directory string) *CurrentDirectoryManager {
	return &CurrentDirectoryManager{path: filepath.Join(directory, ".gitignore")}
}

func (m *CurrentDirectoryManager) Ensure() (bool, error) {
	_, err := os.Stat(m.path)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	file, err := os.OpenFile(m.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return false, err
	}
	if _, err = file.WriteString(Template); err != nil {
		_ = file.Close()
		return false, err
	}
	if err = file.Close(); err != nil {
		return false, err
	}
	return true, nil
}
