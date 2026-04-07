package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/testmd/testmd/internal/models"
)

func defn(title string, watch []string, each map[string]models.EachSource, combinations []map[string]models.EachSource, explicitID string, sourceFile string) models.TestDefinition {
	if watch == nil {
		watch = []string{"src/**/*"}
	}
	if sourceFile == "" {
		sourceFile = "TEST.md"
	}
	return models.TestDefinition{
		Title:        title,
		ExplicitID:   explicitID,
		Watch:        watch,
		Each:         each,
		Combinations: combinations,
		Description:  "desc",
		SourceFile:   sourceFile,
		SourceLine:   1,
	}
}

func instance(tid, title string, labels map[string]string, matchedFiles []string, contentHash string, fileHashes map[string]string) *models.TestInstance {
	if title == "" {
		title = "Test"
	}
	if tid == "" {
		tid = "aaabbbcccdddeee111"
	}
	if labels == nil {
		labels = map[string]string{}
	}
	if contentHash == "" {
		contentHash = "hash123"
	}
	if fileHashes == nil {
		fileHashes = map[string]string{}
	}
	d := defn(title, nil, nil, nil, "", "")
	return &models.TestInstance{
		ID:               tid,
		Definition:       &d,
		Labels:           labels,
		ResolvedPatterns: []string{"src/**/*"},
		MatchedFiles:     matchedFiles,
		ContentHash:      contentHash,
		FileHashes:       fileHashes,
	}
}

func emptyState() *models.State {
	return &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}
}

func TestBuildInstances(t *testing.T) {
	t.Run("simple_no_vars", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "src"), 0755)
		os.WriteFile(filepath.Join(tmp, "src", "main.py"), []byte("print('hi')"), 0644)
		sf := filepath.Join(tmp, "TEST.md")

		d := defn("Test", []string{"src/*.py"}, nil, nil, "", sf)
		instances, err := BuildInstances(tmp, []models.TestDefinition{d}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(instances) != 1 {
			t.Fatalf("expected 1 instance, got %d", len(instances))
		}
		found := false
		for _, f := range instances[0].MatchedFiles {
			if f == "src/main.py" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected src/main.py in matched_files, got %v", instances[0].MatchedFiles)
		}
		if len(instances[0].Labels) != 0 {
			t.Errorf("expected empty labels, got %v", instances[0].Labels)
		}
		// ID should be 18 hex chars
		if len(instances[0].ID) != 18 {
			t.Errorf("expected 18-char ID, got %d: %q", len(instances[0].ID), instances[0].ID)
		}
	})

	t.Run("each_explicit_values", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "main.py"), []byte("x"), 0644)
		sf := filepath.Join(tmp, "TEST.md")

		d := defn("Test", []string{"main.py"}, map[string]models.EachSource{
			"env": {Values: []string{"dev", "prod"}},
		}, nil, "", sf)
		instances, err := BuildInstances(tmp, []models.TestDefinition{d}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(instances) != 2 {
			t.Fatalf("expected 2 instances, got %d", len(instances))
		}
		labelsList := make([]map[string]string, len(instances))
		for i, inst := range instances {
			labelsList[i] = inst.Labels
		}
		assertContainsMap(t, labelsList, map[string]string{"env": "dev"})
		assertContainsMap(t, labelsList, map[string]string{"env": "prod"})
	})

	t.Run("each_glob_discovery", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "svcA", "src"), 0755)
		os.WriteFile(filepath.Join(tmp, "svcA", "src", "a.py"), []byte("a"), 0644)
		os.MkdirAll(filepath.Join(tmp, "svcB", "src"), 0755)
		os.WriteFile(filepath.Join(tmp, "svcB", "src", "b.py"), []byte("b"), 0644)
		sf := filepath.Join(tmp, "TEST.md")

		d := defn("Test", []string{"./{svc}/src/*.py"}, map[string]models.EachSource{
			"svc": {Glob: "./*/"},
		}, nil, "", sf)
		instances, err := BuildInstances(tmp, []models.TestDefinition{d}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(instances) != 2 {
			t.Fatalf("expected 2 instances, got %d", len(instances))
		}
	})

	t.Run("combinations_union", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "main.py"), []byte("x"), 0644)
		sf := filepath.Join(tmp, "TEST.md")

		d := defn("Test", []string{"main.py"}, nil, []map[string]models.EachSource{
			{"x": {Values: []string{"1"}}},
			{"x": {Values: []string{"2"}}},
		}, "", sf)
		instances, err := BuildInstances(tmp, []models.TestDefinition{d}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(instances) != 2 {
			t.Fatalf("expected 2 instances, got %d", len(instances))
		}
	})

	t.Run("rebase_from_subdir", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "sub", "lib"), 0755)
		os.WriteFile(filepath.Join(tmp, "sub", "lib", "a.py"), []byte("a"), 0644)
		sf := filepath.Join(tmp, "sub", "TEST.md")

		d := defn("Test", []string{"./lib/*.py"}, nil, nil, "", sf)
		instances, err := BuildInstances(tmp, []models.TestDefinition{d}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(instances) != 1 {
			t.Fatalf("expected 1 instance, got %d", len(instances))
		}
		found := false
		for _, f := range instances[0].MatchedFiles {
			if f == "sub/lib/a.py" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected sub/lib/a.py in matched_files, got %v", instances[0].MatchedFiles)
		}
	})

	t.Run("source_path_in_id", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "main.py"), []byte("x"), 0644)
		sf1 := filepath.Join(tmp, "TEST.md")
		sf2 := filepath.Join(tmp, "sub", "TEST.md")
		os.MkdirAll(filepath.Join(tmp, "sub"), 0755)

		d1 := defn("Test", []string{"main.py"}, nil, nil, "", sf1)
		d2 := defn("Test", []string{"main.py"}, nil, nil, "", sf2)

		inst1, _ := BuildInstances(tmp, []models.TestDefinition{d1}, nil)
		inst2, _ := BuildInstances(tmp, []models.TestDefinition{d2}, nil)

		if inst1[0].ID == inst2[0].ID {
			t.Errorf("same title in different files should have different IDs: %s vs %s", inst1[0].ID, inst2[0].ID)
		}
		// First 12 chars (title+labels) should be same
		if inst1[0].ID[:12] != inst2[0].ID[:12] {
			t.Errorf("first 12 chars should match (same title+labels): %s vs %s", inst1[0].ID[:12], inst2[0].ID[:12])
		}
	})
}

