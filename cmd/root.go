package cmd

import (
	"fmt"
	"os"

	"edge-cli/config"

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
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Simple line scan: look for ParentSystemKey = "..."
	// Avoids pulling in a TOML dependency just for one field.
	import_buf := make([]byte, 4096)
	n, _ := f.Read(import_buf)
	content := string(import_buf[:n])

	for _, line := range splitLines(content) {
		key, val, ok := parseKV(line)
		if ok && key == "ParentSystemKey" {
			return val, nil
		}
	}
	return "", fmt.Errorf("ParentSystemKey not found in %s", path)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// parseKV parses a line of the form: Key = "value" or Key = value
func parseKV(line string) (key, val string, ok bool) {
	for i := 0; i < len(line); i++ {
		if line[i] == '=' {
			key = trimSpace(line[:i])
			rest := trimSpace(line[i+1:])
			if len(rest) >= 2 && rest[0] == '"' && rest[len(rest)-1] == '"' {
				val = rest[1 : len(rest)-1]
			} else {
				val = rest
			}
			return key, val, true
		}
	}
	return "", "", false
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
