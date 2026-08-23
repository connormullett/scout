package lib

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	mapstructure "github.com/mitchellh/mapstructure"
)

type WriteTextFileParams struct {
	FilePath  string `json:"file_path" mapstructure:"file_path"`
	Content   string `json:"content" mapstructure:"content"`
	Overwrite bool   `json:"overwrite" mapstructure:"overwrite"`
}

func WriteTextFile(params any) string {
	var oParams WriteTextFileParams
	err := mapstructure.Decode(params, &oParams)
	if err != nil {
		return fmt.Sprintf("error: failed to decode parameters: %v", err)
	}

	filePath := oParams.FilePath
	content := oParams.Content
	overwrite := oParams.Overwrite

	_, statErr := os.Stat(filePath)
	if statErr == nil && !overwrite {
		return "file already exists and overwrite is false"
	}

	if os.IsNotExist(statErr) {
		fmt.Println("creating file", filePath)
	}

	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return fmt.Sprintf("Success: file written successfully to %s", filePath)
}

var writeFileTool anthropic.ToolUnionParam = anthropic.ToolUnionParam{
	OfTool: &anthropic.ToolParam{
		Name:        "write_file",
		Description: anthropic.String("Writes content to a file. If the file already exists, it will be overwritten if the 'overwrite' parameter is set to true."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "The path to the file where the content will be written.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The content to write to the file.",
				},
				"overwrite": map[string]any{
					"type":        "boolean",
					"description": "If true, the file will be overwritten if it already exists. If false, an error will be returned if the file exists.",
				},
			},
			Required: []string{"file_path", "content", "overwrite"},
		},
	},
}

type InsertLineParams struct {
	FilePath   string `json:"file_path" mapstructure:"file_path"`
	LineNumber int    `json:"line_number" mapstructure:"line_number"`
	NewLine    string `json:"new_line" mapstructure:"new_line"`
}

func InsertLineToFile(params any) string {
	var oParams InsertLineParams
	err := mapstructure.Decode(params, &oParams)
	if err != nil {
		return fmt.Sprintf("error: failed to decode parameters: %v", err)
	}
	filePath := oParams.FilePath
	lineNumber := oParams.LineNumber
	newLine := oParams.NewLine

	// Read the existing file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Sprintf("error: failed to open file: %v", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("error: failed to scan file: %v", err)
	}

	// Adjust line number to 0-indexed bounds
	if lineNumber < 1 {
		lineNumber = 1
	}
	idx := min(lineNumber-1, len(lines))

	// Insert the new line using slice manipulation
	lines = append(lines, "") // Grow the slice capacity by 1
	copy(lines[idx+1:], lines[idx:])
	lines[idx] = newLine

	// Overwrite the file with the updated content
	output := strings.Join(lines, "\n") + "\n"
	err = os.WriteFile(filePath, []byte(output), 0644)
	if err != nil {
		return fmt.Sprintf("error: failed to write updated content: %v", err)
	}

	return "Success: line inserted successfully"
}

var insertLineTool anthropic.ToolUnionParam = anthropic.ToolUnionParam{
	OfTool: &anthropic.ToolParam{
		Name:        "insert_line_to_file",
		Description: anthropic.String("Inserts a line into a file at the specified line number."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "The path to the file where the line will be inserted.",
				},
				"line_number": map[string]any{
					"type":        "integer",
					"description": "The line number at which to insert the new line.",
				},
				"new_line": map[string]any{
					"type":        "string",
					"description": "The content of the new line to insert.",
				},
			},
			Required: []string{"file_path", "line_number", "new_line"},
		},
	},
}

type PatchFileParams struct {
	FilePath string `json:"file_path" mapstructure:"file_path"`
	Search   string `json:"search" mapstructure:"search"`
	Replace  string `json:"replace" mapstructure:"replace"`
}

