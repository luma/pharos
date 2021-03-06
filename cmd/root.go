package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/luma/pharos/cmd/gen"
	"github.com/luma/pharos/internal/meta"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "pharos",
	Short:   "Pharos api service",
	Long:    `Pharos api service`,
	Version: meta.Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(gen.RootCmd)
	rootCmd.AddCommand(StartCmd)
}
