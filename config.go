package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	configFileName    = "config.json"
	placeholderAPIKey = "your-key-here"
)

type config struct {
	OpenRouter struct {
		APIKey string `json:"api_key"`
		Model  string `json:"model"`
	} `json:"openrouter"`
}

func defaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to find user config directory: %w", err)
	}
	return filepath.Join(configDir, "commiter", configFileName), nil
}

// loadConfig reads the config file. An empty configPath selects the
// platform's default user config path; a missing default config yields an
// empty config, while a missing explicit config is an error.
func loadConfig(configPath string) (cfg config, path string, err error) {
	explicitPath := configPath != ""
	if !explicitPath {
		configPath, err = defaultConfigPath()
		if err != nil {
			return config{}, "", err
		}
	}

	cfg, err = readConfig(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicitPath {
			return config{}, configPath, nil
		}
		return config{}, configPath, err
	}
	return cfg, configPath, nil
}

// resolveOpenRouterAPIKey reads the config before checking the environment.
// An empty configPath selects the platform's default user config path.
func resolveOpenRouterAPIKey(configPath string) (string, error) {
	cfg, path, err := loadConfig(configPath)
	if err != nil {
		return "", err
	}
	return resolveAPIKey(cfg, path)
}

func resolveAPIKey(cfg config, configPath string) (string, error) {
	if apiKey := cfg.OpenRouter.APIKey; apiKey != "" && apiKey != placeholderAPIKey {
		return resolveAPIKeyValue(apiKey)
	}

	if apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")); apiKey != "" {
		return resolveAPIKeyValue(apiKey)
	}

	return "", fmt.Errorf("OpenRouter API key not found in %s or OPENROUTER_API_KEY", configPath)
}

// resolveModel prefers an explicit flag value over the config. There is no
// built-in default; the installer writes one into new configs.
func resolveModel(flagModel string, cfg config, configPath string) (string, error) {
	if model := strings.TrimSpace(flagModel); model != "" {
		return model, nil
	}
	if cfg.OpenRouter.Model != "" {
		return cfg.OpenRouter.Model, nil
	}
	return "", fmt.Errorf("OpenRouter model not set; add openrouter.model to %s or pass --model", configPath)
}

func resolveAPIKeyValue(value string) (string, error) {
	if !strings.HasPrefix(value, "op://") {
		return value, nil
	}

	return readOnePasswordSecret(value)
}

func readOnePasswordSecret(reference string) (string, error) {
	opPath, err := exec.LookPath("op")
	if err != nil {
		return "", errors.New("cannot resolve 1Password reference: 1Password CLI (op) not found in PATH")
	}

	// Pass the reference as one argument without a shell. Config contents can
	// therefore never be interpreted as shell syntax.
	cmd := exec.Command(opPath, "read", "--no-newline", reference)
	cmd.Stdin = os.Stdin
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return "", fmt.Errorf("failed to read OpenRouter API key from 1Password: %s", detail)
		}
		return "", fmt.Errorf("failed to read OpenRouter API key from 1Password: %w", err)
	}

	secret := strings.TrimSpace(string(output))
	if secret == "" {
		return "", errors.New("1Password returned an empty OpenRouter API key")
	}
	return secret, nil
}

func readConfig(path string) (config, error) {
	file, err := os.Open(path)
	if err != nil {
		return config{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return config{}, fmt.Errorf("failed to inspect config %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return config{}, fmt.Errorf("config %s is not a regular file", path)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return config{}, fmt.Errorf("config %s permissions are too open (%04o); run: chmod 600 %q", path, info.Mode().Perm(), path)
	}

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var cfg config
	if err := decoder.Decode(&cfg); err != nil {
		return config{}, fmt.Errorf("failed to parse config %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return config{}, fmt.Errorf("failed to parse config %s: multiple JSON values", path)
		}
		return config{}, fmt.Errorf("failed to parse config %s: %w", path, err)
	}

	cfg.OpenRouter.APIKey = strings.TrimSpace(cfg.OpenRouter.APIKey)
	cfg.OpenRouter.Model = strings.TrimSpace(cfg.OpenRouter.Model)
	return cfg, nil
}
