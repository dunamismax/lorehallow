package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewJSON(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, "json", slog.LevelInfo).Info("hello", "player", 7)
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("output is not JSON: %v: %q", err, buf.String())
	}
	if rec["msg"] != "hello" || rec["player"] != float64(7) {
		t.Errorf("unexpected record: %v", rec)
	}
}

func TestNewTextAndLevel(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, "text", slog.LevelWarn)
	log.Info("hidden")
	log.Warn("shown")
	out := buf.String()
	if strings.Contains(out, "hidden") || !strings.Contains(out, "msg=shown") {
		t.Errorf("unexpected text output: %q", out)
	}
}
