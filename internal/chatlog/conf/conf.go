package conf

import (
	"encoding/json"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/pkg/config"
)

const (
	AppName          = "chatlog"
	ServerConfigName = "chatlog"
	EnvPrefix        = "CHATLOG"
	EnvConfigDir     = "CHATLOG_DIR"
)

func LoadAppConfig(configPath string) (*AppConfig, *config.Manager, error) {

	if configPath == "" {
		configPath = os.Getenv(EnvConfigDir)
	}

	tcm, err := config.New(AppName, configPath, "", EnvPrefix, true)
	if err != nil {
		log.Error().Err(err).Msg("load app config failed")
		return nil, nil, err
	}

	conf := &AppConfig{}
	config.SetDefaults(tcm.Viper, conf, AppDefaults)

	if err := tcm.Load(conf); err != nil {
		log.Error().Err(err).Msg("load app config failed")
		return nil, nil, err
	}
	conf.ConfigDir = tcm.Path

	b, _ := json.Marshal(conf)
	log.Info().Msgf("app config: %s", string(b))

	return conf, tcm, nil
}

// LoadServiceConfig 加载服务配置
func LoadServiceConfig(configPath string, cmdConf map[string]any) (*ServerConfig, *config.Manager, error) {

	if configPath == "" {
		configPath = os.Getenv(EnvConfigDir)
	}

	scm, err := config.New(AppName, configPath, ServerConfigName, EnvPrefix, false)
	if err != nil {
		log.Error().Err(err).Msg("load server config failed")
		return nil, nil, err
	}

	conf := &ServerConfig{}
	config.SetDefaults(scm.Viper, conf, ServerDefaults)

	// Load cmd Conf
	for key, value := range cmdConf {
		scm.SetConfig(key, value)
	}

	if err := scm.Load(conf); err != nil {
		log.Error().Err(err).Msg("load server config failed")
		return nil, nil, err
	}

	b, _ := json.Marshal(conf)
	log.Info().Msgf("server config: %s", string(b))

	return conf, scm, nil
}
