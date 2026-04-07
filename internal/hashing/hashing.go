package hashing

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HashFile computes sha256(relativePath + "\0" + content).
func HashFile(root, relPath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(relPath))
	h.Write([]byte{0})
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// HashFiles returns (contentHash, fileHashes) for a sorted list of files.
func HashFiles(root string, files []string) (string, map[string]string, error) {
	fileHashes := make(map[string]string, len(files))
	for _, f := range files {
		h, err := HashFile(root, f)
		if err != nil {
			return "", nil, err
		}
		fileHashes[f] = h
	}

	// Content hash = sha256(concat of file hashes in sorted order)
	sorted := make([]string, len(files))
	copy(sorted, files)
	sort.Strings(sorted)

	combined := strings.Builder{}
	for _, f := range sorted {
		combined.WriteString(fileHashes[f])
	}

	h := sha256.Sum256([]byte(combined.String()))
	return fmt.Sprintf("%x", h[:]), fileHashes, nil
}

// MakeID generates a test ID: 18 hex chars (no separators).
// Format: hash6(title_or_id) + hash6(labels) + hash6(sourcePath)
func MakeID(title, explicitID string, labels map[string]string, sourcePath string) string {
	source := title
	if explicitID != "" {
		source = explicitID
	}
	first := hash6(source)

	labelStr := ""
	if len(labels) > 0 {
		keys := make([]string, 0, len(labels))
		for k := range labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = k + "=" + labels[k]
		}
		labelStr = strings.Join(parts, ",")
	}
	second := hash6(labelStr)

	third := hash6(sourcePath)

	return first + second + third
}

func hash6(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:3])
}
