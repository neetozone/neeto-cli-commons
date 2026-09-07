package generator

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/neetozone/neeto-cli-commons/gen/vars"
)

// renderPath translates an embedFS-relative path to its output location
// for the given Variables. It replaces "__BINARY__" segments with the
// binary name and strips a single trailing ".tmpl" suffix.
func renderPath(p string, v *vars.Variables) string {
	parts := strings.Split(p, "/")
	for i, seg := range parts {
		parts[i] = strings.ReplaceAll(seg, "__BINARY__", v.BinaryName)
	}
	last := len(parts) - 1
	parts[last] = strings.TrimSuffix(parts[last], ".tmpl")
	return filepath.Join(parts...)
}

// renderFile reads embedPath from templateFS, optionally runs it through
// text/template if it has a ".tmpl" suffix, and writes to dst. Parent
// directories must exist.
func renderFile(embedPath, dst string, v *vars.Variables) error {
	data, err := fs.ReadFile(templateFS, embedPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", embedPath, err)
	}

	if !strings.HasSuffix(embedPath, ".tmpl") {
		return os.WriteFile(dst, data, 0o644)
	}

	tmpl, err := template.New(filepath.Base(embedPath)).
		Option("missingkey=error").
		Parse(string(data))
	if err != nil {
		return fmt.Errorf("parse %s: %w", embedPath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, v); err != nil {
		return fmt.Errorf("execute %s: %w", embedPath, err)
	}

	return os.WriteFile(dst, buf.Bytes(), 0o644)
}
