package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/testmd/testmd/internal/models"
)

// Load reads state from a YAML lock file. Returns empty state if file does not exist.
func Load(lockFile string) (*models.State, error) {
	data, err := os.ReadFile(lockFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}, nil
		}
		return nil, err
	}

	var st models.State
	if err := yaml.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	if st.Tests == nil {
		st.Tests = map[string]*models.TestRecord{}
	}
	return &st, nil
}

// Save writes state as deterministic YAML using atomic write (temp + rename).
// If state has no tests, the lock file is deleted.
func Save(lockFile string, st *models.State) error {
	// Clean up stale temp file from previous crash
	tmp := lockFile + ".tmp"
	os.Remove(tmp)

	if len(st.Tests) == 0 {
		err := os.Remove(lockFile)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	data := marshalState(st)

	if err := os.MkdirAll(filepath.Dir(lockFile), 0755); err != nil {
		return err
	}

	// Atomic write: temp file + rename
	if err := os.WriteFile(tmp, []byte(data), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, lockFile)
}

// marshalState produces deterministic YAML with controlled key order.
// Tests sorted by ID, labels by key, files by path. Block-style only.
func marshalState(st *models.State) string {
	var b strings.Builder
	b.WriteString("version: 1\n")
	b.WriteString("tests:\n")

	ids := make([]string, 0, len(st.Tests))
	for id := range st.Tests {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		rec := st.Tests[id]
		fmt.Fprintf(&b, "  %s:\n", id)

		// Scalar fields first (for compact diffs)
		fmt.Fprintf(&b, "    title: %s\n", yamlQuote(rec.Title))
		fmt.Fprintf(&b, "    source: %s\n", yamlQuote(rec.Source))
		fmt.Fprintf(&b, "    status: %s\n", rec.Status)
		fmt.Fprintf(&b, "    content_hash: %s\n", yamlQuote(rec.ContentHash))
		writeNullable(&b, "    ", "resolved_at", rec.ResolvedAt)
		writeNullable(&b, "    ", "failed_at", rec.FailedAt)
		writeNullable(&b, "    ", "message", rec.Message)

		// Labels (sorted by key)
		writeStringMap(&b, "    ", "labels", rec.Labels)

		// Files (sorted by path)
		writeStringMap(&b, "    ", "files", rec.Files)
	}

	return b.String()
}

func writeNullable(b *strings.Builder, indent, key string, val *string) {
	if val == nil {
		fmt.Fprintf(b, "%s%s: null\n", indent, key)
	} else {
		fmt.Fprintf(b, "%s%s: %s\n", indent, key, yamlQuote(*val))
	}
}

func writeStringMap(b *strings.Builder, indent, key string, m map[string]string) {
	if len(m) == 0 {
		fmt.Fprintf(b, "%s%s: {}\n", indent, key)
		return
	}
	fmt.Fprintf(b, "%s%s:\n", indent, key)
	keys := sortedKeys(m)
	for _, k := range keys {
		fmt.Fprintf(b, "%s  %s: %s\n", indent, k, yamlQuote(m[k]))
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func yamlQuote(s string) string {
	if s == "" || s == "null" || s == "true" || s == "false" ||
		strings.ContainsAny(s, ":{}[]#&*!|>'\",\n") ||
		strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") ||
		looksNumeric(s) {
		return fmt.Sprintf("%q", s)
	}
	return s
}

func looksNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			if c != '.' && c != '-' && c != '+' && c != 'e' && c != 'E' {
				return false
			}
		}
	}
	return true
}
