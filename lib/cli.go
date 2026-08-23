package lib

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

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
	apiKey := Scan(scanner, "Enter your API key: ")
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	model := Scan(scanner, "Enter the model to use (e.g., 'gemma4:e4b'): ")
	if model == "" {
		model = viper.GetString("model")
	}

	maxTokensStr := Scan(scanner, "Enter the maximum number of tokens to use (default 10000): ")
	if maxTokensStr == "" {
		maxTokensStr = strconv.Itoa(viper.GetInt("max_tokens"))
	}
	maxTokens, err := strconv.Atoi(maxTokensStr)
	if err != nil {
		return fmt.Errorf("invalid max tokens value: %v", err)
	}

	baseURL := Scan(scanner, "Enter the base URL for the API (default http://localhost:11434): ")
	if baseURL == "" {
		baseURL = viper.GetString("base_url")
	}

	showThinkingStr := Scan(scanner, "Show model thinking? (Y/n): ")
	showThinking := viper.GetBool("show_thinking")
	if showThinkingStr != "" {
		showThinking, err = strconv.ParseBool(normalizeYesNo(showThinkingStr))
		if err != nil {
			return fmt.Errorf("invalid value for show thinking: %v", err)
		}
	}

	config = &Config{
		APIKey:       apiKey,
		Model:        model,
		MaxTokens:    maxTokens,
		BaseURL:      baseURL,
		ShowThinking: showThinking,
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	return WriteConfig(config)
}

// normalizeYesNo maps common yes/no shorthand to values strconv.ParseBool
// understands, so prompts like "Y/n" behave as expected.
func normalizeYesNo(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes":
		return "true"
	case "n", "no":
		return "false"
	default:
		return s
	}
}

func Scan(scanner *bufio.Scanner, prompt string) string {
	fmt.Fprintf(os.Stdout, "%s", prompt)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "Error reading input")
		os.Exit(1)
	}
	return scanner.Text()
}
