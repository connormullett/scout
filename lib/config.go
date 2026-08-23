package lib

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey       string `mapstructure:"api_key"`
	Model        string
	MaxTokens    int    `mapstructure:"max_tokens"`
	BaseURL      string `mapstructure:"base_url"`
	ShowThinking bool   `mapstructure:"show_thinking"`
}

func SetConfigDefaults() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME/.scout")

	viper.SetDefault("base_url", "http://localhost:11434")
	viper.SetDefault("max_tokens", 10000)
	viper.SetDefault("model", "gemma4:e4b")
	viper.SetDefault("show_thinking", true)
}

func LoadConfig() (*Config, error) {
	SetConfigDefaults()
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			if writeErr := viper.SafeWriteConfig(); writeErr != nil {
				return nil, fmt.Errorf("failed to create default config file: %v", writeErr)
			}
		}
		return nil, err
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func WriteConfig(config *Config) error {
	viper.Set("api_key", config.APIKey)
	viper.Set("model", config.Model)
	viper.Set("max_tokens", config.MaxTokens)
	viper.Set("base_url", config.BaseURL)
	viper.Set("show_thinking", config.ShowThinking)

	err := viper.WriteConfig()
	if err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			if writeErr := viper.SafeWriteConfig(); writeErr != nil {
				return fmt.Errorf("failed to create config file: %v", writeErr)
			}
		} else {
			return fmt.Errorf("failed to write config file: %v", err)
		}
	}

	return nil
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if c.Model == "" {
		return fmt.Errorf("Model is required")
	}
	if c.MaxTokens <= 0 {
		return fmt.Errorf("Max tokens must be greater than 0")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("Base URL is required")
	}
	return nil
}

func EnsureConfigDirExists() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	configDir := fmt.Sprintf("%s/.scout", homeDir)
	info, err := os.Stat(configDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("config directory does not exist: %s", configDir)
	}
	if err != nil {
		return fmt.Errorf("failed to stat config directory: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("config path is not a directory: %s", configDir)
	}

	return nil
}

func InitConfigDir() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	configDir := fmt.Sprintf("%s/.scout", homeDir)
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	skillsDir := fmt.Sprintf("%s/skills", configDir)
	err = os.MkdirAll(skillsDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create skills directory: %v", err)
	}

	sessionsDir := fmt.Sprintf("%s/sessions", configDir)
	err = os.MkdirAll(sessionsDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create sessions directory: %v", err)
	}

	return nil
}