func PatchFile(params any) string {
	var oParams PatchFileParams
	err := mapstructure.Decode(params, &oParams)
	if err != nil {
		return fmt.Sprintf("error: failed to decode parameters: %v", err)
	}
	filePath := oParams.FilePath
	search := oParams.Search
	replace := oParams.Replace

	// Read the existing file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Sprintf("error: failed to open file: %v", err)
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Sprintf("error: failed to read file: %v", err)
	}

	contentStr := bytes.ReplaceAll(content, []byte(search), []byte(replace))
	err = os.WriteFile(filePath, contentStr, 0644)
	if err != nil {
		return fmt.Sprintf("error: failed to write updated content: %v", err)
	}

	return "Success: file patched successfully"
}

var patchFileTool anthropic.ToolUnionParam = anthropic.ToolUnionParam{
	OfTool: &anthropic.ToolParam{
		Name:        "patch_file",
		Description: anthropic.String("Patches a file by replacing all occurrences of a search string with a replacement string."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "The path to the file to patch.",
				},
				"search": map[string]any{
					"type":        "string",
					"description": "The search string to replace.",
				},
				"replace": map[string]any{
					"type":        "string",
					"description": "The replacement string.",
				},
			},
			Required: []string{"file_path", "search", "replace"},
		},
	},
}

type ReadFileParams struct {
	FilePath string `json:"file_path" mapstructure:"file_path"`
}

func ReadFile(params any) string {
	var oParams ReadFileParams
	err := mapstructure.Decode(params, &oParams)
	if err != nil {
		return fmt.Sprintf("error: failed to decode parameters: %v", err)
	}

	file, err := os.Open(oParams.FilePath)
	if err != nil {
		return fmt.Sprintf("error: failed to open file: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Sprintf("error: failed to read file: %v", err)
	}
	return string(content)
}

var readFileTool anthropic.ToolUnionParam = anthropic.ToolUnionParam{
	OfTool: &anthropic.ToolParam{
		Name:        "read_file",
		Description: anthropic.String("Reads the content of a file and returns it as a string."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "The path to the file to read.",
				},
			},
			Required: []string{"file_path"},
		},
	},
}

type ShellCommandParams struct {
	Command string `json:"command" mapstructure:"command"`
}

func ExecuteShellCommand(params any) string {
	var oParams ShellCommandParams
	err := mapstructure.Decode(params, &oParams)
	if err != nil {
		return fmt.Sprintf("error: failed to decode parameters: %v", err)
	}

	command := strings.TrimSpace(oParams.Command)
	if command == "" {
		return "error: no command provided"
	}

	shell, flag := shellInterpreter()
	cmd := exec.Command(shell, flag, command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("error: failed to execute command: %v\noutput:\n%s", err, output)
	}

	return string(output)
}

// shellInterpreter returns the shell binary and the flag used to run a
// command string through it, so shell_command supports pipes, redirects,
// command separators, quoting and builtins the way a real shell would,
// rather than naively splitting on whitespace.
func shellInterpreter() (string, string) {
	if runtime.GOOS == "windows" {
		if shell := os.Getenv("COMSPEC"); shell != "" {
			return shell, "/C"
		}
		return "cmd", "/C"
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return shell, "-c"
}

var shellCommandTool anthropic.ToolUnionParam = anthropic.ToolUnionParam{
	OfTool: &anthropic.ToolParam{
		Name:        "shell_command",
		Description: anthropic.String("Executes a shell command via the user's shell (or /bin/sh) and returns the combined stdout/stderr output. Supports pipes, redirects, command separators (&&, ;, ||), quoting and builtins like cd, since the command is interpreted by a real shell rather than being split on whitespace."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The shell command to execute.",
				},
			},
			Required: []string{"command"},
		},
	},
}

var Tools []anthropic.ToolUnionParam = []anthropic.ToolUnionParam{
	writeFileTool,
	insertLineTool,
	patchFileTool,
	readFileTool,
	shellCommandTool,
}

var ToolMappings map[string]func(params any) string = map[string]func(params any) string{
	"write_file":          WriteTextFile,
	"insert_line_to_file": InsertLineToFile,
	"patch_file":          PatchFile,
	"read_file":           ReadFile,
	"shell_command":       ExecuteShellCommand,
}

var ToolChoice anthropic.ToolChoiceUnionParam = anthropic.ToolChoiceUnionParam{
	OfAuto: &anthropic.ToolChoiceAutoParam{
		DisableParallelToolUse: anthropic.Bool(true),
	},
}