func TestComputeStatuses(t *testing.T) {
	t.Run("pending", func(t *testing.T) {
		inst := instance("", "", nil, nil, "", nil)
		st := emptyState()
		results := ComputeStatuses([]*models.TestInstance{inst}, st)
		if results[0].Status != "pending" {
			t.Errorf("expected pending, got %s", results[0].Status)
		}
	})

	t.Run("resolved", func(t *testing.T) {
		inst := instance("aaabbbcccdddeee111", "", nil, nil, "abc", nil)
		st := emptyState()
		st.Tests["aaabbbcccdddeee111"] = &models.TestRecord{ContentHash: "abc", Status: "resolved"}
		results := ComputeStatuses([]*models.TestInstance{inst}, st)
		if results[0].Status != "resolved" {
			t.Errorf("expected resolved, got %s", results[0].Status)
		}
	})

	t.Run("outdated", func(t *testing.T) {
		inst := instance("aaabbbcccdddeee111", "", nil, nil, "new_hash", nil)
		st := emptyState()
		st.Tests["aaabbbcccdddeee111"] = &models.TestRecord{ContentHash: "old_hash", Status: "resolved"}
		results := ComputeStatuses([]*models.TestInstance{inst}, st)
		if results[0].Status != "outdated" {
			t.Errorf("expected outdated, got %s", results[0].Status)
		}
	})

	t.Run("failed", func(t *testing.T) {
		inst := instance("aaabbbcccdddeee111", "", nil, nil, "h", nil)
		st := emptyState()
		st.Tests["aaabbbcccdddeee111"] = &models.TestRecord{ContentHash: "h", Status: "failed"}
		results := ComputeStatuses([]*models.TestInstance{inst}, st)
		if results[0].Status != "failed" {
			t.Errorf("expected failed, got %s", results[0].Status)
		}
	})
}

func TestResolveTest(t *testing.T) {
	t.Run("updates_state", func(t *testing.T) {
		tmp := t.TempDir()
		inst := instance("aaabbbcccdddeee111", "", nil, []string{"f.txt"}, "ch", map[string]string{"f.txt": "fh"})
		inst.Definition.SourceFile = filepath.Join(tmp, "TEST.md")
		st := emptyState()
		ResolveTest(st, inst, tmp)

		rec := st.Tests["aaabbbcccdddeee111"]
		if rec == nil {
			t.Fatal("expected record")
		}
		if rec.Status != "resolved" {
			t.Errorf("expected status 'resolved', got %q", rec.Status)
		}
		if rec.ContentHash != "ch" {
			t.Errorf("expected content_hash 'ch', got %q", rec.ContentHash)
		}
		if rec.Files["f.txt"] != "fh" {
			t.Errorf("expected files[f.txt]='fh', got %v", rec.Files)
		}
		if rec.ResolvedAt == nil {
			t.Error("expected resolved_at to be set")
		}
		if rec.Source != "TEST.md" {
			t.Errorf("expected source 'TEST.md', got %q", rec.Source)
		}
	})
}

