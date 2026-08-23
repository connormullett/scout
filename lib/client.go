package lib

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	// ANSI escapes used to visually separate thinking from the final answer.
	dim   = "\033[2;3m"
	reset = "\033[0m"
)

type ScoutClient struct {
	Client   anthropic.Client
	Messages []anthropic.MessageParam
	Model    anthropic.Model
	Config   *Config
	Out      io.Writer
	Cwd      string
	Session  *ChatSession
}

func CreateClient(config *Config) *ScoutClient {
	client := anthropic.NewClient(option.WithAPIKey(config.APIKey), option.WithBaseURL(config.BaseURL))

	// Sessions are scoped to the directory scout was started in so that
	// `/load` only offers conversations from this project.
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	return &ScoutClient{
		Client:   client,
		Model:    anthropic.Model(config.Model),
		Messages: []anthropic.MessageParam{},
		Config:   config,
		Out:      os.Stdout,
		Cwd:      cwd,
		Session:  NewChatSession(cwd, time.Now()),
	}
}

func (sc *ScoutClient) AddMessage(content string) {
	// The first thing the user typed becomes the session's hint, which is
	// what `/load` shows to identify the session.
	if sc.Session.Hint == "" {
		sc.Session.Hint = content
	}

	sc.Messages = append(sc.Messages, anthropic.NewUserMessage(anthropic.NewTextBlock(content)))
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

	sc.Messages = append(sc.Messages, anthropic.NewAssistantMessage(assistantContent...))
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

// SaveSession persists the conversation. sc.Messages is the single source of
// truth for history -- it is the slice actually sent to the API, and it is
// the only one that carries tool results -- so the session is synced from it
// at save time rather than being appended to in parallel.
func (sc *ScoutClient) SaveSession() error {
	sc.Session.History = sc.Messages
	return sc.Session.Save()
}

// ResumeSession switches the client to a previously saved conversation,
// replacing the in-memory history. Subsequent saves write back to the
// resumed session's file rather than starting a new one.
func (sc *ScoutClient) ResumeSession(session *ChatSession) {
	sc.Session = session
	sc.Messages = session.History
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
