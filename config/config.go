package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	KeyToken     = "dev_token"
	KeyEmail     = "email"
	KeyURL       = "url"
	KeySystemKey = "system_key"
)

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".edge-cli")
}

func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// Load initialises viper with the CLI config file and env var bindings.
// Call once from root.go PersistentPreRun.
func Load() {
	viper.SetConfigFile(ConfigFile())
	viper.SetEnvPrefix("CB")
	viper.BindEnv(KeyToken, "CB_DEV_TOKEN")
	viper.BindEnv(KeySystemKey, "CB_SYSTEM_KEY")
	viper.BindEnv(KeyURL, "CB_EDGE_URL")
	viper.SetDefault(KeyURL, "http://localhost:9000")
	_ = viper.ReadInConfig()
}

func SaveCredentials(token, email string) error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return err
	}
	viper.Set(KeyToken, token)
	viper.Set(KeyEmail, email)
	return viper.WriteConfigAs(ConfigFile())
}

func ClearCredentials() error {
	viper.Set(KeyToken, "")
	viper.Set(KeyEmail, "")
	return viper.WriteConfigAs(ConfigFile())
}

func Token() string     { return viper.GetString(KeyToken) }
func Email() string     { return viper.GetString(KeyEmail) }
func URL() string       { return viper.GetString(KeyURL) }
func SystemKey() string { return viper.GetString(KeySystemKey) }
