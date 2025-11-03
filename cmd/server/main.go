package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	envconfig "maze/internal/config"
	"maze/internal/maze"
	"maze/internal/render"
	"maze/internal/server"
)

type config struct {
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

	if err := srv.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("run server", "error", err)
		os.Exit(1)
	}

	logger.Info("application stopped")
}

func loadConfig() (config, error) {
	cfg := config{
		httpAddress: envconfig.String("HTTP_ADDRESS", ":8080"),
		mazeWidth:   envconfig.Int("MAZE_WIDTH", 40),
		mazeHeight:  envconfig.Int("MAZE_HEIGHT", 40),
		cellSize:    envconfig.Int("MAZE_CELL_SIZE", 20),
		exitLabels:  envconfig.SplitAndTrim(envconfig.String("MAZE_EXIT_LABELS", "A,B,C,D")),
	}

	if len(cfg.exitLabels) < 2 {
		return config{}, errors.New("provide at least two exit labels")
	}

	if cfg.mazeWidth <= 0 || cfg.mazeHeight <= 0 {
		return config{}, errors.New("maze dimensions must be positive")
	}

	if cfg.cellSize <= 0 {
		return config{}, errors.New("cell size must be positive")
	}

	return cfg, nil
}
