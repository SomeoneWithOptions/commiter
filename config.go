package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const configFileName = "config.json"

type config struct {
	OpenRouter struct {
		APIKey string `json:"api_key"`
	} `json:"openrouter"`
}

func defaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to find user config directory: %w", err)
	}
	return filepath.Join(configDir, "commiter", configFileName), nil
}

// resolveOpenRouterAPIKey reads the config before checking the environment.
// An empty configPath selects the platform's default user config path.
func resolveOpenRouterAPIKey(configPath string) (string, error) {
	explicitPath := configPath != ""
	if !explicitPath {
		var err error
		configPath, err = defaultConfigPath()
		if err != nil {
			return "", err
		}
	}

	apiKey, found, err := readConfigAPIKey(configPath)
	if err != nil {
		if !(errors.Is(err, os.ErrNotExist) && !explicitPath) {
			return "", err
		}
	} else if found {
		return apiKey, nil
	}

	if apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")); apiKey != "" {
		return apiKey, nil
	}

	return "", fmt.Errorf("OpenRouter API key not found in %s or OPENROUTER_API_KEY", configPath)
}

func readConfigAPIKey(path string) (apiKey string, found bool, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", false, fmt.Errorf("failed to inspect config %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("config %s is not a regular file", path)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return "", false, fmt.Errorf("config %s permissions are too open (%04o); run: chmod 600 %q", path, info.Mode().Perm(), path)
	}

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var cfg config
	if err := decoder.Decode(&cfg); err != nil {
		return "", false, fmt.Errorf("failed to parse config %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return "", false, fmt.Errorf("failed to parse config %s: multiple JSON values", path)
		}
		return "", false, fmt.Errorf("failed to parse config %s: %w", path, err)
	}

	apiKey = strings.TrimSpace(cfg.OpenRouter.APIKey)
	return apiKey, apiKey != "", nil
}
