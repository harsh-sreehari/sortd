package peek

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func TextPeek(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	buf := make([]byte, 2048)
	n, _ := f.Read(buf)
	if n == 0 {
		return ""
	}

	content := string(buf[:n])
	if !utf8.ValidString(content) {
		// Quick fix for binary data
		return ""
	}

	return content
}

func PdfPeek(path string) string {
	// Call pdftotext -l 1 -q
	cmd := exec.Command("pdftotext", "-l", "1", "-q", path, "-")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	content := strings.TrimSpace(string(out))
	if len(content) > 1000 {
		return content[:1000]
	}
	return content
}

func PeekDispatcher(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".pdf":
		return PdfPeek(path)
	case ".txt", ".md", ".rst", ".tex", ".go", ".py":
		return TextPeek(path)
	default:
		return ""
	}
}
