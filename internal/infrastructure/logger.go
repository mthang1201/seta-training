package infrastructure

import (
	"log/slog"
	"os"
	"path/filepath"
)

var Log *slog.Logger

func InitLogger() {
	// Create logs directory if not exists
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic("Failed to create log directory")
	}

	// Open log file
	file, err := os.OpenFile(filepath.Join(logDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic("Failed to open log file")
	}

	// Create JSON handler
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Also write to stdout for local development visibility
	// but mostly we want promtail to read from app.log
	multiHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	
	// Create a wrapper to write to both (simplified, standard library doesn't have a multi-handler out of the box,
	// so we will just write to the file for promtail to scrape).
	// Actually, writing to stdout and capturing it via Docker driver is better when running in Docker,
	// but since the app runs locally on the host via `go run`, we write to file.
	Log = slog.New(handler)
	
	// Set it as default logger so standard log package uses it
	slog.SetDefault(Log)
	
	// Optional: add a logger for stdout as well if user wants to see it in terminal
	_ = multiHandler
}
