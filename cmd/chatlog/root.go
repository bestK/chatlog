package chatlog

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	// windows only
	cobra.MousetrapHelpText = ""

	rootCmd.PersistentFlags().BoolVar(&Debug, "debug", false, "debug")
	rootCmd.PersistentPreRun = initLog
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Err(err).Msg("command execution failed")
	}
}

var rootCmd = &cobra.Command{
	Use:     "chatlog",
	Short:   "chatlog",
	Long:    `chatlog`,
	Example: `chatlog`,
	Args:    cobra.MinimumNArgs(0),
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	PreRun: initLog,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
