package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"

	"github.com/testmd/testmd/internal/models"
	"github.com/testmd/testmd/internal/parser"
	"github.com/testmd/testmd/internal/patterns"
	"github.com/testmd/testmd/internal/report"
	"github.com/testmd/testmd/internal/resolver"
	"github.com/testmd/testmd/internal/state"
)

type context struct {
	root      string
	instances []*models.TestInstance
	state     *models.State
}

// Run executes the CLI.
func Run() {
	rootCmd := &cobra.Command{
		Use:           "testmd",
		Short:         "Track manual tests in markdown",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	var rootFlag string
	rootCmd.PersistentFlags().StringVar(&rootFlag, "root", "", "Path to project root (directory containing .testmd.yaml)")

	rootCmd.AddCommand(
		statusCmd(&rootFlag),
		resolveCmd(&rootFlag),
		failCmd(&rootFlag),
		resetCmd(&rootFlag),
		getCmd(&rootFlag),
		gcCmd(&rootFlag),
		ciCmd(&rootFlag),
		initCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func statusCmd(rootFlag *string) *cobra.Command {
	var reportMD, reportJSON string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*rootFlag)
			if err != nil {
				return err
			}
			results := resolver.ComputeStatuses(ctx.instances, ctx.state)
			report.PrintStatus(results, ctx.root)
			if reportMD != "" {
				if err := report.WriteReportMD(results, reportMD); err != nil {
					return err
				}
			}
			if reportJSON != "" {
				if err := report.WriteReportJSON(results, reportJSON); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&reportMD, "report-md", "", "Save markdown report")
	cmd.Flags().StringVar(&reportJSON, "report-json", "", "Save JSON report")
	return cmd
}

func resolveCmd(rootFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <id>",
		Short: "Mark test(s) as resolved",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*rootFlag)
			if err != nil {
				return err
			}
			matches := resolver.FindInstances(ctx.instances, args[0])
			if len(matches) == 0 {
				return fmt.Errorf("no test matching '%s'", args[0])
			}

			lk := lockPath(ctx.root)
			f, err := state.Lock(lk)
			if err != nil {
				return fmt.Errorf("lock: %w", err)
			}
			defer state.Unlock(f)

			// Reload state under lock
			ctx.state, err = state.Load(lk)
			if err != nil {
				return err
			}

			for _, inst := range matches {
				resolver.ResolveTest(ctx.state, inst, ctx.root)
				suffix := labelSuffix(inst.Labels)
				fmt.Printf("Resolved: %s%s\n", inst.Definition.Title, suffix)
			}
			return state.Save(lk, ctx.state)
		},
	}
}

func failCmd(rootFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "fail <id> <message>",
		Short: "Mark test as failed with a message",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*rootFlag)
			if err != nil {
				return err
			}
			matches := resolver.FindInstances(ctx.instances, args[0])
			if len(matches) == 0 {
				return fmt.Errorf("no test matching '%s'", args[0])
			}

			lk := lockPath(ctx.root)
			f, err := state.Lock(lk)
			if err != nil {
				return fmt.Errorf("lock: %w", err)
			}
			defer state.Unlock(f)

			ctx.state, err = state.Load(lk)
			if err != nil {
				return err
			}

			for _, inst := range matches {
				resolver.FailTest(ctx.state, inst, args[1], ctx.root)
				suffix := labelSuffix(inst.Labels)
				fmt.Printf("Failed: %s%s\n", inst.Definition.Title, suffix)
				fmt.Printf("  Message: %s\n", args[1])
			}
			return state.Save(lk, ctx.state)
		},
	}
}

func resetCmd(rootFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "reset <id>",
		Short: "Reset test(s) to pending (remove stored state)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*rootFlag)
			if err != nil {
				return err
			}
			matches := resolver.FindInstances(ctx.instances, args[0])
			if len(matches) == 0 {
				return fmt.Errorf("no test matching '%s'", args[0])
			}

			lk := lockPath(ctx.root)
			f, err := state.Lock(lk)
			if err != nil {
				return fmt.Errorf("lock: %w", err)
			}
			defer state.Unlock(f)

			ctx.state, err = state.Load(lk)
			if err != nil {
				return err
			}

			for _, inst := range matches {
				resolver.ResetTest(ctx.state, inst)
				suffix := labelSuffix(inst.Labels)
				fmt.Printf("Reset: %s%s\n", inst.Definition.Title, suffix)
			}
			return state.Save(lk, ctx.state)
		},
	}
}

func getCmd(rootFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show test details and description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*rootFlag)
			if err != nil {
				return err
			}
			matches := resolver.FindInstances(ctx.instances, args[0])
			if len(matches) == 0 {
				return fmt.Errorf("no test matching '%s'", args[0])
			}
			results := resolver.ComputeStatuses(matches, ctx.state)
			for i, r := range results {
				report.PrintGet(r)
				if i < len(results)-1 {
					fmt.Println()
				}
			}
			return nil
		},
	}
}

