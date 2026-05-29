// ABOUTME: registerTools wires each mcptools method to a typed MCP tool.
// ABOUTME: Input structs carry jsonschema tags so the SDK can describe the tools.

package main

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/notpritam/beacon/internal/mcptools"
)

type machineInput struct {
	Machine string `json:"machine" jsonschema:"target machine name"`
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
}
