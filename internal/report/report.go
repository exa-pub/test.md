package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/testmd/testmd/internal/models"
	"github.com/testmd/testmd/internal/resolver"
)

var statusStyles = map[string]struct {
	icon  string
	color *color.Color
}{
	"resolved": {"✓", color.New(color.FgGreen)},
	"failed":   {"✗", color.New(color.FgRed)},
	"outdated": {"⟳", color.New(color.FgYellow)},
	"pending":  {"…", color.New(color.FgCyan)},
}

// PrintStatus prints the status of all tests to stdout, grouped by source file.
func PrintStatus(results []models.StatusResult, root string) {
	groups := groupBySource(results, root)

	for _, sg := range groups {
		dim := color.New(color.Faint)
		dim.Println(sg.source)
		for _, g := range sg.defs {
			bold := color.New(color.Bold)
			bold.Printf("  %s\n", g.title)
			for _, r := range g.results {
				printInstanceLine(r)
			}
		}
		fmt.Println()
	}
	printSummary(results)
}

// PrintGet prints detailed info about a test instance.
func PrintGet(r models.StatusResult) {
	inst := r.Instance
	bold := color.New(color.Bold)

	title := substituteLabels(inst.Definition.Title, inst.Labels)
	bold.Printf("# %s\n", title)

	if len(inst.Labels) > 0 {
		fmt.Printf("Labels: %s\n", FormatLabels(inst.Labels))
	}
	s := statusStyles[r.Status]
	fmt.Printf("Status: %s\n", s.color.Sprint(r.Status))

	if r.Record != nil {
		if r.Record.ResolvedAt != nil {
			fmt.Printf("Resolved at: %s\n", *r.Record.ResolvedAt)
		}
		if r.Record.FailedAt != nil {
			fmt.Printf("Failed at: %s\n", *r.Record.FailedAt)
		}
		if r.Record.Message != nil {
			fmt.Printf("Message: %s\n", *r.Record.Message)
		}
	}

	fmt.Printf("Patterns: %s\n", strings.Join(inst.ResolvedPatterns, ", "))
	fmt.Printf("Files: %d\n", len(inst.MatchedFiles))

	if r.Status == "outdated" {
		diff := resolver.ChangedFiles(inst, r.Record)
		if len(diff) > 0 {
			color.Yellow("Changed:")
			for _, f := range diff {
				fmt.Printf("  %s\n", f)
			}
		}
	}

	fmt.Println("---")
	fmt.Println(substituteLabels(inst.Definition.Description, inst.Labels))
}

// FormatLabels formats labels as "key=val key2=val2" sorted by key.
func FormatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + labels[k]
	}
	return strings.Join(parts, " ")
}

// WriteReportMD writes a markdown report to a file.
func WriteReportMD(results []models.StatusResult, path string) error {
	var sb strings.Builder
	sb.WriteString("# Test Report\n\n")

	groups := groupByDefinition(results)
	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("## %s\n\n", g.title))
		sb.WriteString("| ID | Labels | Status | Message |\n")
		sb.WriteString("|----|--------|--------|--------|\n")
		for _, r := range g.results {
			s := statusStyles[r.Status]
			labels := FormatLabels(r.Instance.Labels)
			if labels == "" {
				labels = "—"
			}
			msg := ""
			if r.Record != nil && r.Record.Message != nil {
				msg = *r.Record.Message
			}
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s %s | %s |\n",
				r.Instance.ID, labels, s.icon, r.Status, msg))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Summary\n\n")
	counts := countStatuses(results)
	for _, s := range []string{"resolved", "failed", "outdated", "pending"} {
		sb.WriteString(fmt.Sprintf("- %s: %d\n", s, counts[s]))
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return err
	}
	fmt.Printf("Report saved to %s\n", path)
	return nil
}

