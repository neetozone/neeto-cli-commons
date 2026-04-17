package questionnaire

import (
	"fmt"
	"os"

	"github.com/neetozone/neeto-cli-template/internal/vars"
	"gopkg.in/yaml.v3"
)

// LoadConfig reads a YAML answers file, applies any missing defaults from
// PrettyName/BinaryName/etc, and validates the result. Callers get a fully
// populated Variables ready for generation.
func LoadConfig(path, templateVersion string) (*vars.Variables, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	var v vars.Variables
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("invalid YAML in %s: %w", path, err)
	}

	v.TemplateVersion = templateVersion
	v.ApplyDerivations()

	if err := v.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return &v, nil
}
