package lib

import (
	"context"
	"log"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	USER      = "user"
	ASSISTANT = "assistant"
)

type ScoutClient struct {
	Client   anthropic.Client
	Messages []anthropic.MessageParam
	Model    anthropic.Model
}

func CreateClient(apiKey, model string) *ScoutClient {
	client := anthropic.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL("http://localhost:11434"))
	return &ScoutClient{
		Client:   client,
		Model:    model,
		Messages: []anthropic.MessageParam{},
	}
}

func (sc *ScoutClient) AddMessage(content string) {
	message := anthropic.NewUserMessage(anthropic.NewTextBlock(content))
	sc.Messages = append(sc.Messages, message)
}

func (sc *ScoutClient) AddAssistantMessage(response *anthropic.Message) {
	var assistantContent []anthropic.ContentBlockParamUnion
	for _, block := range response.Content {
		log.Println("Adding assistant message block:", block.Type, "with content:", block.Text)
		assistantContent = append(assistantContent, block.ToParam())
	}

	sc.Messages = append(sc.Messages, anthropic.NewAssistantMessage(assistantContent...))
}

func (sc *ScoutClient) SendMessage(ctx context.Context) (*anthropic.Message, error) {
	log.Println("Sending message with", len(sc.Messages), "messages to model", sc.Model)
	response, err := sc.Client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:      sc.Model,
		Messages:   sc.Messages,
		Tools:      Tools,
		ToolChoice: ToolChoice,
		MaxTokens:  1024,
	})
	return response, err
}

func (sc *ScoutClient) CallTool(ctx context.Context, toolID, toolName string, params any) string {
	log.Println("Calling tool:", toolName, "with params:", params)
	toolFunc, exists := ToolMappings[toolName]
	if !exists {
		log.Fatalf("Tool %s not found in ToolMappings", toolName)
	}
	toolResult := toolFunc(params)

	log.Println("Tool", toolName, "returned result:", toolResult)
	sc.Messages = append(sc.Messages, anthropic.NewUserMessage(anthropic.NewToolResultBlock(toolID, toolResult, false)))

	followup, err := sc.Client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:      sc.Model,
		MaxTokens:  1024,
		Tools:      Tools,
		ToolChoice: ToolChoice,
		Messages:   sc.Messages,
	})
	if err != nil {
		log.Fatal(err)
	}

	// display response and tool calls
	for _, block := range followup.Content {
		if block.Type == "text" {
			return block.Text
		}
	}

	return ""
}
