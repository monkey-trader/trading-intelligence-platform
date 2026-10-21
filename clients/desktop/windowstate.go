package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Untergrenzen entsprechen MinWidth/MinHeight in main.go; kleinere Werte sind kaputt.
const (
	minWindowWidth  = 960
	minWindowHeight = 600
)

type windowState struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	X      int `json:"x"`
	Y      int `json:"y"`
}

// windowStatePath liegt neben der Datenbank unter <UserConfigDir>/tip/.
func windowStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tip", "window.json"), nil
}

func loadWindowState(path string) (windowState, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return windowState{}, false
	}
	var state windowState
	if err := json.Unmarshal(raw, &state); err != nil {
		return windowState{}, false
	}
	if state.Width < minWindowWidth || state.Height < minWindowHeight {
		return windowState{}, false
	}
	return state, true
}

func saveWindowState(path string, state windowState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}