func TestFailTest(t *testing.T) {
	t.Run("updates_state_with_message", func(t *testing.T) {
		tmp := t.TempDir()
		inst := instance("aaabbbcccdddeee111", "", nil, nil, "", nil)
		inst.Definition.SourceFile = filepath.Join(tmp, "TEST.md")
		st := emptyState()
		FailTest(st, inst, "broken", tmp)

		rec := st.Tests["aaabbbcccdddeee111"]
		if rec == nil {
			t.Fatal("expected record")
		}
		if rec.Status != "failed" {
			t.Errorf("expected status 'failed', got %q", rec.Status)
		}
		if rec.Message == nil || *rec.Message != "broken" {
			t.Errorf("expected message 'broken', got %v", rec.Message)
		}
	})
}

func TestGCState(t *testing.T) {
	t.Run("removes_orphans", func(t *testing.T) {
		inst := instance("aaabbbcccdddeee111", "", nil, nil, "", nil)
		st := emptyState()
		st.Tests["aaabbbcccdddeee111"] = &models.TestRecord{Status: "resolved"}
		st.Tests["orphan000000000000"] = &models.TestRecord{Status: "resolved"}

		removed := GCState(st, []*models.TestInstance{inst})
		if removed != 1 {
			t.Errorf("expected 1 removed, got %d", removed)
		}
	})

	t.Run("nothing_to_gc", func(t *testing.T) {
		inst := instance("aaabbbcccdddeee111", "", nil, nil, "", nil)
		st := emptyState()
		st.Tests["aaabbbcccdddeee111"] = &models.TestRecord{Status: "resolved"}

		removed := GCState(st, []*models.TestInstance{inst})
		if removed != 0 {
			t.Errorf("expected 0 removed, got %d", removed)
		}
	})
}

func TestFindInstances(t *testing.T) {
	mkInstances := func() []*models.TestInstance {
		return []*models.TestInstance{
			instance("abc123def456111222", "", nil, nil, "", nil),
			instance("abc123aaa111222333", "", nil, nil, "", nil),
			instance("xyz789bbb222333444", "", nil, nil, "", nil),
		}
	}

	t.Run("exact_match", func(t *testing.T) {
		result := FindInstances(mkInstances(), "abc123def456111222")
		if len(result) != 1 || result[0].ID != "abc123def456111222" {
			t.Errorf("expected exact match, got %v", ids(result))
		}
	})

	t.Run("prefix_6_chars_matches_title", func(t *testing.T) {
		result := FindInstances(mkInstances(), "abc123")
		if len(result) != 2 {
			t.Errorf("expected 2 matches, got %d: %v", len(result), ids(result))
		}
	})

	t.Run("prefix_match", func(t *testing.T) {
		result := FindInstances(mkInstances(), "xyz")
		if len(result) != 1 || result[0].ID != "xyz789bbb222333444" {
			t.Errorf("expected xyz789bbb222333444, got %v", ids(result))
		}
	})

	t.Run("no_match", func(t *testing.T) {
		result := FindInstances(mkInstances(), "zzzzz")
		if len(result) != 0 {
			t.Errorf("expected no matches, got %v", ids(result))
		}
	})
}

func TestChangedFiles(t *testing.T) {
	t.Run("no_record", func(t *testing.T) {
		inst := instance("", "", nil, []string{"a.txt", "b.txt"}, "", nil)
		result := ChangedFiles(inst, nil)
		if len(result) != 2 {
			t.Errorf("expected 2 files, got %v", result)
		}
	})

	t.Run("modified_file", func(t *testing.T) {
		inst := instance("", "", nil, nil, "", map[string]string{"a.txt": "new_hash", "b.txt": "same"})
		rec := &models.TestRecord{Files: map[string]string{"a.txt": "old_hash", "b.txt": "same"}}
		result := ChangedFiles(inst, rec)
		if len(result) != 1 || result[0] != "a.txt" {
			t.Errorf("expected [a.txt], got %v", result)
		}
	})

	t.Run("no_changes", func(t *testing.T) {
		inst := instance("", "", nil, nil, "", map[string]string{"a.txt": "h"})
		rec := &models.TestRecord{Files: map[string]string{"a.txt": "h"}}
		result := ChangedFiles(inst, rec)
		if len(result) != 0 {
			t.Errorf("expected [], got %v", result)
		}
	})
}

func ids(insts []*models.TestInstance) []string {
	result := make([]string, len(insts))
	for i, inst := range insts {
		result[i] = inst.ID
	}
	return result
}

func assertContainsMap(t *testing.T, list []map[string]string, want map[string]string) {
	t.Helper()
	for _, m := range list {
		if len(m) != len(want) {
			continue
		}
		match := true
		for k, v := range want {
			if m[k] != v {
				match = false
				break
			}
		}
		if match {
			return
		}
	}
	t.Errorf("expected %v in %v", want, list)
}
