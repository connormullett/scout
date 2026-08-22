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

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
			return // stdin closed
		}

		input := scanner.Text()
		if input == "exit" {
			fmt.Println("Exiting...")
			return
		}
		client.AddMessage(input)

		stop := false
		for !stop {
			// the response streams to stdout as it is generated
			response, err := client.SendMessage(ctx)
			if err != nil {
				log.Fatal(err)
			}

			client.AddAssistantMessage(response)

			log.Println("Stop reason:", response.StopReason)
			switch response.StopReason {
			case anthropic.StopReasonToolUse:
				// run every tool call in this turn, then hand all the results
				// back in a single user message
				var results []anthropic.ContentBlockParamUnion
				for _, block := range response.Content {
					if block.Type != "tool_use" {
						continue
					}

					var toolParams map[string]any
					if err := json.Unmarshal(block.Input, &toolParams); err != nil {
						log.Fatal(err)
					}

					results = append(results, client.ExecuteTool(block.ID, block.Name, toolParams))
				}
				client.AddToolResults(results)

			case anthropic.StopReasonEndTurn:
				stop = true

			default:
				log.Println("Unknown stop reason:", response.StopReason)
				stop = true
			}
		}
	}
}
