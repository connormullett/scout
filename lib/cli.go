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

// maxHintWidth caps how much of a session's first prompt is shown in the
// `/load` picker, so one long prompt can't wrap and break the column layout.
const maxHintWidth = 72

// SelectSession renders the saved sessions for cwd as a numbered list and
// returns the one the user picks. It returns nil when there is nothing to
// load or the user cancels, which the caller treats as a no-op rather than
// an error.
func SelectSession(scanner *bufio.Scanner, cwd string) (*ChatSession, error) {
	summaries, err := ListSessions(cwd)
	if err != nil {
		return nil, err
	}

	if len(summaries) == 0 {
		fmt.Printf("no saved sessions for %s\n", cwd)
		return nil, nil
	}

	fmt.Printf("\nsessions for %s:\n\n", cwd)
	for i, summary := range summaries {
		fmt.Printf("  %2d  %s  %3d msgs  %s\n",
			i+1,
			summary.CreatedAt.Format("2006-01-02 15:04"),
			summary.Messages,
			formatHint(summary.Hint),
		)
	}
	fmt.Println()

	for {
		choice := strings.TrimSpace(Scan(scanner, fmt.Sprintf("select a session (1-%d, blank to cancel): ", len(summaries))))
		if choice == "" {
			return nil, nil
		}

		n, err := strconv.Atoi(choice)
		if err != nil || n < 1 || n > len(summaries) {
			fmt.Printf("enter a number between 1 and %d, or leave blank to cancel\n", len(summaries))
			continue
		}

		return LoadSession(summaries[n-1].SessionID)
	}
}

// formatHint makes a session's first prompt safe to print on one line.
func formatHint(hint string) string {
	hint = strings.Join(strings.Fields(hint), " ")
	if hint == "" {
		return "(no prompt recorded)"
	}

	// Count runes, not bytes, so a multi-byte prompt isn't cut mid-character.
	if runes := []rune(hint); len(runes) > maxHintWidth {
		return string(runes[:maxHintWidth-1]) + "\u2026"
	}
	return hint
}
