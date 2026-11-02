package logs

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLevelFromPrefix(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(levelPrefixHandler{slog.NewJSONHandler(&out, nil)})

	for message, want := range map[string][2]string{
		"WARN failed to look up a session: oops":  {"WARN", "failed to look up a session: oops"},
		"ERROR GET /trades: boom":                 {"ERROR", "GET /trades: boom"},
		"[WARN] old style":                        {"WARN", "old style"},
		"Server running on http://localhost:8080": {"INFO", "Server running on http://localhost:8080"},
	} {
		out.Reset()
		logger.Info(message)
		var line struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
		}
		require.NoError(t, json.Unmarshal(out.Bytes(), &line))
		require.Equal(t, want[0], line.Level, message)
		require.Equal(t, want[1], line.Msg)
	}
}
