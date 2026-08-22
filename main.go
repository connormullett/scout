package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/connormullett/scout/lib"
)

func main() {
	client := lib.CreateClient("api_key", "gemma4:e4b")
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// get user input
		fmt.Print(" > ")

		for scanner.Scan() {
			input := scanner.Text()
			if input == "exit" {
				fmt.Println("Exiting...")
				return
			}
			client.AddMessage(input)
			break
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		stop := false
		for !stop {
			// execute client with input
			response, err := client.SendMessage(ctx)
			if err != nil {
				log.Fatal(err)
			}
			for _, block := range response.Content {
				if block.Type == "text" {
					fmt.Println(block.Text)
				}
			}

			client.AddAssistantMessage(response)

			log.Println("Stop reason:", response.StopReason)
			switch response.StopReason {
			case anthropic.StopReasonToolUse:
				// get response and tool calls
				var toolUse anthropic.ContentBlockUnion
				for _, block := range response.Content {
					if block.Type == "tool_use" {
						toolUse = block
						break
					}
				}

				var toolParams map[string]any
				err = json.Unmarshal(toolUse.Input, &toolParams)
				if err != nil {
					log.Fatal(err)
				}

				clientResponse := client.CallTool(ctx, toolUse.ID, toolUse.Name, toolParams)
				if clientResponse != "" {
					fmt.Println(clientResponse)
				}

			case anthropic.StopReasonEndTurn:
				// get response and tool calls
				for _, block := range response.Content {
					if block.Type == "text" {
						fmt.Println(block.Text)
					}
				}
				stop = true

			default:
				log.Println("Unknown stop reason:", response.StopReason)
			}
		}
	}
}
