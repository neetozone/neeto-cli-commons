package questionnaire

import (
	"fmt"
	"os"

	"github.com/neetozone/neeto-cli-commons/gen/vars"
	"gopkg.in/yaml.v3"
)

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
