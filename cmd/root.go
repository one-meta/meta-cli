package cmd

import (
	"github.com/one-meta/meta-cli/util"
	"github.com/spf13/cobra"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "meta-cli",
	Short: "CLI for create meta backend and frontend project",
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "new meta backend and frontend project",
	Run: func(cmd *cobra.Command, args []string) {
		util.NewProject()
	},
}
var generateCmd = &cobra.Command{
	Use:   "renew",
	Short: "re generate project config ",
	Run: func(cmd *cobra.Command, args []string) {
		util.ReNewProject()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(generateCmd)
}
