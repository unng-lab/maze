package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
}

func New(cfg Config, builder maze.Builder, renderer *render.Renderer, logger *slog.Logger) *Server {
	return &Server{cfg: cfg, builder: builder, renderer: renderer, logger: logger}
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
	layout, err := s.builder.Build(s.cfg.MazeWidth, s.cfg.MazeHeight, s.cfg.ExitLabels)
	if err != nil {
		s.logger.Error("build maze", "error", err)
		http.Error(w, "cannot build maze", http.StatusInternalServerError)
		return
	}

	html, err := s.renderer.Render(layout, s.cfg.CellSize)
	if err != nil {
		s.logger.Error("render maze", "error", err)
		http.Error(w, "cannot render maze", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
