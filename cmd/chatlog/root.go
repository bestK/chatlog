package chatlog

import (
	"strings"

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

func IsCLIInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}

	knownCommands := map[string]struct{}{
		"server":     {},
		"key":        {},
		"decrypt":    {},
		"dumpmemory": {},
		"version":    {},
		"help":       {},
		"completion": {},
	}

	first := strings.TrimSpace(args[0])
	if first == "" {
		return false
	}
	if _, ok := knownCommands[first]; ok {
		return true
	}
	return strings.HasPrefix(first, "-")
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
