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
	Short: "Enforce tag retention policies on Docker registries",
	Long: `
Regmaid is a CLI tool that deletes image tags in Docker registries based on retention policies.
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return ExecuteClean(cmd.Context())
	},
}

func init() {
	cobra.OnInitialize()

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().StringVarP(&ConfigPath, "config", "c", "regmaid.yaml", "Path to the config file")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&Yes, "yes", "y", false, "Auto confirm cleanup")
	rootCmd.PersistentFlags().BoolVarP(&DryRun, "dry-run", "", false, "Dry run (only list tags eligible for deletion)")
}

func Execute() error {
	return rootCmd.Execute()
}
