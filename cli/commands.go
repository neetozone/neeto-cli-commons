package cli

func (a *App) registerCommonCommands() {
	registerTemplateFuncs()
	a.root.SetUsageTemplate(usageTemplate)
	if a.Product.DocsURL != "" {
		if a.root.Annotations == nil {
			a.root.Annotations = map[string]string{}
		}
		a.root.Annotations[docsURLAnnotation] = a.Product.DocsURL
	}
	a.root.CompletionOptions.DisableDefaultCmd = true

	a.root.AddCommand(
		a.newLoginCommand(),
		a.newLogoutCommand(),
		a.newWhoamiCommand(),
		a.newVersionCommand(),
		a.newDoctorCommand(),
		a.newUpdateCommand(),
		a.newCompletionCommand(),
		a.newSetupCommand(),
		a.newCommandsCatalogCommand(),
	)
}
