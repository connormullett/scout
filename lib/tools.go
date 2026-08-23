package lib

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

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

// defaultShellCommandTimeout bounds how long a single shell_command
// invocation is allowed to run before it is forcibly killed. This prevents
// long-running/blocking commands (dev servers, `tail -f`, interactive
// prompts, backgrounded processes, etc.) from hanging the tool call forever.
// It can be overridden via the SCOUT_SHELL_TIMEOUT_SECONDS env var.
const defaultShellCommandTimeout = 2 * time.Minute

// shellCommandWaitDelay is passed through to exec.Cmd.WaitDelay. Once the
// context is cancelled/times out and the process (group) has been signalled,
// this is the grace period before Go forcibly closes the stdout/stderr pipes
// even if some descendant process is still holding them open. Without this,
// an orphaned child that inherited the pipes can make Wait() block forever
// even after the main process has been killed.
const shellCommandWaitDelay = 5 * time.Second

func shellCommandTimeout() time.Duration {
	if v := os.Getenv("SCOUT_SHELL_TIMEOUT_SECONDS"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultShellCommandTimeout
}

// maxShellOutputBytes caps how much command output is handed back to the
// model. Without a cap, a single command can return megabytes: `go doc -all`
// on a large SDK emits ~1.8MB, which is ~760k tokens in one tool_result.
// The API does not reject that -- it fits a 1M-token context -- so instead
// of failing fast the request spends minutes in prefill with nothing
// streaming back, which reads as a hang. It then poisons every later turn,
// since the full result is resent as conversation history each time.
const maxShellOutputBytes = 32 * 1024

// truncateShellOutput keeps the head and tail of oversized output and
// replaces the middle with an explicit marker. Both ends matter: the head
// carries the command's framing, and the tail carries the errors and
// summaries that most tools print last. The marker tells the model the
// output was cut so it narrows the command instead of assuming it read
// everything.
func truncateShellOutput(output []byte) string {
	if len(output) <= maxShellOutputBytes {
		return string(output)
	}

	head := maxShellOutputBytes / 2
	tail := maxShellOutputBytes - head

	return fmt.Sprintf(
		"%s\n\n[... %d of %d bytes elided: output exceeded the %d byte limit. "+
			"Re-run a narrower command (grep, head, sed -n) to see the rest ...]\n\n%s",
		trimPartialLine(output[:head], false),
		len(output)-head-tail, len(output), maxShellOutputBytes,
		trimPartialLine(output[len(output)-tail:], true),
	)
}

// trimPartialLine drops the incomplete line at a cut point so the model is
// never handed a half-sliced line. leading trims from the start of the
// slice, otherwise the end is trimmed.
func trimPartialLine(b []byte, leading bool) string {
	if leading {
		if i := bytes.IndexByte(b, '\n'); i >= 0 {
			return string(b[i+1:])
		}
		return string(b)
	}
	if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
		return string(b[:i])
	}
	return string(b)
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

	timeout := shellCommandTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	shell, flag := shellInterpreter()
	cmd := exec.CommandContext(ctx, shell, flag, command)

	// Run in its own process group and give Go a grace period to forcibly
	// close the I/O pipes, so the call can never hang forever even if the
	// command spawns background/orphaned children.
	setNewProcessGroup(cmd)
	cmd.WaitDelay = shellCommandWaitDelay
	cmd.Cancel = func() error {
		return killProcessGroup(cmd)
	}

	output, err := cmd.CombinedOutput()
	text := truncateShellOutput(output)

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Sprintf("error: command timed out after %s and was killed\noutput so far:\n%s", timeout, text)
	}

	if err != nil {
		return fmt.Sprintf("error: failed to execute command: %v\noutput:\n%s", err, text)
	}

	return text
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