func gcCmd(rootFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "gc",
		Short: "Remove orphaned test records",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*rootFlag)
			if err != nil {
				return err
			}

			lk := lockPath(ctx.root)
			f, err := state.Lock(lk)
			if err != nil {
				return fmt.Errorf("lock: %w", err)
			}
			defer state.Unlock(f)

			ctx.state, err = state.Load(lk)
			if err != nil {
				return err
			}

			n := resolver.GCState(ctx.state, ctx.instances)
			if err := state.Save(lk, ctx.state); err != nil {
				return err
			}
			fmt.Printf("Removed %d orphaned record(s).\n", n)
			return nil
		},
	}
}

func ciCmd(rootFlag *string) *cobra.Command {
	var reportMD, reportJSON string
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "Check all tests pass (for CI). Exits 1 if any test needs attention",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := load(*rootFlag)
			if err != nil {
				return err
			}
			results := resolver.ComputeStatuses(ctx.instances, ctx.state)

			if reportMD != "" {
				if err := report.WriteReportMD(results, reportMD); err != nil {
					return err
				}
			}
			if reportJSON != "" {
				if err := report.WriteReportJSON(results, reportJSON); err != nil {
					return err
				}
			}

			var failing []models.StatusResult
			for _, r := range results {
				if r.Status != "resolved" {
					failing = append(failing, r)
				}
			}

			if len(failing) == 0 {
				color.Green("OK: all tests resolved")
				return nil
			}

			bold := color.New(color.FgRed, color.Bold)
			bold.Printf("FAIL: %d test(s) require attention\n\n", len(failing))

			report.PrintCI(failing, ctx.root)
			os.Exit(1)
			return nil
		},
	}
	cmd.Flags().StringVar(&reportMD, "report-md", "", "Save markdown report")
	cmd.Flags().StringVar(&reportJSON, "report-json", "", "Save JSON report")
	return cmd
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create .testmd.yaml in current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, name := range []string{".testmd.yaml", ".testmd.yml"} {
				if _, err := os.Stat(name); err == nil {
					return fmt.Errorf("%s already exists", name)
				}
			}
			if err := os.WriteFile(".testmd.yaml", []byte("ignorefile: .gitignore\n"), 0644); err != nil {
				return err
			}
			fmt.Println("Created .testmd.yaml")
			return nil
		},
	}
}

// --- root / config discovery ---

func findRoot(rootFlag string) (string, error) {
	if rootFlag != "" {
		abs, err := filepath.Abs(rootFlag)
		if err != nil {
			return "", err
		}
		if hasConfig(abs) {
			return abs, nil
		}
		return "", fmt.Errorf("no .testmd.yaml or .testmd.yml in %s", abs)
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		if hasConfig(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .testmd.yaml found (searched from cwd to filesystem root)")
		}
		dir = parent
	}
}

func hasConfig(dir string) bool {
	for _, name := range []string{".testmd.yaml", ".testmd.yml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func loadConfig(root string) (models.Config, error) {
	for _, name := range []string{".testmd.yaml", ".testmd.yml"} {
		path := filepath.Join(root, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return parser.ParseConfig(data)
	}
	return models.Config{Ignorefile: ".gitignore"}, nil
}

// --- TEST.md discovery ---

func discoverTestFiles(root string, ig *ignore.GitIgnore) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && rel != "." {
				return filepath.SkipDir
			}
			if ig != nil && rel != "." && ig.MatchesPath(rel+"/") {
				return filepath.SkipDir
			}
			return nil
		}

		if info.Name() == "TEST.md" {
			if ig == nil || !ig.MatchesPath(rel) {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// --- load / save ---

func load(rootFlag string) (*context, error) {
	root, err := findRoot(rootFlag)
	if err != nil {
		return nil, err
	}

	cfg, err := loadConfig(root)
	if err != nil {
		return nil, err
	}

	ig := patterns.LoadIgnorefile(root, cfg.Ignorefile)

	testFiles, err := discoverTestFiles(root, ig)
	if err != nil {
		return nil, err
	}

	var allDefs []models.TestDefinition
	for _, tf := range testFiles {
		data, err := os.ReadFile(tf)
		if err != nil {
			return nil, err
		}
		defs, err := parser.Parse(string(data), tf)
		if err != nil {
			return nil, err
		}
		allDefs = append(allDefs, defs...)
	}

	instances, err := resolver.BuildInstances(root, allDefs, ig)
	if err != nil {
		return nil, err
	}

	st, err := state.Load(lockPath(root))
	if err != nil {
		return nil, err
	}

	return &context{
		root:      root,
		instances: instances,
		state:     st,
	}, nil
}

func lockPath(root string) string {
	return filepath.Join(root, ".testmd.lock")
}

func labelSuffix(labels map[string]string) string {
	s := report.FormatLabels(labels)
	if s == "" {
		return ""
	}
	return " (" + s + ")"
}

func init() {
	cobra.EnableCommandSorting = false
}
