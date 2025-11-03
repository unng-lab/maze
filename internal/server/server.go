package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"log/slog"

	"maze/internal/maze"
	"maze/internal/render"
)

type Config struct {
	Address    string
	MazeWidth  int
	MazeHeight int
	CellSize   int
	ExitLabels []string
}

type Server struct {
	cfg      Config
	builder  maze.Builder
	renderer *render.Renderer
	logger   *slog.Logger
	tempDir  string
	health   string
}

func New(cfg Config, builder maze.Builder, renderer *render.Renderer, logger *slog.Logger) (*Server, error) {
	tempDir, err := os.MkdirTemp("", "maze-server-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	healthFile := filepath.Join(tempDir, "healthz.txt")
	if err := os.WriteFile(healthFile, []byte("ok"), 0o644); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("write health file: %w", err)
	}

	return &Server{
		cfg:      cfg,
		builder:  builder,
		renderer: renderer,
		logger:   logger,
		tempDir:  tempDir,
		health:   healthFile,
	}, nil
}

func (s *Server) Close() error {
	if s.tempDir == "" {
		return nil
	}

	if err := os.RemoveAll(s.tempDir); err != nil {
		return fmt.Errorf("remove temp dir: %w", err)
	}

	s.tempDir = ""
	s.health = ""
	return nil
}

func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleMaze)
	mux.HandleFunc("/maze", s.handleMaze)
	mux.HandleFunc("/healthz", s.handleHealth)
	return mux
}

func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:    s.cfg.Address,
		Handler: s.handler(),
	}

	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("HTTP server starting", "address", s.cfg.Address)
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s.logger.Info("HTTP server shutting down")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) handleMaze(w http.ResponseWriter, r *http.Request) {
	filePath, err := s.prepareMazeFile()
	if err != nil {
		s.logger.Error("prepare maze file", "error", err)
		http.Error(w, "cannot prepare maze", http.StatusInternalServerError)
		return
	}
	defer func() {
		if removeErr := os.Remove(filePath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			s.logger.Error("remove maze file", "error", removeErr, "path", filePath)
		}
	}()

	http.ServeFile(w, r, filePath)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, s.health)
}

func (s *Server) prepareMazeFile() (string, error) {
	layout, err := s.builder.Build(s.cfg.MazeWidth, s.cfg.MazeHeight, s.cfg.ExitLabels)
	if err != nil {
		return "", fmt.Errorf("build maze: %w", err)
	}

	html, err := s.renderer.Render(layout, s.cfg.CellSize)
	if err != nil {
		return "", fmt.Errorf("render maze: %w", err)
	}

	if err := os.MkdirAll(s.tempDir, 0o755); err != nil {
		return "", fmt.Errorf("ensure temp dir: %w", err)
	}

	filePath := filepath.Join(s.tempDir, fmt.Sprintf("maze-%d.html", time.Now().UnixNano()))
	if err := os.WriteFile(filePath, []byte(html), 0o644); err != nil {
		return "", fmt.Errorf("write maze file: %w", err)
	}

	return filePath, nil
}
