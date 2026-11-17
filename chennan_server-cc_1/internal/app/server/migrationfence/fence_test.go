package migrationfence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActiveFollowsRegularMarker(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "fence.json")
	t.Setenv(markerPathEnv, marker)

	if Active() {
		t.Fatal("missing marker must leave the fence inactive")
	}
	if err := os.WriteFile(marker, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !Active() {
		t.Fatal("regular marker must activate the fence")
	}
}

func TestActiveFailsClosedForMalformedMarker(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "fence.json")
	t.Setenv(markerPathEnv, marker)

	if err := os.Symlink(filepath.Join(root, "missing"), marker); err != nil {
		t.Fatal(err)
	}
	if !Active() {
		t.Fatal("symlink marker must fail closed")
	}
}

func TestActiveFailsClosedForOversizeMarker(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "fence.json")
	t.Setenv(markerPathEnv, marker)

	if err := os.WriteFile(marker, make([]byte, 4097), 0o600); err != nil {
		t.Fatal(err)
	}
	if !Active() {
		t.Fatal("oversize marker must fail closed")
	}
}
