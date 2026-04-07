package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testmd/testmd/internal/models"
)

func TestLoad(t *testing.T) {
	t.Run("file_not_found", func(t *testing.T) {
		st, err := Load(filepath.Join(t.TempDir(), ".testmd.lock"))
		if err != nil {
			t.Fatal(err)
		}
		if st.Version != 1 {
			t.Errorf("expected version 1, got %d", st.Version)
		}
		if len(st.Tests) != 0 {
			t.Errorf("expected empty tests, got %v", st.Tests)
		}
	})

	t.Run("reads_yaml", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, ".testmd.lock")
		yaml := "version: 1\ntests:\n  abc123def456789abc:\n    status: resolved\n    title: T\n    source: TEST.md\n    content_hash: x\n    resolved_at: null\n    failed_at: null\n    message: null\n    labels: {}\n    files: {}\n"
		os.WriteFile(f, []byte(yaml), 0644)

		st, err := Load(f)
		if err != nil {
			t.Fatal(err)
		}
		if st.Tests["abc123def456789abc"] == nil {
			t.Fatal("expected test record for abc123def456789abc")
		}
		if st.Tests["abc123def456789abc"].Status != "resolved" {
			t.Errorf("expected status 'resolved', got %q", st.Tests["abc123def456789abc"].Status)
		}
	})

	t.Run("nil_tests_map", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, ".testmd.lock")
		os.WriteFile(f, []byte("version: 1\n"), 0644)

		st, err := Load(f)
		if err != nil {
			t.Fatal(err)
		}
		if st.Tests == nil {
			t.Error("expected non-nil tests map")
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("writes_yaml", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, ".testmd.lock")

		st := &models.State{
			Version: 1,
			Tests: map[string]*models.TestRecord{
				"abc123def456789abc": {
					Title:       "T",
					Source:      "TEST.md",
					Status:      "resolved",
					ContentHash: "hash123",
					Labels:      map[string]string{},
					Files:       map[string]string{},
				},
			},
		}
		if err := Save(f, st); err != nil {
			t.Fatal(err)
		}
		text, _ := os.ReadFile(f)
		s := string(text)
		if !strings.Contains(s, "abc123def456789abc:") {
			t.Error("expected test id in YAML output")
		}
		if !strings.Contains(s, "version: 1") {
			t.Error("expected version field")
		}
		if !strings.Contains(s, "status: resolved") {
			t.Error("expected status field")
		}
	})

	t.Run("deterministic_ordering", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, ".testmd.lock")

		st := &models.State{
			Version: 1,
			Tests: map[string]*models.TestRecord{
				"zzz000000000000000": {Title: "Z", Source: "TEST.md", Status: "resolved", ContentHash: "h", Labels: map[string]string{"b": "2", "a": "1"}, Files: map[string]string{"z.go": "h1", "a.go": "h2"}},
				"aaa000000000000000": {Title: "A", Source: "TEST.md", Status: "resolved", ContentHash: "h", Labels: map[string]string{}, Files: map[string]string{}},
			},
		}
		Save(f, st)
		text, _ := os.ReadFile(f)
		s := string(text)

		// aaa should come before zzz
		aIdx := strings.Index(s, "aaa000")
		zIdx := strings.Index(s, "zzz000")
		if aIdx >= zIdx {
			t.Error("expected aaa before zzz (sorted by ID)")
		}

		// labels: a before b
		aLabel := strings.Index(s, "a: \"1\"")
		bLabel := strings.Index(s, "b: \"2\"")
		if aLabel >= bLabel {
			t.Error("expected label 'a' before 'b' (sorted by key)")
		}

		// files: a.go before z.go
		aFile := strings.Index(s, "a.go:")
		zFile := strings.Index(s, "z.go:")
		if aFile >= zFile {
			t.Error("expected file 'a.go' before 'z.go' (sorted by path)")
		}
	})

	t.Run("overwrites_existing", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, ".testmd.lock")

		old := &models.State{Version: 1, Tests: map[string]*models.TestRecord{"old000000000000000": {Title: "Old", Source: "TEST.md", Status: "resolved", ContentHash: "h", Labels: map[string]string{}, Files: map[string]string{}}}}
		Save(f, old)

		new := &models.State{Version: 1, Tests: map[string]*models.TestRecord{"new000000000000000": {Title: "New", Source: "TEST.md", Status: "resolved", ContentHash: "h", Labels: map[string]string{}, Files: map[string]string{}}}}
		Save(f, new)

		text, _ := os.ReadFile(f)
		s := string(text)
		if strings.Contains(s, "old000") {
			t.Error("should not contain old record")
		}
		if !strings.Contains(s, "new000") {
			t.Error("expected new record")
		}
	})

	t.Run("empty_state_deletes_file", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, ".testmd.lock")

		st := &models.State{Version: 1, Tests: map[string]*models.TestRecord{"x00000000000000000": {Title: "X", Source: "TEST.md", Status: "resolved", ContentHash: "h", Labels: map[string]string{}, Files: map[string]string{}}}}
		Save(f, st)

		empty := &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}
		if err := Save(f, empty); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Error("file should be deleted when state is empty")
		}
	})

	t.Run("roundtrip", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, ".testmd.lock")
		ts := "2026-04-06T12:00:00Z"

		st := &models.State{
			Version: 1,
			Tests: map[string]*models.TestRecord{
				"abc123def456789abc": {
					Title:       "My Test",
					Source:      "sub/TEST.md",
					Status:      "resolved",
					ContentHash: "abcdef1234567890",
					ResolvedAt:  &ts,
					Labels:      map[string]string{"env": "prod"},
					Files:       map[string]string{"main.go": "hash1"},
				},
			},
		}
		if err := Save(f, st); err != nil {
			t.Fatal(err)
		}

		loaded, err := Load(f)
		if err != nil {
			t.Fatal(err)
		}
		rec := loaded.Tests["abc123def456789abc"]
		if rec == nil {
			t.Fatal("expected record after roundtrip")
		}
		if rec.Title != "My Test" {
			t.Errorf("title: got %q", rec.Title)
		}
		if rec.Source != "sub/TEST.md" {
			t.Errorf("source: got %q", rec.Source)
		}
		if rec.Status != "resolved" {
			t.Errorf("status: got %q", rec.Status)
		}
		if rec.Labels["env"] != "prod" {
			t.Errorf("labels: got %v", rec.Labels)
		}
		if rec.Files["main.go"] != "hash1" {
			t.Errorf("files: got %v", rec.Files)
		}
		if rec.ResolvedAt == nil || *rec.ResolvedAt != ts {
			t.Errorf("resolved_at: got %v", rec.ResolvedAt)
		}
	})
}
