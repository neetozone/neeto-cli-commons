package cli

func (a *App) registerCommonCommands() {
	registerTemplateFuncs()
	a.root.SetUsageTemplate(usageTemplate)
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
