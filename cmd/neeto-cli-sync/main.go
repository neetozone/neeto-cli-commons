// Command neeto-cli-sync writes the files neeto-cli-commons owns into a
// product repository, or verifies that what is on disk already matches.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neetozone/neeto-cli-commons/config"
	"github.com/neetozone/neeto-cli-commons/render"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("neeto-cli-sync", flag.ContinueOnError)
	configPath := flags.String("config", ".neeto-cli.yml", "Path to the product config")
	check := flags.Bool("check", false, "Exit non-zero when anything on disk differs from what would be written")
	out := flags.String("out", "", "Write into this directory instead of the repository")
	env := flags.Bool("env", false, "Print the product's release values as shell assignments")
	if err := flags.Parse(args); err != nil {
		return err
	}

	product, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	if *env {
		fmt.Fprint(stdout, render.ShellEnv(*product))
		return nil
	}

	files, err := render.All(*product)
	if err != nil {
		return err
	}

	dir := *out
	if dir == "" {
		dir = "."
	}

	if *check {
		return checkTree(*product, dir, files, *out == "", stdout)
	}
	return writeTree(*product, dir, files, *out == "", stdout)
}

func writeTree(product config.Product, dir string, files map[string][]byte, mergeReadme bool, stdout io.Writer) error {
	for _, name := range sortedKeys(files) {
		dst := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		mode := modeFor(name)
		if err := os.WriteFile(dst, files[name], mode); err != nil {
			return err
		}
		if err := os.Chmod(dst, mode); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "wrote "+name)
	}

	if !mergeReadme {
		return nil
	}

	path := filepath.Join(dir, "README.md")
	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		fmt.Fprintln(stdout, "skipped README.md: no README.md in "+dir)
		return nil
	}
	if err != nil {
		return err
	}
	merged, err := render.MergeREADME(product, existing)
	if err != nil {
		return err
	}
	if missing := render.MissingSections(existing); len(missing) > 0 {
		fmt.Fprintln(stdout, "README.md has no markers for: "+strings.Join(missing, ", "))
	}
	if bytes.Equal(merged, existing) {
		return nil
	}
	if err := os.WriteFile(path, merged, 0o644); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "wrote README.md")
	return nil
}

func checkTree(product config.Product, dir string, files map[string][]byte, mergeReadme bool, stdout io.Writer) error {
	var stale []string
	for _, name := range sortedKeys(files) {
		dst := filepath.Join(dir, filepath.FromSlash(name))
		onDisk, err := os.ReadFile(dst)
		if os.IsNotExist(err) {
			stale = append(stale, name+" (missing)")
			continue
		}
		if err != nil {
			return err
		}
		if !bytes.Equal(onDisk, files[name]) {
			stale = append(stale, name+" (out of date)")
			continue
		}
		info, err := os.Stat(dst)
		if err != nil {
			return err
		}
		if want := modeFor(name); info.Mode().Perm()&0o111 != want&0o111 {
			stale = append(stale, fmt.Sprintf("%s (mode %o, want %o)", name, info.Mode().Perm(), want))
		}
	}

	if mergeReadme {
		path := filepath.Join(dir, "README.md")
		existing, err := os.ReadFile(path)
		switch {
		case os.IsNotExist(err):
		case err != nil:
			return err
		default:
			merged, err := render.MergeREADME(product, existing)
			if err != nil {
				return err
			}
			if !bytes.Equal(merged, existing) {
				stale = append(stale, "README.md (generated sections out of date)")
			}
		}
	}

	if len(stale) == 0 {
		fmt.Fprintln(stdout, "neeto-cli-sync: everything is up to date.")
		return nil
	}
	return fmt.Errorf("neeto-cli-sync --check failed; run neeto-cli-sync to update:\n  %s",
		strings.Join(stale, "\n  "))
}

func modeFor(name string) fs.FileMode {
	if render.IsExecutable(name) {
		return 0o755
	}
	return 0o644
}

func sortedKeys(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
