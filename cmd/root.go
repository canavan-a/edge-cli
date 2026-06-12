package cmd

import (
	"fmt"
	"os"

	"edge-cli/config"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "edge-cli",
	Short: "CLI for managing a ClearBlade Edge device",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.Load()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("token", "", "ClearBlade dev token (overrides config / CB_DEV_TOKEN)")
	rootCmd.PersistentFlags().String("system-key", "", "System key (overrides config / CB_SYSTEM_KEY)")
	rootCmd.PersistentFlags().String("url", "", "Edge URL (overrides config / CB_EDGE_URL)")
	rootCmd.PersistentFlags().String("edge-config", "", "Path to the edge TOML config file (to auto-detect system key)")

	viper.BindPFlag("dev_token", rootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("system_key", rootCmd.PersistentFlags().Lookup("system-key"))
	viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
}

// resolveSystemKey returns the system key from flags/env/config, or attempts
// to read it from the edge's own TOML config file as a fallback.
func resolveSystemKey(edgeConfigPath string) (string, error) {
	if sk := config.SystemKey(); sk != "" {
		return sk, nil
	}

	paths := []string{edgeConfigPath, "/etc/clearblade/config.toml", "./config.toml"}
	for _, p := range paths {
		if p == "" {
			continue
		}
		if sk, err := readParentSystemKeyFromTOML(p); err == nil && sk != "" {
			return sk, nil
		}
	}

	return "", fmt.Errorf("system key not found — set --system-key, CB_SYSTEM_KEY, or ensure the edge config is readable")
}

func readParentSystemKeyFromTOML(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var cfg map[string]any
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", path, err)
	}

	val, ok := cfg["ParentSystemKey"]
	if !ok {
		return "", fmt.Errorf("ParentSystemKey not found in %s", path)
	}

	sk, ok := val.(string)
	if !ok || sk == "" {
		return "", fmt.Errorf("ParentSystemKey is empty in %s", path)
	}

	return sk, nil
}
