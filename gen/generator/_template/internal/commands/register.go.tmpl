package commands

import (
	"encoding/json/jsontext"
	"net/url"

	"github.com/neetozone/neeto-cli-commons/v2/cli"
	"github.com/neetozone/neeto-cli-commons/v2/client"
	"github.com/neetozone/neeto-cli-commons/v2/output"
	"github.com/spf13/cobra"
)

var (
	app           *cli.App
	registrations []func(root *cobra.Command)
)

func register(fn func(root *cobra.Command)) {
	registrations = append(registrations, fn)
}

func Register(a *cli.App) {
	app = a
	for _, fn := range registrations {
		fn(a.Root())
	}
}

func getClient(cmd *cobra.Command) (*client.Client, error) { return app.Client(cmd) }

func printList(data jsontext.Value, resourceKey string, breadcrumbs []output.Breadcrumb) {
	app.PrintList(data, resourceKey, breadcrumbs)
}

func printResource(data jsontext.Value, breadcrumbs []output.Breadcrumb) {
	app.PrintResource(data, breadcrumbs)
}

func paginationParams(cmd *cobra.Command) url.Values { return app.PaginationParams(cmd) }

func addPaginationFlags(cmd *cobra.Command) { cli.AddPaginationFlags(0, cmd) }
