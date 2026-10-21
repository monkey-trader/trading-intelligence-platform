package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowState_SpeichernUndLaden(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tip", "window.json")
	want := windowState{Width: 1200, Height: 800, X: 40, Y: 60}

	if err := saveWindowState(path, want); err != nil {
		t.Fatal(err)
	}
	got, ok := loadWindowState(path)

	if !ok || got != want {
		t.Fatalf("got %+v ok=%v, erwartet %+v", got, ok, want)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("Rechte = %v, erwartet 0600", info.Mode().Perm())
	}
}

func TestWindowState_FehlendeOderKaputteDateiLiefertNichts(t *testing.T) {
	dir := t.TempDir()
	if _, ok := loadWindowState(filepath.Join(dir, "fehlt.json")); ok {
		t.Fatal("fehlende Datei darf nichts liefern")
	}
	broken := filepath.Join(dir, "kaputt.json")
	_ = os.WriteFile(broken, []byte("{nope"), 0o600)
	if _, ok := loadWindowState(broken); ok {
		t.Fatal("kaputte Datei darf nichts liefern")
	}
}

func TestWindowState_UnsinnigeGroesseWirdVerworfen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "window.json")
	_ = saveWindowState(path, windowState{Width: 10, Height: 10})
	if _, ok := loadWindowState(path); ok {
		t.Fatal("zu kleines Fenster muss verworfen werden (MinWidth/MinHeight)")
	}
}
