// ABOUTME: registerTools wires each mcptools method to a typed MCP tool.
// ABOUTME: Input structs carry jsonschema tags so the SDK can describe the tools.

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/notpritam/beacon/internal/mcptools"
)

type machineInput struct {
	Machine string `json:"machine" jsonschema:"target machine name"`
}

type guiInput struct {
	Machine string `json:"machine" jsonschema:"target machine name"`
	Action  string `json:"action" jsonschema:"one of: move, click, type, key, hotkey, open_app, activate_app, quit_app, list_apps, screen_size"`
	X       int    `json:"x,omitempty" jsonschema:"x coordinate for move/click"`
	Y       int    `json:"y,omitempty" jsonschema:"y coordinate for move/click"`
	Button  string `json:"button,omitempty" jsonschema:"left (default) or right"`
	Double  bool   `json:"double,omitempty" jsonschema:"double-click"`
	Text    string `json:"text,omitempty" jsonschema:"text to type"`
	Key     string `json:"key,omitempty" jsonschema:"named key for the key action (return, esc, tab, space, arrow-down, ...)"`
	Combo   string `json:"combo,omitempty" jsonschema:"modifier combo for hotkey, e.g. cmd+space or cmd+c"`
	App     string `json:"app,omitempty" jsonschema:"app name for open_app/activate_app/quit_app"`
}

type runCommandInput struct {
	Machine     string `json:"machine"                jsonschema:"target machine name"`
	Cmd         string `json:"cmd"                    jsonschema:"shell command to run"`
	Cwd         string `json:"cwd,omitempty"          jsonschema:"working directory"`
	TimeoutSecs int    `json:"timeout_secs,omitempty" jsonschema:"per-command timeout in seconds"`
}

type pathInput struct {
	Machine string `json:"machine" jsonschema:"target machine name"`
	Path    string `json:"path"    jsonschema:"absolute file or directory path"`
}

type writeInput struct {
	Machine string `json:"machine"  jsonschema:"target machine name"`
	Path    string `json:"path"     jsonschema:"absolute file path"`
	Content string `json:"content"  jsonschema:"file content to write"`
}

type jobInput struct {
	JobID string `json:"job_id" jsonschema:"the job id to fetch"`
}

// machineList wraps the slice so the SDK can infer an "object" output schema.
type machineList struct {
	Machines []mcptools.MachineInfo `json:"machines"`
}

func registerTools(s *mcp.Server, t *mcptools.Tools) {
	mcp.AddTool(s, &mcp.Tool{Name: "list_machines", Description: "List registered machines and their online status."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, machineList, error) {
			out, err := t.ListMachines(ctx)
			return nil, machineList{Machines: out}, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "machine_status", Description: "Get a machine's online status, last_seen, and kill-switch state."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in machineInput) (*mcp.CallToolResult, mcptools.MachineInfo, error) {
			out, err := t.MachineStatus(ctx, in.Machine)
			return nil, out, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "run_command", Description: "Run a shell command on a machine; waits for the result if the machine is online, else returns a queued job id."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in runCommandInput) (*mcp.CallToolResult, mcptools.JobOutcome, error) {
			out, err := t.RunCommand(ctx, in.Machine, in.Cmd, in.Cwd, in.TimeoutSecs)
			return nil, out, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "read_file", Description: "Read a file on a machine."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in pathInput) (*mcp.CallToolResult, mcptools.JobOutcome, error) {
			out, err := t.ReadFile(ctx, in.Machine, in.Path)
			return nil, out, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "write_file", Description: "Write content to a file on a machine."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in writeInput) (*mcp.CallToolResult, mcptools.JobOutcome, error) {
			out, err := t.WriteFile(ctx, in.Machine, in.Path, in.Content)
			return nil, out, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "list_dir", Description: "List a directory on a machine."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in pathInput) (*mcp.CallToolResult, mcptools.JobOutcome, error) {
			out, err := t.ListDir(ctx, in.Machine, in.Path)
			return nil, out, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "get_job", Description: "Get the status and result of a job by id."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in jobInput) (*mcp.CallToolResult, mcptools.JobOutcome, error) {
			out, err := t.GetJob(ctx, in.JobID)
			return nil, out, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "screenshot", Description: "Capture the machine's screen; result has a base64-encoded JPEG."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in machineInput) (*mcp.CallToolResult, mcptools.JobOutcome, error) {
			out, err := t.Screenshot(ctx, in.Machine)
			return nil, out, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "gui", Description: "Computer-use on the machine (macOS): mouse, keyboard, and app control. Set 'action' and the relevant fields."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in guiInput) (*mcp.CallToolResult, mcptools.JobOutcome, error) {
			payload, err := json.Marshal(in) // executor ignores the extra "machine" field
			if err != nil {
				return nil, mcptools.JobOutcome{}, fmt.Errorf("gui payload: %w", err)
			}
			out, err := t.GUI(ctx, in.Machine, payload)
			return nil, out, err
		})
}
