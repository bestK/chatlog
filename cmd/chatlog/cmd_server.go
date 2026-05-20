package chatlog

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/sjzar/chatlog/internal/chatlog"
	"github.com/sjzar/chatlog/internal/chatlog/conf"
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.PersistentPreRun = initLog
	serverCmd.PersistentFlags().BoolVar(&Debug, "debug", false, "debug")
	serverCmd.Flags().StringVarP(&serverAddr, "addr", "a", "", "server address")
	serverCmd.Flags().StringVarP(&serverPlatform, "platform", "p", "", "platform")
	serverCmd.Flags().IntVarP(&serverVer, "version", "v", 0, "version")
	serverCmd.Flags().StringVarP(&serverDataDir, "data-dir", "d", "", "data dir")
	serverCmd.Flags().StringVarP(&serverDataKey, "data-key", "k", "", "data key")
	serverCmd.Flags().StringVarP(&serverImgKey, "img-key", "i", "", "img key")
	serverCmd.Flags().StringVarP(&serverWorkDir, "work-dir", "w", "", "work dir")
	serverCmd.Flags().BoolVarP(&serverAutoDecrypt, "auto-decrypt", "", true, "auto decrypt")
}

var (
	serverAddr        string
	serverDataDir     string
	serverDataKey     string
	serverImgKey      string
	serverWorkDir     string
	serverPlatform    string
	serverVer         int
	serverAutoDecrypt bool
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		overrides := getServerOverrides()
		log.Info().Msgf("server overrides: %+v", overrides)

		m := chatlog.New()
		if err := m.CommandHTTPServer("", overrides); err != nil {
			log.Err(err).Msg("failed to start server")
			return
		}
	},
}

func getServerOverrides() conf.ServerOverrides {
	overrides := conf.ServerOverrides{}
	if len(serverAddr) != 0 {
		overrides.HTTPAddr = conf.StringOverride(serverAddr)
	}
	if len(serverDataDir) != 0 {
		overrides.DataDir = conf.StringOverride(serverDataDir)
	}
	if len(serverDataKey) != 0 {
		overrides.DataKey = conf.StringOverride(serverDataKey)
	}
	if len(serverImgKey) != 0 {
		overrides.ImgKey = conf.StringOverride(serverImgKey)
	}
	if len(serverWorkDir) != 0 {
		overrides.WorkDir = conf.StringOverride(serverWorkDir)
	}
	if len(serverPlatform) != 0 {
		overrides.Platform = conf.StringOverride(serverPlatform)
	}
	overrides.AutoDecrypt = conf.BoolOverride(serverAutoDecrypt)
	if Debug {
		overrides.Debug = conf.BoolOverride(true)
	}
	return overrides
}
