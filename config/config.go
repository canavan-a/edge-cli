package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	KeyToken      = "dev_token"
	KeyEmail      = "email"
	KeyURL        = "url"
	KeySystemKey  = "system_key"
	KeyProxyURL   = "proxy_url"
	KeyEdgeName   = "edge_name"
	KeyEdgeTokens = "edge_tokens"
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
	viper.BindEnv(KeyProxyURL, "CB_PROXY_URL")
	viper.BindEnv(KeyEdgeName, "CB_EDGE_NAME")
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

func SaveProxyConfig(proxyURL, edgeName string) error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return err
	}
	viper.Set(KeyProxyURL, proxyURL)
	viper.Set(KeyEdgeName, edgeName)
	return viper.WriteConfigAs(ConfigFile())
}

func ClearCredentials() error {
	viper.Set(KeyToken, "")
	viper.Set(KeyEmail, "")
	viper.Set(KeyProxyURL, "")
	viper.Set(KeyEdgeName, "")
	return viper.WriteConfigAs(ConfigFile())
}

func Token() string     { return viper.GetString(KeyToken) }
func Email() string     { return viper.GetString(KeyEmail) }
func URL() string       { return viper.GetString(KeyURL) }
func SystemKey() string { return viper.GetString(KeySystemKey) }
func ProxyURL() string  { return viper.GetString(KeyProxyURL) }
func EdgeName() string  { return viper.GetString(KeyEdgeName) }

// IsProxyMode returns true when a proxy URL and edge name are configured.
func IsProxyMode() bool { return ProxyURL() != "" && EdgeName() != "" }

// EdgeToken returns the cached dev token for a named edge, or empty string.
func EdgeToken(edgeName string) string {
	tokens := viper.GetStringMapString(KeyEdgeTokens)
	return tokens[edgeName]
}

// SaveEdgeToken caches a dev token for a named edge.
func SaveEdgeToken(edgeName, token string) error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return err
	}
	tokens := viper.GetStringMapString(KeyEdgeTokens)
	if tokens == nil {
		tokens = map[string]string{}
	}
	tokens[edgeName] = token
	viper.Set(KeyEdgeTokens, tokens)
	return viper.WriteConfigAs(ConfigFile())
}

// ClearEdgeToken removes the cached token for a named edge.
func ClearEdgeToken(edgeName string) error {
	tokens := viper.GetStringMapString(KeyEdgeTokens)
	delete(tokens, edgeName)
	viper.Set(KeyEdgeTokens, tokens)
	return viper.WriteConfigAs(ConfigFile())
}
