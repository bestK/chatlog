package chatlog

import (
	"fmt"

	"github.com/sjzar/chatlog/internal/chatlog"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(decryptCmd)
	decryptCmd.Flags().StringVarP(&dataDir, "data-dir", "d", "", "data dir")
	decryptCmd.Flags().StringVarP(&workDir, "work-dir", "w", "", "work dir")
	decryptCmd.Flags().StringVarP(&key, "key", "k", "", "key")
	decryptCmd.Flags().IntVarP(&decryptVer, "version", "v", 3, "version")
}

var dataDir string
var workDir string
var key string
var decryptVer int

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "decrypt",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := chatlog.New("")
		if err != nil {
			log.Error(err)
			return
		}
		if err := m.CommandDecrypt(dataDir, workDir, key, decryptVer); err != nil {
			log.Error(err)
			return
		}
		fmt.Println("decrypt success")
	},
}
