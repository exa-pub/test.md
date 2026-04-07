package resolver

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/testmd/testmd/internal/hashing"
	"github.com/testmd/testmd/internal/models"
	"github.com/testmd/testmd/internal/patterns"
)

// BuildInstances expands definitions into concrete instances with computed hashes.
func BuildInstances(root string, defs []models.TestDefinition, ig *ignore.GitIgnore) ([]*models.TestInstance, error) {
	var instances []*models.TestInstance

	// Lock file to exclude from hashing
	const lockName = ".testmd.lock"

	for i := range defs {
		defn := &defs[i]
		watch := rebasePatterns(root, defn.SourceFile, defn.Watch)
		each := rebaseEach(root, defn.SourceFile, defn.Each)
		combinations := rebaseCombinations(root, defn.SourceFile, defn.Combinations)

		var labelCombos []map[string]string
		if len(combinations) > 0 {
			labelCombos = patterns.ExpandCombinations(root, combinations, ig)
		} else if len(each) > 0 {
			labelCombos = patterns.ExpandEach(root, each, ig)
		} else {
			labelCombos = []map[string]string{{}}
		}

		// Compute relative source path for ID generation
		sourcePath, _ := filepath.Rel(root, defn.SourceFile)
		sourcePath = filepath.ToSlash(sourcePath)

		for _, labels := range labelCombos {
			var resolvedPatterns []string
			allFiles := map[string]bool{}

			for _, pat := range watch {
				resolved := pat
				for k, v := range labels {
					resolved = strings.ReplaceAll(resolved, "{"+k+"}", v)
				}
				resolvedPatterns = append(resolvedPatterns, resolved)

				files, err := patterns.ResolveFiles(root, pat, labels, ig)
				if err != nil {
					return nil, err
				}
				for _, f := range files {
					allFiles[f] = true
				}
			}

			delete(allFiles, lockName)

			matched := make([]string, 0, len(allFiles))
			for f := range allFiles {
				matched = append(matched, f)
			}
			sort.Strings(matched)

			contentHash, fileHashes, err := hashing.HashFiles(root, matched)
			if err != nil {
				return nil, err
			}
			tid := hashing.MakeID(defn.Title, defn.ExplicitID, labels, sourcePath)

			instances = append(instances, &models.TestInstance{
				ID:               tid,
				Definition:       defn,
				Labels:           labels,
				ResolvedPatterns: resolvedPatterns,
				MatchedFiles:     matched,
				ContentHash:      contentHash,
				FileHashes:       fileHashes,
			})
		}
	}
	return instances, nil
}

// ComputeStatuses determines the effective status of each instance.
func ComputeStatuses(instances []*models.TestInstance, st *models.State) []models.StatusResult {
	results := make([]models.StatusResult, len(instances))
	for i, inst := range instances {
		rec := st.Tests[inst.ID]
		status := "pending"
		if rec != nil {
			if rec.ContentHash != inst.ContentHash {
				status = "outdated"
			} else {
				status = rec.Status
			}
		}
		results[i] = models.StatusResult{Instance: inst, Status: status, Record: rec}
	}
	return results
}

// ResolveTest marks a test as resolved.
func ResolveTest(st *models.State, inst *models.TestInstance, root string) {
	now := time.Now().UTC().Format(time.RFC3339)
	st.Tests[inst.ID] = makeRecord(inst, "resolved", root)
	st.Tests[inst.ID].ResolvedAt = &now
}

// FailTest marks a test as failed with a message.
func FailTest(st *models.State, inst *models.TestInstance, message string, root string) {
	now := time.Now().UTC().Format(time.RFC3339)
	st.Tests[inst.ID] = makeRecord(inst, "failed", root)
	st.Tests[inst.ID].FailedAt = &now
	st.Tests[inst.ID].Message = &message
}

// GCState removes orphaned test records. Returns the count removed.
func GCState(st *models.State, instances []*models.TestInstance) int {
	currentIDs := map[string]bool{}
	for _, inst := range instances {
		currentIDs[inst.ID] = true
	}
	var orphans []string
	for id := range st.Tests {
		if !currentIDs[id] {
			orphans = append(orphans, id)
		}
	}
	for _, id := range orphans {
		delete(st.Tests, id)
	}
	return len(orphans)
}

// FindInstances finds instances matching by prefix (pure prefix match).
func FindInstances(instances []*models.TestInstance, query string) []*models.TestInstance {
	// Exact match first
	for _, inst := range instances {
		if inst.ID == query {
			return []*models.TestInstance{inst}
		}
	}
	// Prefix match
	var matches []*models.TestInstance
	for _, inst := range instances {
		if strings.HasPrefix(inst.ID, query) {
			matches = append(matches, inst)
		}
	}
	return matches
}

// ChangedFiles returns files that differ between instance and stored record.
func ChangedFiles(inst *models.TestInstance, rec *models.TestRecord) []string {
	if rec == nil || rec.Files == nil {
		return inst.MatchedFiles
	}
	changed := map[string]bool{}
	for f, h := range inst.FileHashes {
		if rec.Files[f] != h {
			changed[f] = true
		}
	}
	for f := range rec.Files {
		if _, ok := inst.FileHashes[f]; !ok {
			changed[f] = true
		}
	}
	result := make([]string, 0, len(changed))
	for f := range changed {
		result = append(result, f)
	}
	sort.Strings(result)
	return result
}

func rebasePatterns(root, sourceFile string, pats []string) []string {
	sourceDir := filepath.Dir(sourceFile)
	if sourceDir == root {
		return pats
	}
	rel, err := filepath.Rel(root, sourceDir)
	if err != nil {
		return pats
	}
	rel = filepath.ToSlash(rel)

	rebased := make([]string, len(pats))
	for i, p := range pats {
		p = strings.TrimPrefix(p, "./")
		rebased[i] = "./" + rel + "/" + p
	}
	return rebased
}

func rebaseEach(root, sourceFile string, each map[string]models.EachSource) map[string]models.EachSource {
	if len(each) == 0 || filepath.Dir(sourceFile) == root {
		return each
	}
	rel, err := filepath.Rel(root, filepath.Dir(sourceFile))
	if err != nil {
		return each
	}
	relSlash := filepath.ToSlash(rel)

	rebased := make(map[string]models.EachSource, len(each))
	for k, src := range each {
		if src.Glob != "" {
			g := strings.TrimPrefix(src.Glob, "./")
			rebased[k] = models.EachSource{Glob: "./" + relSlash + "/" + g}
		} else {
			rebased[k] = src
		}
	}
	return rebased
}

func rebaseCombinations(root, sourceFile string, combos []map[string]models.EachSource) []map[string]models.EachSource {
	if len(combos) == 0 || filepath.Dir(sourceFile) == root {
		return combos
	}
	rebased := make([]map[string]models.EachSource, len(combos))
	for i, entry := range combos {
		rebased[i] = rebaseEach(root, sourceFile, entry)
	}
	return rebased
}

func makeRecord(inst *models.TestInstance, status string, root string) *models.TestRecord {
	labels := inst.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	files := inst.FileHashes
	if files == nil {
		files = map[string]string{}
	}
	source, _ := filepath.Rel(root, inst.Definition.SourceFile)
	source = filepath.ToSlash(source)
	return &models.TestRecord{
		Title:       inst.Definition.Title,
		Source:      source,
		Labels:      labels,
		ContentHash: inst.ContentHash,
		Files:       files,
		Status:      status,
	}
}
