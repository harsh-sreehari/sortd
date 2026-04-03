package mover

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
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
	// Resolve absolute paths for comparison
	srcAbs, _ := filepath.Abs(src)
	destAbs, _ := filepath.Abs(dest)

	// If dest is a directory or ends in a slash, append the file name
	if info, err := os.Stat(destAbs); (err == nil && info.IsDir()) || strings.HasSuffix(dest, "/") {
		destAbs = filepath.Join(destAbs, filepath.Base(srcAbs))
	}

	if srcAbs == destAbs {
		return srcAbs, nil
	}

	finalPath := m.GenerateUniquePath(destAbs)

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

func (m *Mover) WriteXattr(path string, tags []string) {
	if len(tags) == 0 {
		return
	}

	val := strings.Join(tags, ",")
	// user. namespace is required for user-defined attributes on Linux
	err := unix.Setxattr(path, "user.sortd.tags", []byte(val), 0)
	if err != nil {
		// Log and continue; xattr support is optional and shouldn't break the flow
		fmt.Printf("⚠️  Failed to write xattr to %s: %v\n", path, err)
	}
}