// WriteReportJSON writes a JSON report to a file.
func WriteReportJSON(results []models.StatusResult, path string) error {
	type testEntry struct {
		ID      string            `json:"id"`
		Title   string            `json:"title"`
		Source  string            `json:"source"`
		Labels  map[string]string `json:"labels"`
		Status  string            `json:"status"`
		Message *string           `json:"message"`
		Files   []string          `json:"files"`
	}

	entries := make([]testEntry, len(results))
	for i, r := range results {
		var msg *string
		if r.Record != nil {
			msg = r.Record.Message
		}
		entries[i] = testEntry{
			ID:      r.Instance.ID,
			Title:   r.Instance.Definition.Title,
			Source:  r.Instance.Definition.SourceFile,
			Labels:  r.Instance.Labels,
			Status:  r.Status,
			Message: msg,
			Files:   r.Instance.MatchedFiles,
		}
	}

	data := map[string]interface{}{
		"tests":   entries,
		"summary": countStatuses(results),
	}

	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, append(body, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("Report saved to %s\n", path)
	return nil
}

// --- grouping ---

type sourceGroup struct {
	source string
	defs   []defGroup
}

type defGroup struct {
	title   string
	defn    *models.TestDefinition
	results []models.StatusResult
}

func groupBySource(results []models.StatusResult, root string) []sourceGroup {
	type key struct {
		source string
		defn   *models.TestDefinition
	}
	sourceOrder := []string{}
	sourceMap := map[string][]defGroup{}

	for _, r := range results {
		defn := r.Instance.Definition
		src, _ := filepath.Rel(root, defn.SourceFile)
		src = filepath.ToSlash(src)

		defs, exists := sourceMap[src]
		if !exists {
			sourceOrder = append(sourceOrder, src)
		}

		if len(defs) > 0 && defs[len(defs)-1].defn == defn {
			defs[len(defs)-1].results = append(defs[len(defs)-1].results, r)
		} else {
			defs = append(defs, defGroup{
				title:   defn.Title,
				defn:    defn,
				results: []models.StatusResult{r},
			})
		}
		sourceMap[src] = defs
	}

	groups := make([]sourceGroup, len(sourceOrder))
	for i, src := range sourceOrder {
		groups[i] = sourceGroup{source: src, defs: sourceMap[src]}
	}
	return groups
}

type group struct {
	title   string
	defn    *models.TestDefinition
	results []models.StatusResult
}

func groupByDefinition(results []models.StatusResult) []group {
	var groups []group
	for _, r := range results {
		defn := r.Instance.Definition
		if len(groups) > 0 && groups[len(groups)-1].defn == defn {
			groups[len(groups)-1].results = append(groups[len(groups)-1].results, r)
		} else {
			groups = append(groups, group{
				title:   defn.Title,
				defn:    defn,
				results: []models.StatusResult{r},
			})
		}
	}
	return groups
}

func printInstanceLine(r models.StatusResult) {
	s := statusStyles[r.Status]
	dim := color.New(color.Faint)

	fmt.Print("    ")
	s.color.Print(s.icon)
	fmt.Print(" ")
	dim.Print(r.Instance.ID)

	if len(r.Instance.Labels) > 0 {
		fmt.Print("  " + FormatLabels(r.Instance.Labels))
	}

	fmt.Print("  ")
	s.color.Print(r.Status)

	if r.Status == "failed" && r.Record != nil && r.Record.Message != nil {
		fmt.Printf("  \"%s\"", *r.Record.Message)
	} else if (r.Status == "resolved" || r.Status == "failed") && r.Record != nil {
		ts := r.Record.ResolvedAt
		if ts == nil {
			ts = r.Record.FailedAt
		}
		if ts != nil {
			if ago := timeAgo(*ts); ago != "" {
				fmt.Printf("  (%s)", ago)
			}
		}
	}
	fmt.Println()
}

// PrintCI prints failing tests in the same grouped format as status.
func PrintCI(failing []models.StatusResult, root string) {
	groups := groupBySource(failing, root)
	for _, sg := range groups {
		dim := color.New(color.Faint)
		dim.Println(sg.source)
		for _, g := range sg.defs {
			bold := color.New(color.Bold)
			bold.Printf("  %s\n", g.title)
			for _, r := range g.results {
				printInstanceLine(r)
			}
		}
		fmt.Println()
	}
}

func printSummary(results []models.StatusResult) {
	counts := countStatuses(results)
	parts := []struct {
		status string
		c      *color.Color
	}{
		{"resolved", color.New(color.FgGreen)},
		{"failed", color.New(color.FgRed)},
		{"outdated", color.New(color.FgYellow)},
		{"pending", color.New(color.FgCyan)},
	}

	fmt.Print("Summary: ")
	for i, p := range parts {
		if i > 0 {
			fmt.Print(", ")
		}
		p.c.Printf("%d %s", counts[p.status], p.status)
	}
	fmt.Println()
}

func countStatuses(results []models.StatusResult) map[string]int {
	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status]++
	}
	return counts
}

func substituteLabels(text string, labels map[string]string) string {
	for k, v := range labels {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

func timeAgo(isoStr string) string {
	t, err := time.Parse(time.RFC3339, isoStr)
	if err != nil {
		return ""
	}
	secs := int(time.Since(t).Seconds())
	if secs < 0 {
		return ""
	}
	if secs < 60 {
		return fmt.Sprintf("%ds ago", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%dm ago", secs/60)
	}
	if secs < 86400 {
		return fmt.Sprintf("%dh ago", secs/3600)
	}
	return fmt.Sprintf("%dd ago", secs/86400)
}
