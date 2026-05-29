// ABOUTME: Screenshot job execution: captures the screen via macOS screencapture, optionally
// ABOUTME: downscales with sips, and returns a base64-encoded JPEG with size metadata.

package executor

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

const osDarwin = "darwin"

type screenshotResult struct {
	Format string `json:"format"`
	Bytes  int    `json:"bytes"`
	Base64 string `json:"base64"`
}

// screenshot captures the whole screen (macOS only), downscales it to keep the
// payload small, and returns it as a base64-encoded JPEG.
func (e *Executor) screenshot(ctx context.Context) (json.RawMessage, error) {
	if runtime.GOOS != osDarwin {
		return nil, fmt.Errorf("executor: screenshot is only supported on macOS")
	}

	dir, err := os.MkdirTemp("", "beacon-shot-")
	if err != nil {
		return nil, fmt.Errorf("executor: screenshot tempdir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "shot.jpg")

	// -x: no camera sound. -t jpg: JPEG output. Captures the full desktop.
	capCmd := exec.CommandContext(ctx, "screencapture", "-x", "-t", "jpg", path) //nolint:gosec // capturing the local screen is this agent's purpose
	if out, err := capCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("executor: screencapture failed (grant Screen Recording permission?): %w: %s", err, string(out))
	}
	if fi, statErr := os.Stat(path); statErr != nil || fi.Size() == 0 {
		return nil, fmt.Errorf("executor: screencapture produced no image (Screen Recording permission may be missing)")
	}

	// Best-effort downscale so the encoded payload stays small.
	if e.cfg.MaxScreenshotDim > 0 {
		resize := exec.CommandContext(ctx, "sips", "-Z", strconv.Itoa(e.cfg.MaxScreenshotDim), path) //nolint:gosec // resizing our own temp screenshot
		_ = resize.Run()                                                                             // if sips fails, fall back to the full-size capture
	}

	data, err := os.ReadFile(path) //nolint:gosec // reading our own temp screenshot file
	if err != nil {
		return nil, fmt.Errorf("executor: read screenshot: %w", err)
	}
	if e.cfg.MaxScreenshotBytes > 0 && len(data) > e.cfg.MaxScreenshotBytes {
		return nil, fmt.Errorf("executor: screenshot is %d bytes, exceeds limit %d", len(data), e.cfg.MaxScreenshotBytes)
	}
	return encodeScreenshot(data)
}

// encodeScreenshot wraps raw JPEG bytes into the screenshot result JSON.
func encodeScreenshot(data []byte) (json.RawMessage, error) {
	res := screenshotResult{
		Format: "jpeg",
		Bytes:  len(data),
		Base64: base64.StdEncoding.EncodeToString(data),
	}
	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("executor: screenshot marshal: %w", err)
	}
	return out, nil
}
