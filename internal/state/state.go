package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const statePath = "zyr/state.json"

type GitPathResolver interface {
	GitPath(name string) (string, error)
}

type GitState struct {
	resolver GitPathResolver
}

type fileState struct {
	Version         int  `json:"version"`
	AddAllConfirmed bool `json:"addAllConfirmed"`
}

func NewGitState(resolver GitPathResolver) *GitState {
	return &GitState{resolver: resolver}
}

func (s *GitState) AddAllConfirmed() (bool, error) {
	path, err := s.resolver.GitPath(statePath)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var saved fileState
	if err := json.Unmarshal(data, &saved); err != nil {
		return false, fmt.Errorf("estado local do Zyr inválido: %w", err)
	}
	return saved.AddAllConfirmed, nil
}

func (s *GitState) ConfirmAddAll() error {
	path, err := s.resolver.GitPath(statePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(fileState{Version: 1, AddAllConfirmed: true}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}
