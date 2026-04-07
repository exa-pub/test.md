package parser

import (
	"strings"
	"testing"
)

func makeTest(title, watch string) string {
	return "# " + title + "\n\nSome description.\n\n```yaml\nwatch: " + watch + "\n```\n"
}

func TestParseSingleTest_Basic(t *testing.T) {
	text := makeTest("My Test", "src/**/*")
	tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(tests))
	}
	tt := tests[0]
	if tt.Title != "My Test" {
		t.Errorf("expected title 'My Test', got %q", tt.Title)
	}
	if len(tt.Watch) != 1 || tt.Watch[0] != "src/**/*" {
		t.Errorf("expected watch [src/**/*], got %v", tt.Watch)
	}
	if tt.ExplicitID != "" {
		t.Errorf("expected empty explicit_id, got %q", tt.ExplicitID)
	}
	if tt.Each != nil {
		t.Errorf("expected nil each, got %v", tt.Each)
	}
	if tt.Combinations != nil {
		t.Errorf("expected nil combinations, got %v", tt.Combinations)
	}
	if !strings.Contains(tt.Description, "Some description.") {
		t.Errorf("expected description to contain 'Some description.', got %q", tt.Description)
	}
}

func TestParseSingleTest_SourceLine(t *testing.T) {
	text := makeTest("My Test", "src/**/*")
	tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if tests[0].SourceLine != 1 {
		t.Errorf("expected source_line 1, got %d", tests[0].SourceLine)
	}
}

func TestParseSingleTest_WatchAsList(t *testing.T) {
	text := "# T\n\n```yaml\nwatch:\n  - a.txt\n  - b.txt\n```\n"
	tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(tests[0].Watch) != 2 || tests[0].Watch[0] != "a.txt" || tests[0].Watch[1] != "b.txt" {
		t.Errorf("expected [a.txt b.txt], got %v", tests[0].Watch)
	}
}

func TestParseSingleTest_WatchAsString(t *testing.T) {
	text := "# T\n\n```yaml\nwatch: foo.py\n```\n"
	tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(tests[0].Watch) != 1 || tests[0].Watch[0] != "foo.py" {
		t.Errorf("expected [foo.py], got %v", tests[0].Watch)
	}
}

func TestParseSingleTest_ExplicitID(t *testing.T) {
	text := "# T\n\n```yaml\nid: my-id\nwatch: x\n```\n"
	tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if tests[0].ExplicitID != "my-id" {
		t.Errorf("expected explicit_id 'my-id', got %q", tests[0].ExplicitID)
	}
}

func TestParseSingleTest_EachParsed(t *testing.T) {
	text := "# T\n\n```yaml\neach:\n  svc: ./services/*/\n  env: [prod, staging]\nwatch: ./services/{svc}/**\n```\n"
	tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if tests[0].Each == nil {
		t.Fatal("expected non-nil each")
	}
	svc := tests[0].Each["svc"]
	if svc.Glob != "./services/*/" {
		t.Errorf("expected svc glob './services/*/', got %q", svc.Glob)
	}
	env := tests[0].Each["env"]
	if len(env.Values) != 2 || env.Values[0] != "prod" || env.Values[1] != "staging" {
		t.Errorf("expected env values [prod staging], got %v", env.Values)
	}
}

func TestParseSingleTest_CombinationsParsed(t *testing.T) {
	text := "# T\n\n```yaml\ncombinations:\n  - db: [postgres, mysql]\n    suite: [full]\n  - db: [sqlite]\n    suite: [basic]\nwatch: ./migrations/{db}/**\n```\n"
	tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(tests[0].Combinations) != 2 {
		t.Fatalf("expected 2 combination entries, got %d", len(tests[0].Combinations))
	}
	db := tests[0].Combinations[0]["db"]
	if len(db.Values) != 2 || db.Values[0] != "postgres" {
		t.Errorf("expected db values [postgres mysql], got %v", db.Values)
	}
}

func TestParseMultipleTests_TwoTests(t *testing.T) {
	text := makeTest("First", "a.txt") + "\n" + makeTest("Second", "b.txt")
	tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(tests))
	}
	if tests[0].Title != "First" {
		t.Errorf("expected first test title 'First', got %q", tests[0].Title)
	}
	if tests[1].Title != "Second" {
		t.Errorf("expected second test title 'Second', got %q", tests[1].Title)
	}
}

func TestNoFrontmatter_PlainMarkdown(t *testing.T) {
	// Frontmatter is no longer parsed — it's treated as plain content
	text := "---\ninclude:\n  - other.md\n---\n" + makeTest("My Test", "src/**/*")
	tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	// The --- block is not a heading, so only the # heading produces a test
	if len(tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(tests))
	}
}

func TestErrors_MissingYamlBlock(t *testing.T) {
	text := "# T\n\nNo yaml here.\n"
	_, err := Parse(text, "TEST.md")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing yaml config block") {
		t.Errorf("expected 'missing yaml config block' in error, got %q", err.Error())
	}
}

func TestErrors_MissingWatch(t *testing.T) {
	text := "# T\n\n```yaml\nid: foo\n```\n"
	_, err := Parse(text, "TEST.md")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing watch") {
		t.Errorf("expected 'missing watch' in error, got %q", err.Error())
	}
}

func TestErrors_EachAndCombinations(t *testing.T) {
	text := "# T\n\n```yaml\neach:\n  x: [a]\ncombinations:\n  - x: [b]\nwatch: x\n```\n"
	_, err := Parse(text, "TEST.md")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot use both") {
		t.Errorf("expected 'cannot use both' in error, got %q", err.Error())
	}
}

func TestDescriptionContent_YamlBlockExcluded(t *testing.T) {
	text := "# T\n\nBefore yaml.\n\n```yaml\nwatch: x\n```\n\nAfter yaml.\n"
	tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	desc := tests[0].Description
	if !strings.Contains(desc, "Before yaml.") {
		t.Errorf("expected 'Before yaml.' in description, got %q", desc)
	}
	if !strings.Contains(desc, "After yaml.") {
		t.Errorf("expected 'After yaml.' in description, got %q", desc)
	}
	if strings.Contains(desc, "watch") {
		t.Errorf("description should not contain 'watch', got %q", desc)
	}
}

func TestParseConfig_Defaults(t *testing.T) {
	cfg, err := ParseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Ignorefile != ".gitignore" {
		t.Errorf("expected default ignorefile '.gitignore', got %q", cfg.Ignorefile)
	}
}

func TestParseConfig_CustomIgnorefile(t *testing.T) {
	cfg, err := ParseConfig([]byte("ignorefile: .myignore\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Ignorefile != ".myignore" {
		t.Errorf("expected ignorefile '.myignore', got %q", cfg.Ignorefile)
	}
}

func TestParseConfig_EmptyFile(t *testing.T) {
	cfg, err := ParseConfig([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Ignorefile != ".gitignore" {
		t.Errorf("expected default ignorefile '.gitignore', got %q", cfg.Ignorefile)
	}
}
