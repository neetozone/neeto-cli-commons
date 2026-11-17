package cli

import (
	"errors"
	"net/url"

	"github.com/neetozone/neeto-cli-commons/v2/client"
	"github.com/neetozone/neeto-cli-commons/v2/output"
	"github.com/spf13/cobra"
)

// CommonsAPIBasePath is where neeto-commons-backend serves the endpoints every
// product shares, whatever the product's own api_base_path is.
const CommonsAPIBasePath = "/api/external/v2"

// AddRoleCommands adds the `roles` and `permissions` commands, backed by the
// role and permission endpoints neeto-commons-backend serves to every product.
// It is opt-in: call it only when the product's backend serves the commons
// external API. Call it after the product adds its own commands; a command the
// product already defines under either name is kept and the shared one skipped.
func (a *App) AddRoleCommands() {
	for _, cmd := range []*cobra.Command{a.newRolesCommand(), a.newPermissionsCommand()} {
		if !a.hasCommand(cmd.Name()) {
			a.root.AddCommand(cmd)
		}
	}
}

func (a *App) hasCommand(name string) bool {
	for _, cmd := range a.root.Commands() {
		if cmd.Name() == name {
			return true
		}
	}
	return false
}

func (a *App) commonsClient(cmd *cobra.Command) (*client.Client, error) {
	c, err := a.Client(cmd)
	if err != nil {
		return nil, err
	}
	c.BaseURL = c.Host + CommonsAPIBasePath
	return c, nil
}

func (a *App) newPermissionsCommand() *cobra.Command {
	permissions := &cobra.Command{
		Use:   "permissions",
		Short: "View the permissions a role can grant",
	}

	permissions.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List every permission a role can grant",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.commonsClient(cmd)
			if err != nil {
				return err
			}
			data, err := c.Get("/permissions", nil)
			if err != nil {
				return err
			}
			a.PrintList(data, "permissions", []output.Breadcrumb{
				{Label: "Create role", Command: a.Product.BinaryName + " roles create --name <name> --permission-ids <id,id>"},
			})
			return nil
		},
	})

	return permissions
}

func (a *App) newRolesCommand() *cobra.Command {
	roles := &cobra.Command{
		Use:   "roles",
		Short: "Manage the roles team members can have",
	}

	roles.AddCommand(
		a.newRolesListCommand(),
		a.newRolesCreateCommand(),
		a.newRolesUpdateCommand(),
		a.newRolesDeleteCommand(),
	)

	return roles
}

func (a *App) newRolesListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List roles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.commonsClient(cmd)
			if err != nil {
				return err
			}
			data, err := c.Get("/roles", nil)
			if err != nil {
				return err
			}
			a.PrintList(data, "roles", []output.Breadcrumb{
				{Label: "Update", Command: a.Product.BinaryName + " roles update <id>"},
				{Label: "Delete", Command: a.Product.BinaryName + " roles delete <id>"},
				{Label: "Permissions", Command: a.Product.BinaryName + " permissions list"},
			})
			return nil
		},
	}
}

func (a *App) newRolesCreateCommand() *cobra.Command {
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a role",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.commonsClient(cmd)
			if err != nil {
				return err
			}
			data, err := c.Post("/roles", roleBody(cmd))
			if err != nil {
				return err
			}
			a.PrintActionResult(data, []output.Breadcrumb{
				{Label: "List", Command: a.Product.BinaryName + " roles list"},
			})
			return nil
		},
	}
	addRoleFlags(create)
	MarkFlagsRequired(create, "name")
	return create
}

func (a *App) newRolesUpdateCommand() *cobra.Command {
	update := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a role's name, description or permissions",
		Long: "Update a role. Only the flags you pass change. --permission-ids replaces the role's\n" +
			"permissions; pass --permission-ids \"\" to remove them all. The Admin role cannot be changed.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := roleBody(cmd)
			if len(body) == 0 {
				return errors.New("Nothing to update. Pass --name, --description or --permission-ids.")
			}
			c, err := a.commonsClient(cmd)
			if err != nil {
				return err
			}
			data, err := c.Patch("/roles/"+url.PathEscape(args[0]), body)
			if err != nil {
				return err
			}
			a.PrintActionResult(data, []output.Breadcrumb{
				{Label: "List", Command: a.Product.BinaryName + " roles list"},
			})
			return nil
		},
	}
	addRoleFlags(update)
	return update
}

func (a *App) newRolesDeleteCommand() *cobra.Command {
	del := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a role",
		Long: "Delete a role. If team members have it, pass --new-role-id to move them to another\n" +
			"role first; the move and the delete happen together or not at all.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.commonsClient(cmd)
			if err != nil {
				return err
			}
			params := url.Values{}
			if newRoleID, _ := cmd.Flags().GetString("new-role-id"); newRoleID != "" {
				params.Set("new_role_id", newRoleID)
			}
			if err := c.DeleteWithParams("/roles/"+url.PathEscape(args[0]), params); err != nil {
				return err
			}
			a.PrintMessage("Role deleted.")
			return nil
		},
	}
	del.Flags().String("new-role-id", "", "Role to move the deleted role's team members to")
	return del
}

func addRoleFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "Role name")
	cmd.Flags().String("description", "", "Role description")
	cmd.Flags().String("permission-ids", "", "Comma-separated permission IDs (see 'permissions list')")
}

// roleBody sends only the flags the user passed, so an update leaves the rest
// of the role alone and an empty --permission-ids clears its permissions.
func roleBody(cmd *cobra.Command) map[string]any {
	body := map[string]any{}
	for _, name := range []string{"name", "description"} {
		if cmd.Flags().Changed(name) {
			body[name], _ = cmd.Flags().GetString(name)
		}
	}
	if cmd.Flags().Changed("permission-ids") {
		ids, _ := cmd.Flags().GetString("permission-ids")
		body["permission_ids"] = SplitCSV(ids)
	}
	return body
}
