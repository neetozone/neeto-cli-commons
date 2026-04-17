// Package generator orchestrates rendering the embedded CLI template into
// a brand-new target directory. It never modifies an existing directory.
package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/neetozone/neeto-cli-template/internal/questionnaire"
	"github.com/neetozone/neeto-cli-template/internal/vars"
)

// Options controls a single Run invocation.
type Options struct {
	OutputDir       string
	ConfigPath      string
	NonInteractive  bool
	SkipTidy        bool
	SkipGitInit     bool
	TemplateVersion string
}

// Run executes the full generation pipeline.
func Run(opts Options) error {
	v, err := loadVariables(opts)
	if err != nil {
		return err
	}

	target, err := resolveTarget(opts, v)
	if err != nil {
		return err
	}

	if err := writeTree(target, v); err != nil {
		return err
	}

	if err := restorePermissions(target, v); err != nil {
		return fmt.Errorf("chmod exec files: %w", err)
	}

	if err := os.WriteFile(
		filepath.Join(target, ".template-version"),
		[]byte(v.TemplateVersion+"\n"),
		0o644,
	); err != nil {
		return fmt.Errorf("write .template-version: %w", err)
	}

	if v.RunTidy && !opts.SkipTidy {
		if err := runGoModTidy(target); err != nil {
			fmt.Fprintf(os.Stderr, "warning: go mod tidy failed: %v\n", err)
		}
	}

	if v.RunGitInit && !opts.SkipGitInit {
		if err := runGitInit(target, v.TemplateVersion); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git init failed: %v\n", err)
		}
	}

	printNextSteps(target, v)
	return nil
}

func loadVariables(opts Options) (*vars.Variables, error) {
	if opts.ConfigPath != "" {
		return questionnaire.LoadConfig(opts.ConfigPath, opts.TemplateVersion)
	}
	if opts.NonInteractive {
		return nil, fmt.Errorf("--non-interactive requires --config")
	}
	return questionnaire.Ask(opts.TemplateVersion)
}

func resolveTarget(opts Options, v *vars.Variables) (string, error) {
	target := opts.OutputDir
	if target == "" {
		target = v.RepoName
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(abs)
	switch {
	case os.IsNotExist(err):
		// Will be created below.
	case err != nil:
		return "", err
	case !info.IsDir():
		return "", fmt.Errorf("%s exists and is not a directory", abs)
	default:
		entries, err := os.ReadDir(abs)
		if err != nil {
			return "", err
		}
		if len(entries) > 0 {
			return "", fmt.Errorf("%s is not empty; refusing to generate", abs)
		}
	}

	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	return abs, nil
}

func writeTree(target string, v *vars.Variables) error {
	return fs.WalkDir(templateFS, "_template", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "_template" {
			return nil
		}
		rel := strings.TrimPrefix(p, "_template/")

		// The exec manifest is a generator-side artifact; do not copy.
		if rel == ".exec-manifest" {
			return nil
		}

		renderedRel := renderPath(rel, v)
		dst := filepath.Join(target, renderedRel)

		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return renderFile(p, dst, v)
	})
}

func printNextSteps(target string, v *vars.Variables) {
	fmt.Printf("\nGenerated %s at %s\n\n", v.RepoName, target)
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", target)
	if !v.RunTidy {
		fmt.Println("  go mod tidy")
	}
	fmt.Printf("  make build\n")
	fmt.Printf("  ./%s --help\n", v.BinaryName)
	fmt.Println()
	fmt.Println("Add product-specific commands per docs/adding-commands.md.")
}
