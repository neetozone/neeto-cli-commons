package render

import (
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/config"
)

func TestCommonsRawBase_RefKinds(t *testing.T) {
	cases := map[string]string{
		"":       "main",
		"main":   "main",
		"1.2.0":  "v1.2.0",
		"v1.2.0": "v1.2.0",
	}
	for in, want := range cases {
		got := commonsRawBase(config.Product{CommonsVersion: in})
		if !strings.HasSuffix(got, "/"+want) {
			t.Errorf("CommonsVersion %q -> %q, want it to end in /%s", in, got, want)
		}
	}
}
