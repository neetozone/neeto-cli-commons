package generator

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/neetozone/neeto-cli-commons/gen/vars"
)

// restorePermissions reads template/.exec-manifest, resolves each entry
// through renderPath, and sets 0755 on the corresponding output file. The
// manifest itself is not copied to output.
func restorePermissions(target string, v *vars.Variables) error {
	f, err := templateFS.Open("_template/.exec-manifest")
	if err != nil {
		// Not having a manifest is fine; there just aren't any executables.
		if isNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rendered := renderPath(line, v)
		if err := os.Chmod(filepath.Join(target, rendered), 0o755); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
	}
	return scanner.Err()
}

func isNotExist(err error) bool {
	if pe, ok := err.(*fs.PathError); ok {
		return os.IsNotExist(pe.Err)
	}
	return os.IsNotExist(err)
}
