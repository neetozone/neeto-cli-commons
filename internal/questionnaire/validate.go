package questionnaire

import (
	"fmt"
	"regexp"
)

var (
	prettyRE         = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9 \-]{1,39}$`)
	binaryRE         = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)
	orgRE            = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
	repoRE           = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
	moduleRE         = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)
	domainRE         = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`)
	envVarRE         = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	tapRE            = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)
	goVersRE         = regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)
	emailRE          = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	apiBaseRE        = regexp.MustCompile(`^/.+`)
	reservedBinaries = map[string]struct{}{"go": {}, "test": {}, "help": {}, "completion": {}}
)

func validatePretty(s string) error {
	if !prettyRE.MatchString(s) {
		return fmt.Errorf("2-40 chars; letters/digits/spaces/hyphens; must start with a letter")
	}
	return nil
}

func validateBinary(s string) error {
	if !binaryRE.MatchString(s) {
		return fmt.Errorf("lowercase letters/digits/hyphens; 2-32 chars; must start with a letter")
	}
	if _, reserved := reservedBinaries[s]; reserved {
		return fmt.Errorf("%q is a reserved name", s)
	}
	return nil
}

func validateOrg(s string) error {
	if !orgRE.MatchString(s) {
		return fmt.Errorf("invalid GitHub org")
	}
	return nil
}

func validateRepo(s string) error {
	if !repoRE.MatchString(s) {
		return fmt.Errorf("invalid GitHub repo")
	}
	return nil
}

func validateModule(s string) error {
	if !moduleRE.MatchString(s) {
		return fmt.Errorf("invalid Go module path")
	}
	return nil
}

func validateDomain(s string) error {
	if !domainRE.MatchString(s) {
		return fmt.Errorf("invalid host name")
	}
	return nil
}

func validateAPIBase(s string) error {
	if !apiBaseRE.MatchString(s) {
		return fmt.Errorf("must start with / and be non-empty")
	}
	return nil
}

func validateEnvVar(s string) error {
	if !envVarRE.MatchString(s) {
		return fmt.Errorf("UPPER_SNAKE_CASE, must start with a letter")
	}
	return nil
}

func validateTap(s string) error {
	if !tapRE.MatchString(s) {
		return fmt.Errorf("owner/repo format required")
	}
	return nil
}

func validateGoVersion(s string) error {
	if !goVersRE.MatchString(s) {
		return fmt.Errorf("expected X.Y or X.Y.Z")
	}
	return nil
}

func validateEmail(s string) error {
	if !emailRE.MatchString(s) {
		return fmt.Errorf("invalid email")
	}
	return nil
}

func validateNonEmpty(field string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
}

func validateShortDesc(s string) error {
	if n := len(s); n < 5 || n > 120 {
		return fmt.Errorf("5-120 chars required, got %d", n)
	}
	return nil
}

func validateLongDesc(s string) error {
	if n := len(s); n < 10 || n > 240 {
		return fmt.Errorf("10-240 chars required, got %d", n)
	}
	return nil
}
