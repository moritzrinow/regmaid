package regmaid

import (
	"github.com/spf13/cobra"
)

var (
	ConfigPath string
	Verbose    bool
)

const defaultConfigPath = "regmaid.yaml"

var rootCmd = &cobra.Command{
	Use:   "regmaid",
	Short: "regmaid keeps your OCI registry clean",
	Long: `
regmaid is a CLI tool that helps you keep your OCI registry clean.
Delete images based on configured retention policies.
	`,
}

func init() {
	cobra.OnInitialize()

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().StringVarP(&ConfigPath, "config", "c", "regmaid.yaml", "Path to the config file")

	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable verbose output")
}

func Execute() error {
	return rootCmd.Execute()
}
