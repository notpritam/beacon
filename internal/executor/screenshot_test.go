// ABOUTME: Tests for screenshot encoding (pure) and an opt-in real-capture test.
// ABOUTME: The real capture is darwin-only and gated behind BEACON_TEST_SCREENSHOT=1.
package executor

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"runtime"
	"testing"

	"github.com/notpritam/beacon/internal/store"
)

func TestEncodeScreenshot(t *testing.T) {
	raw := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10} // JPEG magic + a few bytes
	out, err := encodeScreenshot(raw)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var res screenshotResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res.Format != "jpeg" {
		t.Errorf("format = %q, want jpeg", res.Format)
	}
	if res.Bytes != len(raw) {
		t.Errorf("bytes = %d, want %d", res.Bytes, len(raw))
	}
	decoded, err := base64.StdEncoding.DecodeString(res.Base64)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if string(decoded) != string(raw) {
		t.Error("decoded base64 does not match the original bytes")
	}
}

func TestScreenshotUnsupportedOnNonDarwin(t *testing.T) {
	if runtime.GOOS == osDarwin {
		t.Skip("darwin supports screenshot; this checks the non-darwin guard")
	}
	e := New(DefaultConfig())
	if _, err := e.Execute(context.Background(), store.Job{Type: store.JobScreenshot, Payload: json.RawMessage(`{}`)}); err == nil {
		t.Error("expected screenshot to be unsupported off macOS")
	}
}

// TestScreenshotCapture does a real screen capture. It is opt-in (set
// BEACON_TEST_SCREENSHOT=1) and requires macOS + Screen Recording permission.
func TestScreenshotCapture(t *testing.T) {
	if os.Getenv("BEACON_TEST_SCREENSHOT") != "1" {
		t.Skip("set BEACON_TEST_SCREENSHOT=1 to run the real capture test")
	}
	if runtime.GOOS != osDarwin {
		t.Skip("real capture only on macOS")
	}
	e := New(DefaultConfig())
	raw, err := e.Execute(context.Background(), store.Job{Type: store.JobScreenshot, Payload: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatalf("screenshot: %v", err)
	}
	var res screenshotResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	img, err := base64.StdEncoding.DecodeString(res.Base64)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(img) < 2 || img[0] != 0xFF || img[1] != 0xD8 {
		t.Errorf("expected JPEG magic bytes, got %d bytes starting %x", len(img), img[:min(2, len(img))])
	}
}
