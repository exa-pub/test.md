package hashing

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	t.Run("consistent", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "f.txt"), []byte("hello"), 0644)

		h1, err := HashFile(tmp, "f.txt")
		if err != nil {
			t.Fatal(err)
		}
		h2, err := HashFile(tmp, "f.txt")
		if err != nil {
			t.Fatal(err)
		}
		if h1 != h2 {
			t.Errorf("hash should be consistent: %s != %s", h1, h2)
		}
	})

	t.Run("includes_path", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("hello"), 0644)
		os.WriteFile(filepath.Join(tmp, "b.txt"), []byte("hello"), 0644)

		ha, _ := HashFile(tmp, "a.txt")
		hb, _ := HashFile(tmp, "b.txt")
		if ha == hb {
			t.Errorf("same content but different paths should have different hashes")
		}
	})

	t.Run("known_value", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "x.txt"), []byte("data"), 0644)

		h := sha256.New()
		h.Write([]byte("x.txt"))
		h.Write([]byte{0})
		h.Write([]byte("data"))
		expected := fmt.Sprintf("%x", h.Sum(nil))

		actual, _ := HashFile(tmp, "x.txt")
		if actual != expected {
			t.Errorf("expected %s, got %s", expected, actual)
		}
	})
}

func TestHashFiles(t *testing.T) {
	t.Run("empty_list", func(t *testing.T) {
		tmp := t.TempDir()
		ch, fh, err := HashFiles(tmp, []string{})
		if err != nil {
			t.Fatal(err)
		}
		if len(fh) != 0 {
			t.Errorf("expected empty file hashes, got %d", len(fh))
		}
		// sha256 of empty string
		if ch != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
			t.Errorf("expected sha256 of empty, got %s", ch)
		}
	})

	t.Run("multiple_files", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("aaa"), 0644)
		os.WriteFile(filepath.Join(tmp, "b.txt"), []byte("bbb"), 0644)

		ch, fh, err := HashFiles(tmp, []string{"a.txt", "b.txt"})
		if err != nil {
			t.Fatal(err)
		}
		if len(fh) != 2 {
			t.Errorf("expected 2 file hashes, got %d", len(fh))
		}
		if ch == "" {
			t.Error("content hash should not be empty")
		}
	})

	t.Run("content_change_changes_hash", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "f.txt"), []byte("v1"), 0644)
		h1, _, _ := HashFiles(tmp, []string{"f.txt"})

		os.WriteFile(filepath.Join(tmp, "f.txt"), []byte("v2"), 0644)
		h2, _, _ := HashFiles(tmp, []string{"f.txt"})

		if h1 == h2 {
			t.Error("content hash should change when file content changes")
		}
	})
}

func TestMakeID(t *testing.T) {
	t.Run("format_18_hex_no_dashes", func(t *testing.T) {
		tid := MakeID("My Test", "", map[string]string{}, "TEST.md")
		if len(tid) != 18 {
			t.Errorf("expected length 18, got %d: %q", len(tid), tid)
		}
		for _, c := range tid {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("expected hex chars only, got %c in %q", c, tid)
			}
		}
	})

	t.Run("explicit_id_overrides_title", func(t *testing.T) {
		id1 := MakeID("Title A", "custom", map[string]string{}, "TEST.md")
		id2 := MakeID("Title B", "custom", map[string]string{}, "TEST.md")
		// First 6 chars (title segment) should be same
		if id1[:6] != id2[:6] {
			t.Errorf("same explicit_id should produce same first segment: %s vs %s", id1[:6], id2[:6])
		}
	})

	t.Run("labels_affect_second_segment", func(t *testing.T) {
		id1 := MakeID("T", "", map[string]string{}, "TEST.md")
		id2 := MakeID("T", "", map[string]string{"svc": "web"}, "TEST.md")
		if id1[:6] != id2[:6] {
			t.Errorf("same title should produce same first segment: %s vs %s", id1[:6], id2[:6])
		}
		if id1[6:12] == id2[6:12] {
			t.Errorf("different labels should produce different second segment: %s vs %s", id1[6:12], id2[6:12])
		}
	})

	t.Run("source_path_affects_third_segment", func(t *testing.T) {
		id1 := MakeID("T", "", map[string]string{}, "TEST.md")
		id2 := MakeID("T", "", map[string]string{}, "sub/TEST.md")
		if id1[:12] != id2[:12] {
			t.Errorf("same title+labels should produce same first two segments: %s vs %s", id1[:12], id2[:12])
		}
		if id1[12:] == id2[12:] {
			t.Errorf("different source paths should produce different third segment: %s vs %s", id1[12:], id2[12:])
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		id1 := MakeID("X", "", map[string]string{"a": "1"}, "TEST.md")
		id2 := MakeID("X", "", map[string]string{"a": "1"}, "TEST.md")
		if id1 != id2 {
			t.Errorf("should be deterministic: %s != %s", id1, id2)
		}
	})

	t.Run("different_titles_different_first_segment", func(t *testing.T) {
		id1 := MakeID("Alpha", "", map[string]string{}, "TEST.md")
		id2 := MakeID("Beta", "", map[string]string{}, "TEST.md")
		if id1[:6] == id2[:6] {
			t.Errorf("different titles should produce different first segments: %s vs %s", id1[:6], id2[:6])
		}
	})
}
