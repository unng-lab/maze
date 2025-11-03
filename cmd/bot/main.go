package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"maze/internal/bot"
	envconfig "maze/internal/config"
)

type config struct {
	botToken  string
	channelID string
	command   string
	webAppURL string
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

	if err := botInstance.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("run bot", "error", err)
		os.Exit(1)
	}

	logger.Info("application stopped")
}

func loadConfig() (config, error) {
	botToken, err := envconfig.RequiredString("TELEGRAM_BOT_TOKEN")
	if err != nil {
		return config{}, err
	}

	channelID, err := envconfig.RequiredString("TELEGRAM_CHANNEL_ID")
	if err != nil {
		return config{}, err
	}

	return config{
		botToken:  botToken,
		channelID: channelID,
		command:   envconfig.String("BOT_COMMAND", "maze"),
		webAppURL: envconfig.String("WEBAPP_URL", "http://localhost:8080"),
	}, nil
}
