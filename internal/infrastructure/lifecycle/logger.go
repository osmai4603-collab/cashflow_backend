package lifecycle

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ANSI color codes
const (
	ansiReset      = "\033[0m"
	ansiDim        = "\033[90m"
	ansiBold       = "\033[1m"
	ansiRed        = "\033[31m"
	ansiGreen      = "\033[32m"
	ansiYellow     = "\033[33m"
	ansiBlue       = "\033[34m"
	ansiMagenta    = "\033[35m"
	ansiCyan       = "\033[36m"
	ansiWhite      = "\033[37m"
	ansiRedBgWhite = "\033[1;37;41m"
	ansiBoldYellow = "\033[1;33m"
)

// PrettyHandler prints human-friendly, high-contrast ANSI colored logs to terminal.
type PrettyHandler struct {
	w    io.Writer
	opts slog.HandlerOptions
	mu   sync.Mutex
}

// NewPrettyHandler creates an ANSI colored terminal handler.
func NewPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *PrettyHandler {
	var handlerOpts slog.HandlerOptions
	if opts != nil {
		handlerOpts = *opts
	}
	return &PrettyHandler{
		w:    w,
		opts: handlerOpts,
	}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	timeStr := r.Time.Format("2006-01-02 03:04:05 PM")

	// Format level with color
	var levelStr string
	switch {
	case r.Level >= slog.LevelError:
		levelStr = fmt.Sprintf("%s%sERROR%s", ansiBold, ansiRed, ansiReset)
	case r.Level >= slog.LevelWarn:
		levelStr = fmt.Sprintf("%s%sWARN%s ", ansiBold, ansiYellow, ansiReset)
	case r.Level >= slog.LevelInfo:
		levelStr = fmt.Sprintf("%s%sINFO%s ", ansiBold, ansiGreen, ansiReset)
	default:
		levelStr = fmt.Sprintf("%s%sDEBUG%s", ansiDim, ansiMagenta, ansiReset)
	}

	// Format message
	msg := r.Message

	// Format attributes
	var attrs strings.Builder
	r.Attrs(func(a slog.Attr) bool {
		val := formatAttrValue(a.Key, a.Value.Any())
		attrs.WriteString(fmt.Sprintf(" %s%s=%s%s%s", ansiDim, a.Key, ansiReset, val, ansiReset))
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := fmt.Fprintf(h.w, "%s%s%s [%s] %s%s\n",
		ansiDim, timeStr, ansiReset,
		levelStr,
		msg,
		attrs.String(),
	)
	return err
}

func formatAttrValue(key string, val any) string {
	if key == "status" {
		switch s := val.(type) {
		case int:
			return ColorizeStatus(s)
		case int64:
			return ColorizeStatus(int(s))
		}
	}
	return fmt.Sprintf("%v", val)
}

// ColorizeStatus formats HTTP status codes with high-contrast ANSI colors.
func ColorizeStatus(status int) string {
	switch {
	case status >= 500:
		return fmt.Sprintf("%sstatus=%d%s", ansiRedBgWhite, status, ansiReset)
	case status >= 400:
		return fmt.Sprintf("%sstatus=%d%s", ansiBoldYellow, status, ansiReset)
	case status >= 300:
		return fmt.Sprintf("%sstatus=%d%s", ansiCyan, status, ansiReset)
	case status >= 200:
		return fmt.Sprintf("%sstatus=%d%s", ansiGreen, status, ansiReset)
	default:
		return fmt.Sprintf("status=%d", status)
	}
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return h
}

// MultiHandler fan-outs records to multiple handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

// NewMultiHandler creates a multiplexing handler.
func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: newHandlers}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: newHandlers}
}

// MinLevelFilterHandler wraps a handler and restricts to a minimum log level.
type MinLevelFilterHandler struct {
	handler  slog.Handler
	minLevel slog.Level
}

func (f *MinLevelFilterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= f.minLevel && f.handler.Enabled(ctx, level)
}

func (f *MinLevelFilterHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= f.minLevel {
		return f.handler.Handle(ctx, r)
	}
	return nil
}

func (f *MinLevelFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &MinLevelFilterHandler{handler: f.handler.WithAttrs(attrs), minLevel: f.minLevel}
}

func (f *MinLevelFilterHandler) WithGroup(name string) slog.Handler {
	return &MinLevelFilterHandler{handler: f.handler.WithGroup(name), minLevel: f.minLevel}
}

// LogCloser ensures buffered and open log file handles are flushed and closed safely.
type LogCloser struct {
	files []*os.File
	mu    sync.Mutex
}

// Sync flushes all log files to disk.
func (c *LogCloser) Sync() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, f := range c.files {
		_ = f.Sync()
	}
	return nil
}

// Close flushes and closes all open log file descriptors.
func (c *LogCloser) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, f := range c.files {
		_ = f.Sync()
		_ = f.Close()
	}
	c.files = nil
	return nil
}

// InitLogger initializes a production-grade multi-target logger.
// It combines an ANSI colored terminal output with structured JSON file logging.
func InitLogger(cfg *Config) (*slog.Logger, *LogCloser, error) {
	var handlers []slog.Handler
	closer := &LogCloser{}

	level := slog.LevelInfo
	switch strings.ToUpper(cfg.LogLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format("2006-01-02 03:04:05 PM"))
			}
			return a
		},
	}

	// 1. Terminal Handler (Colorized if enabled, else JSON)
	if cfg.LogColor {
		handlers = append(handlers, NewPrettyHandler(os.Stderr, opts))
	} else {
		handlers = append(handlers, slog.NewJSONHandler(os.Stderr, opts))
	}

	// 2. File Persistence (app.log and error.log)
	if cfg.LogToFile && cfg.LogDir != "" {
		if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
			return nil, nil, fmt.Errorf("create log directory %s: %w", cfg.LogDir, err)
		}

		// App Log: all events
		appPath := filepath.Join(cfg.LogDir, "app.log")
		appFile, err := os.OpenFile(appPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("open app log %s: %w", appPath, err)
		}
		closer.files = append(closer.files, appFile)
		handlers = append(handlers, slog.NewJSONHandler(appFile, opts))

		// Error Log: Level >= WARN only
		errorPath := filepath.Join(cfg.LogDir, "error.log")
		errorFile, err := os.OpenFile(errorPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			_ = closer.Close()
			return nil, nil, fmt.Errorf("open error log %s: %w", errorPath, err)
		}
		closer.files = append(closer.files, errorFile)
		handlers = append(handlers, &MinLevelFilterHandler{
			handler:  slog.NewJSONHandler(errorFile, opts),
			minLevel: slog.LevelWarn,
		})
	}

	logger := slog.New(NewMultiHandler(handlers...))
	return logger, closer, nil
}
