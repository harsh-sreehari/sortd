package mover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoverCollision(t *testing.T) {
	m := New()
	tmpDir, err := os.MkdirTemp("", "sortd-mover")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dest := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(dest, []byte("exist"), 0644)

	unique := m.GenerateUniquePath(dest)
	if unique != filepath.Join(tmpDir, "test_1.txt") {
		t.Errorf("expected test_1.txt, got %s", unique)
	}
}

func TestMoverMove(t *testing.T) {
	m := New()
	tmpDir, err := os.MkdirTemp("", "sortd-mover-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	src := filepath.Join(tmpDir, "src.txt")
	os.WriteFile(src, []byte("data"), 0644)

	dest := filepath.Join(tmpDir, "folder", "dest.txt")
	final, err := m.Move(src, dest)
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}

	if final != dest {
		t.Errorf("expected %s, got %s", dest, final)
	}

	if _, err := os.Stat(final); os.IsNotExist(err) {
		t.Error("final file does not exist")
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("src file still exists after move")
	}
}
