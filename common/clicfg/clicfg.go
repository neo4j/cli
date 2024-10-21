package clicfg

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/neo4j/cli/common/clicfg/creds"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var ConfigPrefix string

const DefaultAuraBaseUrl = "https://api.neo4j.io/v1"
const DefaultAuraAuthUrl = "https://api.neo4j.io/oauth/token"

var ValidOutputValues = [3]string{"default", "json", "table"}

func NewConfig(fs afero.Fs, version string) (*Config, error) {
	configPath := filepath.Join(ConfigPrefix, "neo4j", "cli")

	Viper := viper.New()

	Viper.SetFs(fs)
	Viper.SetConfigName("config")
	Viper.SetConfigType("json")
	Viper.AddConfigPath(configPath)
	Viper.SetConfigPermissions(0600)

	bindEnvironmentVariables(Viper)
	setDefaultValues(Viper)

	if err := Viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := fs.MkdirAll(configPath, 0755); err != nil {
				return nil, err
			}
			if err = Viper.SafeWriteConfig(); err != nil {
				return nil, err
			}
		} else {
			// Config file was found but another error was produced
			return nil, err
		}
	}

	credentials, err := creds.NewCredentials(fs, ConfigPrefix)
	if err != nil {
		return nil, err
	}

	return &Config{
		Version: version,
		Aura: AuraConfig{
			viper: Viper, pollingOverride: PollingConfig{
				MaxRetries: 60,
				Interval:   20,
			},
			ValidConfigKeys: []string{"auth-url", "base-url", "default-tenant", "output"},
		},
		Credentials: credentials,
	}, nil
}

func bindEnvironmentVariables(Viper *viper.Viper) {
	Viper.BindEnv("aura.base-url", "AURA_BASE_URL")
	Viper.BindEnv("aura.auth-url", "AURA_AUTH_URL")
}

func setDefaultValues(Viper *viper.Viper) {
	Viper.SetDefault("aura.base-url", DefaultAuraBaseUrl)
	Viper.SetDefault("aura.auth-url", DefaultAuraAuthUrl)
	Viper.SetDefault("aura.output", "default")
}

type Config struct {
	Version     string
	Aura        AuraConfig
	Credentials *creds.Credentials
}

type PollingConfig struct {
	Interval   int
	MaxRetries int
}

type AuraConfig struct {
	viper           *viper.Viper
	pollingOverride PollingConfig
	ValidConfigKeys []string
}

func (config *AuraConfig) IsValidConfigKey(key string) bool {
	return slices.Contains(config.ValidConfigKeys, key)
}

func (config *AuraConfig) Get(key string) interface{} {
	return config.viper.Get(fmt.Sprintf("aura.%s", key))
}

func (config *AuraConfig) Set(key string, value string) error {
	config.viper.Set(fmt.Sprintf("aura.%s", key), value)
	return config.viper.WriteConfig()
}

func (config *AuraConfig) Print(cmd *cobra.Command) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "\t")

	if err := encoder.Encode(config.viper.Get("aura")); err != nil {
		return err
	}

	return nil
}

func (config *AuraConfig) BaseUrl() string {
	return config.viper.GetString("aura.base-url")
}

func (config *AuraConfig) BindBaseUrl(flag *pflag.Flag) error {
	return config.viper.BindPFlag("aura.base-url", flag)
}

func (config *AuraConfig) AuthUrl() string {
	return config.viper.GetString("aura.auth-url")
}

func (config *AuraConfig) BindAuthUrl(flag *pflag.Flag) error {
	return config.viper.BindPFlag("aura.auth-url", flag)
}

func (config *AuraConfig) Output() string {
	return config.viper.GetString("aura.output")
}

func (config *AuraConfig) BindOutput(flag *pflag.Flag) error {
	return config.viper.BindPFlag("aura.output", flag)
}

func (config *AuraConfig) DefaultTenant() string {
	return config.viper.GetString("aura.default-tenant")
}

func (config *AuraConfig) PollingConfig() PollingConfig {
	return config.pollingOverride
}

func (config *AuraConfig) SetPollingConfig(maxRetries int, interval int) {
	config.pollingOverride = PollingConfig{
		MaxRetries: maxRetries,
		Interval:   interval,
	}
}
