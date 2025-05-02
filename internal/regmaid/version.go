package regmaid

import (
	"fmt"

	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed VERSION
var Version string

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Long:  "Print the version of regmaid",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}
