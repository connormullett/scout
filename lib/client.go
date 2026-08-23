package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
)

const (
	// ANSI escapes used to visually separate thinking from the final answer.
	dim   = "\033[2;3m"
	reset = "\033[0m"
)

type ChatSession struct {
	SessionID string
	History   []anthropic.MessageParam
	// todo: add mutex for concurrency safety
}

type ScoutClient struct {
	Client   anthropic.Client
	Messages []anthropic.MessageParam
	Model    anthropic.Model
	Config   *Config
	Out      io.Writer
	Session  *ChatSession
}

func CreateClient(config *Config) *ScoutClient {
	client := anthropic.NewClient(option.WithAPIKey(config.APIKey), option.WithBaseURL(config.BaseURL))
	return &ScoutClient{
		Client:   client,
		Model:    anthropic.Model(config.Model),
		Messages: []anthropic.MessageParam{},
		Config:   config,
		Out:      os.Stdout,
		Session: &ChatSession{
			SessionID: uuid.New().String(),
			History:   []anthropic.MessageParam{},
		},
	}
}

func (sc *ScoutClient) AddMessage(content string) {
	message := anthropic.NewUserMessage(anthropic.NewTextBlock(content))
	sc.Session.History = append(sc.Session.History, message)
	sc.Messages = append(sc.Messages, message)
}

func (sc *ScoutClient) AddAssistantMessage(response *anthropic.Message) {
	var assistantContent []anthropic.ContentBlockParamUnion
	for _, block := range response.Content {
		assistantContent = append(assistantContent, block.ToParam())
	}

	// An empty assistant turn is rejected by the API, so don't record one.
	if len(assistantContent) == 0 {
		return
	}

	message := anthropic.NewAssistantMessage(assistantContent...)
	sc.Messages = append(sc.Messages, message)
	sc.Session.History = append(sc.Session.History, message)
}

func (sc *ScoutClient) params() anthropic.MessageNewParams {
	return anthropic.MessageNewParams{
		Model:      sc.Model,
		Messages:   sc.Messages,
		Tools:      Tools,
		ToolChoice: ToolChoice,
		MaxTokens:  int64(sc.Config.MaxTokens),
		System: []anthropic.TextBlockParam{
			{Text: "you are scout, a helpful assistant that can run tools on the user's behalf"},
		},
		Thinking: anthropic.ThinkingConfigParamUnion{
			OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{
				// Without this, thinking blocks stream with empty text.
				Display: anthropic.ThinkingConfigAdaptiveDisplaySummarized,
			},
		},
	}
}

func (sc *ScoutClient) SaveSession() error {
	sessionDir := fmt.Sprintf("%s/sessions", os.Getenv("HOME")+"/.scout")
	sessionFile := fmt.Sprintf("%s/%s.json", sessionDir, sc.Session.SessionID)

	file, err := os.Create(sessionFile)
	if err != nil {
		return fmt.Errorf("failed to create session file: %v", err)
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(sc.Session)
	if err != nil {
		return fmt.Errorf("failed to encode session to file: %v", err)
	}

	return nil
}

// SendMessage streams the next assistant turn, writing thinking, text and tool
// calls to sc.Out as they arrive, and returns the accumulated message.
func (sc *ScoutClient) SendMessage(ctx context.Context) (*anthropic.Message, error) {
	stream := sc.Client.Messages.NewStreaming(ctx, sc.params())

	message := anthropic.Message{}
	thinking := false
	showThinking := sc.Config.ShowThinking

	for stream.Next() {
		event := stream.Current()
		if err := message.Accumulate(event); err != nil {
			return nil, fmt.Errorf("accumulating stream event: %w", err)
		}

		switch event := event.AsAny().(type) {
		case anthropic.ContentBlockStartEvent:
			switch event.ContentBlock.Type {
			case "thinking":
				thinking = true
				if showThinking {
					fmt.Fprintf(sc.Out, "%s[thinking] ", dim)
				}
			case "redacted_thinking":
				if showThinking {
					fmt.Fprintf(sc.Out, "%s[thinking redacted]%s\n", dim, reset)
				}
			case "tool_use":
				fmt.Fprintf(sc.Out, "[tool: %s]\n", event.ContentBlock.Name)
			}

		case anthropic.ContentBlockDeltaEvent:
			switch delta := event.Delta.AsAny().(type) {
			case anthropic.ThinkingDelta:
				if showThinking {
					fmt.Fprint(sc.Out, delta.Thinking)
				}
			case anthropic.TextDelta:
				fmt.Fprint(sc.Out, delta.Text)
			}

		case anthropic.ContentBlockStopEvent:
			if thinking {
				if showThinking {
					fmt.Fprint(sc.Out, reset)
					fmt.Fprintln(sc.Out)
				}
				thinking = false
			} else {
				fmt.Fprintln(sc.Out)
			}
		}
	}

	sc.Session.History = append(sc.Session.History, message.ToParam())

	if err := stream.Err(); err != nil {
		return nil, err
	}

	return &message, nil
}

// ExecuteTool runs a single tool call and returns the tool_result block for it.
func (sc *ScoutClient) ExecuteTool(toolID, toolName string, params any) anthropic.ContentBlockParamUnion {

	toolFunc, exists := ToolMappings[toolName]
	if !exists {
		return anthropic.NewToolResultBlock(toolID, fmt.Sprintf("error: unknown tool %q", toolName), true)
	}

	toolResult := toolFunc(params)

	return anthropic.NewToolResultBlock(toolID, toolResult, false)
}

// AddToolResults records every result from one assistant turn as a single user
// message; splitting them across messages discourages parallel tool use.
func (sc *ScoutClient) AddToolResults(results []anthropic.ContentBlockParamUnion) {
	if len(results) == 0 {
		return
	}
	sc.Messages = append(sc.Messages, anthropic.NewUserMessage(results...))
}
