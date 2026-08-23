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
	if len(os.Args) > 1 {
		subcommand := os.Args[1]
		if subcommand == "init" {
			err := lib.InitCommand()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Configuration initialized successfully.")
			return
		}
	}

	err := lib.EnsureConfigDirExists()
	if err != nil {
		fmt.Println("No configuration found, run `scout init` to create a configuration file.")
	}

	config, err := lib.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := lib.CreateClient(config)
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
				stop = true
			}
		}
	}
}
