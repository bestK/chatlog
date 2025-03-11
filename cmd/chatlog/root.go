package chatlog

import (
	"github.com/sjzar/chatlog/internal/chatlog"

	log "github.com/sirupsen/logrus"
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
		log.Error(err)
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
	PreRun: initTuiLog,
	Run:    Root,
}

func Root(cmd *cobra.Command, args []string) {

	m, err := chatlog.New("")
	if err != nil {
		log.Error(err)
		return
	}

	if err := m.Run(); err != nil {
		log.Error(err)
	}
}
