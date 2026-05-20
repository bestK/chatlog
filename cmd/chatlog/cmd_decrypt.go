package chatlog

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/sjzar/chatlog/internal/chatlog"
	"github.com/sjzar/chatlog/internal/chatlog/conf"
)

func init() {
	rootCmd.AddCommand(decryptCmd)
	decryptCmd.Flags().StringVarP(&decryptPlatform, "platform", "p", "", "platform")
	decryptCmd.Flags().IntVarP(&decryptVer, "version", "v", 0, "version")
	decryptCmd.Flags().StringVarP(&decryptDataDir, "data-dir", "d", "", "data dir")
	decryptCmd.Flags().StringVarP(&decryptDatakey, "data-key", "k", "", "data key")
	decryptCmd.Flags().StringVarP(&decryptWorkDir, "work-dir", "w", "", "work dir")
}

var (
	decryptPlatform string
	decryptVer      int
	decryptDataDir  string
	decryptDatakey  string
	decryptWorkDir  string
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "decrypt",
	Run: func(cmd *cobra.Command, args []string) {
		overrides := getDecryptOverrides()

		m := chatlog.New()
		if err := m.CommandDecrypt("", overrides); err != nil {
			log.Err(err).Msg("failed to decrypt")
			return
		}
		fmt.Println("decrypt success")
	},
}

func getDecryptOverrides() conf.ServerOverrides {
	overrides := conf.ServerOverrides{}
	if len(decryptDataDir) != 0 {
		overrides.DataDir = conf.StringOverride(decryptDataDir)
	}
	if len(decryptDatakey) != 0 {
		overrides.DataKey = conf.StringOverride(decryptDatakey)
	}
	if len(decryptWorkDir) != 0 {
		overrides.WorkDir = conf.StringOverride(decryptWorkDir)
	}
	if len(decryptPlatform) != 0 {
		overrides.Platform = conf.StringOverride(decryptPlatform)
	}
	return overrides
}
