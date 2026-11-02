// Package logs writes the server's logs as one JSON object per line, with a level, for Logstash to index
package logs

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Leading words of log.Printf messages naming their level, e.g "WARN failed to ..."
var levelPrefixes = []struct {
	prefix string
	level  slog.Level
}{
	{"ERROR ", slog.LevelError},
	{"[ERROR] ", slog.LevelError},
	{"WARN ", slog.LevelWarn},
	{"[WARN] ", slog.LevelWarn},
	{"WARNING ", slog.LevelWarn},
	{"INFO ", slog.LevelInfo},
	{"[INFO] ", slog.LevelInfo},
	{"DEBUG ", slog.LevelDebug},
}

// levelPrefixHandler reads the level of messages from their first word, the rest are informational
type levelPrefixHandler struct {
	slog.Handler
}

func (h levelPrefixHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, p := range levelPrefixes {
		if message, ok := strings.CutPrefix(record.Message, p.prefix); ok {
			leveled := slog.NewRecord(record.Time, p.level, message, record.PC)
			record.Attrs(func(attr slog.Attr) bool {
				leveled.AddAttrs(attr)
				return true
			})
			return h.Handler.Handle(ctx, leveled)
		}
	}
	return h.Handler.Handle(ctx, record)
}

func (h levelPrefixHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return levelPrefixHandler{h.Handler.WithAttrs(attrs)}
}

func (h levelPrefixHandler) WithGroup(name string) slog.Handler {
	return levelPrefixHandler{h.Handler.WithGroup(name)}
}

// Setup sends the log and slog packages' output to stdout as JSON lines, see docker/logstash/pipeline/logs.conf
func Setup() {
	slog.SetDefault(slog.New(levelPrefixHandler{slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})}))
}
