package mover

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Mover struct{}

func New() *Mover {
	return &Mover{}
}

func (m *Mover) GenerateUniquePath(dest string) string {
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}

	dir := filepath.Dir(dest)
	base := filepath.Base(dest)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	counter := 1
	for {
		newPath := filepath.Join(dir, fmt.Sprintf("%s_%d%s", name, counter, ext))
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

func (m *Mover) Move(src, dest string) (string, error) {
	// If dest is a directory or ends in a slash, append the file name
	if info, err := os.Stat(dest); (err == nil && info.IsDir()) || strings.HasSuffix(dest, "/") {
		dest = filepath.Join(dest, filepath.Base(src))
	}

	finalPath := m.GenerateUniquePath(dest)

	destDir := filepath.Dir(finalPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir failed: %w", err)
	}

	err := os.Rename(src, finalPath)
	if err != nil {
		if isCrossDeviceError(err) {
			return m.copyDelete(src, finalPath)
		}
		return "", err
	}

	return finalPath, nil
}

func isCrossDeviceError(err error) bool {
	return strings.Contains(err.Error(), "invalid cross-device link")
}

func (m *Mover) copyDelete(src, dest string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return "", err
	}
	
	out.Close()
	in.Close()

	if err := os.Remove(src); err != nil {
		return dest, fmt.Errorf("copied but failed to remove original: %w", err)
	}

	return dest, nil
}

func (m *Mover) Park(src, rootFolder string) (string, error) {
	dest := filepath.Join(rootFolder, ".unsorted", filepath.Base(src))
	return m.Move(src, dest)
}
