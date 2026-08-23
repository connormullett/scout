package lib

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

func InitCommand() error {
	SetConfigDefaults()
	err := EnsureConfigDirExists()
	if err != nil {
		initErr := InitConfigDir()
		if initErr != nil {
			return initErr
		}
	}

	scanner := bufio.NewScanner(os.Stdin)

	var config *Config
	apiKey := Scan(scanner, "Enter your API key")
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	model := Scan(scanner, "Enter the model to use (e.g., 'gemma4:e4b')")
	if model == "" {
		model = viper.GetString("model")
	}

	maxTokensStr := Scan(scanner, "Enter the maximum number of tokens to use (default 10000)")
	if maxTokensStr == "" {
		maxTokensStr = strconv.Itoa(viper.GetInt("max_tokens"))
	}
	maxTokens, err := strconv.Atoi(maxTokensStr)
	if err != nil {
		return fmt.Errorf("invalid max tokens value: %v", err)
	}

	baseURL := Scan(scanner, "Enter the base URL for the API (default http://localhost:11434)")
	if baseURL == "" {
		baseURL = viper.GetString("base_url")
	}

	config = &Config{
		APIKey:    apiKey,
		Model:     model,
		MaxTokens: maxTokens,
		BaseURL:   baseURL,
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	return WriteConfig(config)
}

func Scan(scanner *bufio.Scanner, prompt string) string {
	fmt.Fprintf(os.Stdout, "%s: ", prompt)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "Error reading input")
		os.Exit(1)
	}
	return scanner.Text()
}
