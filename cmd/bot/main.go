package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"maze/internal/bot"
	"maze/internal/maze"
	"maze/internal/render"
	"maze/internal/server"
)

type config struct {
	botToken    string
	channelID   string
	command     string
	webAppURL   string
	httpAddress string
	mazeWidth   int
	mazeHeight  int
	cellSize    int
	exitLabels  []string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := loadConfig()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	renderer, err := render.NewRenderer()
	if err != nil {
		logger.Error("create renderer", "error", err)
		os.Exit(1)
	}

	builder := maze.NewRandomBuilder(time.Now().UnixNano())

	srv, err := server.New(server.Config{
		Address:    cfg.httpAddress,
		MazeWidth:  cfg.mazeWidth,
		MazeHeight: cfg.mazeHeight,
		CellSize:   cfg.cellSize,
		ExitLabels: cfg.exitLabels,
	}, builder, renderer, logger)
	if err != nil {
		logger.Error("create server", "error", err)
		os.Exit(1)
	}
	defer func() {
		if cerr := srv.Close(); cerr != nil {
			logger.Error("cleanup server", "error", cerr)
		}
	}()

	botInstance, err := bot.New(bot.Config{
		Token:     cfg.botToken,
		ChannelID: cfg.channelID,
		Command:   cfg.command,
		WebAppURL: cfg.webAppURL,
		Logger:    logger,
	})
	if err != nil {
		logger.Error("create telegram bot", "error", err)
		os.Exit(1)
	}

	errCh := make(chan error, 2)

	go func() {
		errCh <- srv.Start(ctx)
	}()

	go func() {
		errCh <- botInstance.Run(ctx)
	}()

	var runErr error
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) {
			runErr = err
			stop()
		}
	}

	if runErr != nil {
		logger.Error("shutdown with error", "error", runErr)
		os.Exit(1)
	}

	logger.Info("application stopped")
}

func loadConfig() (config, error) {
	var cfg config

	flag.StringVar(&cfg.botToken, "bot-token", os.Getenv("TELEGRAM_BOT_TOKEN"), "Telegram bot token")
	flag.StringVar(&cfg.channelID, "channel-id", os.Getenv("TELEGRAM_CHANNEL_ID"), "Telegram channel username or ID")
	flag.StringVar(&cfg.command, "command", envOrDefault("BOT_COMMAND", "maze"), "Command to publish mazes")
	flag.StringVar(&cfg.webAppURL, "webapp-url", os.Getenv("WEBAPP_URL"), "Public URL to the web app")
	flag.StringVar(&cfg.httpAddress, "http-address", envOrDefault("HTTP_ADDRESS", ":8080"), "HTTP server address")
	flag.IntVar(&cfg.mazeWidth, "maze-width", envInt("MAZE_WIDTH", 40), "Maze width")
	flag.IntVar(&cfg.mazeHeight, "maze-height", envInt("MAZE_HEIGHT", 40), "Maze height")
	flag.IntVar(&cfg.cellSize, "cell-size", envInt("MAZE_CELL_SIZE", 20), "Cell size in pixels")

	var exitLabels string
	flag.StringVar(&exitLabels, "exit-labels", envOrDefault("MAZE_EXIT_LABELS", "A,B,C,D"), "Comma separated exit labels")

	flag.Parse()

	cfg.exitLabels = splitAndTrim(exitLabels)

	if len(cfg.exitLabels) < 2 {
		return config{}, errors.New("provide at least two exit labels")
	}

	if cfg.botToken == "" {
		return config{}, errors.New("TELEGRAM_BOT_TOKEN is required")
	}

	if cfg.channelID == "" {
		return config{}, errors.New("TELEGRAM_CHANNEL_ID is required")
	}

	if cfg.webAppURL == "" {
		return config{}, errors.New("WEBAPP_URL is required")
	}

	if cfg.mazeWidth <= 0 || cfg.mazeHeight <= 0 {
		return config{}, errors.New("maze dimensions must be positive")
	}

	if cfg.cellSize <= 0 {
		return config{}, errors.New("cell size must be positive")
	}

	return cfg, nil
}

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func envInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func splitAndTrim(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
