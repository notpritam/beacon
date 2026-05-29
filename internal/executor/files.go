// ABOUTME: File-operation jobs: read_file (size-capped), write_file, and list_dir.
// ABOUTME: All operate on operator-supplied paths and return JSON results.

package executor

import (
	"encoding/json"
	"fmt"
	"os"
)

type pathPayload struct {
	Path string `json:"path"`
}

type readResult struct {
	Content string `json:"content"`
}

// Note: file content is carried as a UTF-8 JSON string; binary files are not
// supported (invalid bytes are mangled by JSON). Base64 content is a later phase.
func (e *Executor) readFile(payload json.RawMessage) (json.RawMessage, error) {
	var p pathPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("executor: read_file payload: %w", err)
	}
	if p.Path == "" {
		return nil, fmt.Errorf("executor: read_file: empty path")
	}
	info, err := os.Stat(p.Path)
	if err != nil {
		return nil, fmt.Errorf("executor: read_file stat: %w", err)
	}
	if info.Size() > e.cfg.MaxReadBytes {
		return nil, fmt.Errorf("executor: read_file: %s is %d bytes, exceeds limit %d", p.Path, info.Size(), e.cfg.MaxReadBytes)
	}
	data, err := os.ReadFile(p.Path)
	if err != nil {
		return nil, fmt.Errorf("executor: read_file: %w", err)
	}
	out, err := json.Marshal(readResult{Content: string(data)})
	if err != nil {
		return nil, fmt.Errorf("executor: read_file marshal: %w", err)
	}
	return out, nil
}

type writePayload struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type writeResult struct {
	BytesWritten int `json:"bytes_written"`
}

func (e *Executor) writeFile(payload json.RawMessage) (json.RawMessage, error) {
	var p writePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("executor: write_file payload: %w", err)
	}
	if p.Path == "" {
		return nil, fmt.Errorf("executor: write_file: empty path")
	}
	if err := os.WriteFile(p.Path, []byte(p.Content), 0o644); err != nil { //nolint:gosec // G306: intentional — 0644 is correct for user-writable output files
		return nil, fmt.Errorf("executor: write_file: %w", err)
	}
	out, err := json.Marshal(writeResult{BytesWritten: len(p.Content)})
	if err != nil {
		return nil, fmt.Errorf("executor: write_file marshal: %w", err)
	}
	return out, nil
}

type dirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

type listResult struct {
	Entries []dirEntry `json:"entries"`
}

func (e *Executor) listDir(payload json.RawMessage) (json.RawMessage, error) {
	var p pathPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("executor: list_dir payload: %w", err)
	}
	if p.Path == "" {
		return nil, fmt.Errorf("executor: list_dir: empty path")
	}
	entries, err := os.ReadDir(p.Path)
	if err != nil {
		return nil, fmt.Errorf("executor: list_dir: %w", err)
	}
	result := listResult{Entries: make([]dirEntry, 0, len(entries))}
	for _, ent := range entries {
		info, err := ent.Info()
		if err != nil {
			return nil, fmt.Errorf("executor: list_dir info %s: %w", ent.Name(), err)
		}
		result.Entries = append(result.Entries, dirEntry{
			Name:  ent.Name(),
			IsDir: ent.IsDir(),
			Size:  info.Size(),
		})
	}
	out, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("executor: list_dir marshal: %w", err)
	}
	return out, nil
}
