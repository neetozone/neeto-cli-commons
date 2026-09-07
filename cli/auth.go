package cli

import (
	"bufio"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) newLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate to " + a.Product.PrettyName + " via browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			subdomain, err := a.promptSubdomain(cmd)
			if err != nil {
				return err
			}

			creds, err := a.Auth.Login(subdomain)
			if err != nil {
				return err
			}

			a.PrintMessage(fmt.Sprintf("Authenticated as %s on %s.", creds.Email, a.hostFor(creds.Subdomain)))
			return nil
		},
	}
}

func (a *App) promptSubdomain(cmd *cobra.Command) (string, error) {
	if !a.Auth.RequiresSubdomain() {
		return "", nil
	}

	subdomain, _ := cmd.Flags().GetString("subdomain")
	if subdomain == "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Enter your %s subdomain (e.g., 'acme' for acme.%s): ",
			a.Product.PrettyName, a.Product.Domain)
		input, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
		subdomain = strings.TrimSpace(input)
	}
	if subdomain == "" {
		return "", fmt.Errorf("Subdomain is required.")
	}
	return subdomain, nil
}

func (a *App) newLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Sign out and clear saved credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.Auth.LoadStore()
			if err != nil {
				return err
			}
			if len(store.Credentials) == 0 {
				a.PrintMessage("Not authenticated.")
				return nil
			}

			if !a.Auth.RequiresSubdomain() {
				store.Credentials = nil
				if err := a.Auth.SaveStore(store); err != nil {
					return err
				}
				a.PrintMessage("Signed out.")
				return nil
			}

			if all, _ := cmd.Flags().GetBool("all"); all {
				store.Credentials = nil
				if err := a.Auth.SaveStore(store); err != nil {
					return err
				}
				a.PrintMessage("Signed out of all subdomains.")
				return nil
			}

			subdomain, _ := cmd.Flags().GetString("subdomain")
			if subdomain == "" {
				if len(store.Credentials) > 1 {
					return fmt.Errorf("Multiple subdomains authenticated (%s); specify --subdomain or --all.",
						strings.Join(store.Subdomains(), ", "))
				}
				subdomain = store.Credentials[0].Subdomain
			}

			if !store.Remove(subdomain) {
				return fmt.Errorf("Not authenticated for %q.", subdomain)
			}
			if err := a.Auth.SaveStore(store); err != nil {
				return err
			}

			a.PrintMessage(fmt.Sprintf("Signed out of %s.", a.hostFor(subdomain)))
			return nil
		},
	}

	if a.Auth.RequiresSubdomain() {
		cmd.Flags().Bool("all", false, "Sign out of every saved subdomain")
	}
	return cmd
}

func (a *App) newWhoamiCommand() *cobra.Command {
	short := "Show current authenticated user(s)"
	if !a.Auth.RequiresSubdomain() {
		short = "Show current authenticated user"
	}

	return &cobra.Command{
		Use:   "whoami",
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.Auth.LoadStore()
			if err != nil {
				return err
			}
			if len(store.Credentials) == 0 {
				return fmt.Errorf("Not authenticated. Run '%s login' to authenticate.", a.Product.BinaryName)
			}

			if !a.Auth.RequiresSubdomain() {
				c := store.Credentials[0]
				a.PrintMessage(fmt.Sprintf("Authenticated as %s on %s.", c.Email, a.hostFor(c.Subdomain)))
				return nil
			}

			subdomain, _ := cmd.Flags().GetString("subdomain")
			if subdomain != "" {
				creds, ok := store.Find(subdomain)
				if !ok {
					return fmt.Errorf("Not authenticated for %q. Authenticated subdomains: %s.",
						subdomain, strings.Join(store.Subdomains(), ", "))
				}
				a.PrintMessage(fmt.Sprintf("Authenticated as %s on %s.", creds.Email, a.hostFor(creds.Subdomain)))
				return nil
			}

			if len(store.Credentials) == 1 {
				c := store.Credentials[0]
				a.PrintMessage(fmt.Sprintf("Authenticated as %s on %s (default).", c.Email, a.hostFor(c.Subdomain)))
				return nil
			}

			lines := make([]string, 0, len(store.Credentials))
			for _, c := range store.Credentials {
				lines = append(lines, fmt.Sprintf("%s on %s", c.Email, a.hostFor(c.Subdomain)))
			}
			a.PrintMessage(strings.Join(lines, "\n"))
			return nil
		},
	}
}

// hostFor returns the host a subdomain's requests go to, so messages name the
// workspace the user is actually talking to rather than the canonical domain.
func (a *App) hostFor(subdomain string) string {
	baseURL := a.Auth.BaseURL(subdomain)
	if u, err := url.Parse(baseURL); err == nil && u.Host != "" {
		return u.Host
	}
	if a.Auth.RequiresSubdomain() {
		return subdomain + "." + a.Product.Domain
	}
	return baseURL
}
