package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/luma/pharos/internal/meta"
)

var (
	manDir string
)

var ManPagesCmd = &cobra.Command{
	Use:   "man",
	Short: "Generate man pages for the Pharos api service",
	Long: `This command automatically generates up-to-date man pages of the
	Blinsight processor.  By default, it creates the man page files
	in the "man" directory under the current directory.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		header := &doc.GenManHeader{
			Section: "1",
			Manual:  "Pharos API service Manual",
			Source:  fmt.Sprintf("pharos %s", meta.Version),
		}

		if !strings.HasSuffix(manDir, string(filepath.Separator)) {
			manDir += string(filepath.Separator)
		}

		if _, err := os.Stat(manDir); err != nil && os.IsNotExist(err) {
			fmt.Println("Directory", manDir, "does not exist, creating...")
			if err := os.MkdirAll(manDir, 0750); err != nil {
				return err
			}
		}

		cmd.Root().DisableAutoGenTag = true

		fmt.Println("Generating Pharos API service man pages in", manDir, "...")

		if err := doc.GenManTree(cmd.Root(), header, manDir); err != nil {
			return err
		}

		fmt.Println("Done.")

		return nil
	},
}

func init() {
	flags := ManPagesCmd.PersistentFlags()

	flags.StringVar(&manDir, "dir", "man/", "the directory to write the man pages.")

	// For bash-completion
	if err := flags.SetAnnotation("dir", cobra.BashCompSubdirsInDir, []string{}); err != nil {
		panic(err)
	}
}
