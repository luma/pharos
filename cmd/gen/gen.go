package gen

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "gen",
	Short: "Several useful generators",
	Long:  `Several useful generators`,
}

func init() {
	RootCmd.AddCommand(ManPagesCmd)
}
