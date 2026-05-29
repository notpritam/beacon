// ABOUTME: GUI / computer-use job execution (macOS): mouse, keyboard, and app control.
// ABOUTME: Uses cliclick for pointer/keys and osascript for app + screen queries.

package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type guiPayload struct {
	Action string `json:"action"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Button string `json:"button"` // "left" (default) or "right"
	Double bool   `json:"double"`
	Text   string `json:"text"`
	Key    string `json:"key"`   // named key for "key" (e.g. return, esc, tab, space, arrow-down)
	Combo  string `json:"combo"` // modifier combo for "hotkey" (e.g. "cmd+space", "cmd+c")
	App    string `json:"app"`
}

// gui dispatches a computer-use action. Supported on macOS only.
func (e *Executor) gui(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	if runtime.GOOS != osDarwin {
		return nil, fmt.Errorf("executor: gui is only supported on macOS")
	}
	var p guiPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("executor: gui payload: %w", err)
	}

	switch p.Action {
	case "move":
		return guiDo(runProc(ctx, "cliclick", fmt.Sprintf("m:%d,%d", p.X, p.Y)))
	case "click":
		verb := "c"
		switch {
		case p.Double:
			verb = "dc"
		case p.Button == "right":
			verb = "rc"
		}
		return guiDo(runProc(ctx, "cliclick", fmt.Sprintf("%s:%d,%d", verb, p.X, p.Y)))
	case "type":
		if p.Text == "" {
			return nil, fmt.Errorf("executor: gui type: empty text")
		}
		return guiDo(runProc(ctx, "cliclick", "-w", "10", "t:"+p.Text))
	case "key":
		if p.Key == "" {
			return nil, fmt.Errorf("executor: gui key: empty key")
		}
		return guiDo(runProc(ctx, "cliclick", "kp:"+p.Key))
	case "hotkey":
		return guiDo(hotkey(ctx, p.Combo))
	case "open_app":
		if p.App == "" {
			return nil, fmt.Errorf("executor: gui open_app: empty app")
		}
		return guiDo(runProc(ctx, "open", "-a", p.App))
	case "activate_app":
		if p.App == "" {
			return nil, fmt.Errorf("executor: gui activate_app: empty app")
		}
		return guiDo(runProc(ctx, "osascript", "-e", fmt.Sprintf("tell application %q to activate", p.App)))
	case "quit_app":
		if p.App == "" {
			return nil, fmt.Errorf("executor: gui quit_app: empty app")
		}
		return guiDo(runProc(ctx, "osascript", "-e", fmt.Sprintf("tell application %q to quit", p.App)))
	case "list_apps":
		return listApps(ctx)
	case "screen_size":
		return screenSize(ctx)
	default:
		return nil, fmt.Errorf("executor: gui: unsupported action %q", p.Action)
	}
}

// runProc runs a command and returns its combined output, wrapping failures.
func runProc(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // GUI control via system tools is this agent's purpose
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("executor: gui: %s failed: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// guiDo turns an action error into a JSON {ok:true} result (or the error).
func guiDo(_ string, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, err
	}
	return json.RawMessage(`{"ok":true}`), nil
}

// hotkey presses a modifier+key combo (e.g. "cmd+space", "cmd+c") via osascript.
func hotkey(ctx context.Context, combo string) (string, error) {
	if combo == "" {
		return "", fmt.Errorf("executor: gui hotkey: empty combo")
	}
	parts := strings.Split(strings.ToLower(combo), "+")
	key := parts[len(parts)-1]
	var mods []string
	for _, m := range parts[:len(parts)-1] {
		switch m {
		case "cmd", "command":
			mods = append(mods, "command down")
		case "ctrl", "control":
			mods = append(mods, "control down")
		case "alt", "opt", "option":
			mods = append(mods, "option down")
		case "shift":
			mods = append(mods, "shift down")
		default:
			return "", fmt.Errorf("executor: gui hotkey: unknown modifier %q", m)
		}
	}
	// Named keys map to AppleScript key codes; single chars use keystroke.
	codes := map[string]string{"space": "49", "return": "36", "enter": "36", "tab": "48", "escape": "53", "esc": "53", "delete": "51"}
	var stmt string
	if code, ok := codes[key]; ok {
		stmt = "key code " + code
	} else if len(key) == 1 {
		stmt = fmt.Sprintf("keystroke %q", key)
	} else {
		return "", fmt.Errorf("executor: gui hotkey: unsupported key %q", key)
	}
	if len(mods) > 0 {
		stmt += " using {" + strings.Join(mods, ", ") + "}"
	}
	return runProc(ctx, "osascript", "-e", "tell application \"System Events\" to "+stmt)
}

// listApps returns the names of visible apps and the frontmost app.
func listApps(ctx context.Context) (json.RawMessage, error) {
	out, err := runProc(ctx, "osascript", "-e",
		`tell application "System Events" to get name of (every process whose background only is false)`)
	if err != nil {
		return nil, err
	}
	apps := []string{}
	for _, a := range strings.Split(out, ",") {
		if s := strings.TrimSpace(a); s != "" {
			apps = append(apps, s)
		}
	}
	front, _ := runProc(ctx, "osascript", "-e",
		`tell application "System Events" to get name of first process whose frontmost is true`)
	res := map[string]any{"apps": apps, "frontmost": strings.TrimSpace(front)}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("executor: gui list_apps marshal: %w", err)
	}
	return b, nil
}

// screenSize returns the main display's logical width and height (the coordinate
// space used by move/click).
func screenSize(ctx context.Context) (json.RawMessage, error) {
	out, err := runProc(ctx, "osascript", "-e",
		`tell application "Finder" to get bounds of window of desktop`)
	if err != nil {
		return nil, err
	}
	nums := strings.Split(out, ",")
	if len(nums) != 4 {
		return nil, fmt.Errorf("executor: gui screen_size: unexpected bounds %q", strings.TrimSpace(out))
	}
	atoi := func(s string) int { n, _ := strconv.Atoi(strings.TrimSpace(s)); return n }
	res := map[string]any{"width": atoi(nums[2]) - atoi(nums[0]), "height": atoi(nums[3]) - atoi(nums[1])}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("executor: gui screen_size marshal: %w", err)
	}
	return b, nil
}
