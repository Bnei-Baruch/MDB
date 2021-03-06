package cmd

import (
	"github.com/spf13/cobra"

	"github.com/Bnei-Baruch/mdb/importer/cleanup"
)

func init() {
	command := &cobra.Command{
		Use:   "cleanup-analyze",
		Short: "Analyze content units to be clean",
		Run: func(cmd *cobra.Command, args []string) {
			cleanup.Analyze()
		},
	}
	RootCmd.AddCommand(command)

	command = &cobra.Command{
		Use:   "cleanup-import",
		Short: "Split clips from non clip content units",
		Run: func(cmd *cobra.Command, args []string) {
			cleanup.Import()
		},
	}
	RootCmd.AddCommand(command)
}
