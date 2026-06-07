package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"github.com/Alston16/convkit/internal/bus"
	convkitctx "github.com/Alston16/convkit/internal/context"
	"github.com/Alston16/convkit/internal/orchestration"
	"github.com/Alston16/convkit/internal/safety"
	"github.com/Alston16/convkit/internal/stream"
	"github.com/Alston16/convkit/internal/tools"
	"github.com/Alston16/convkit/internal/transport"
)

type serverConfig struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Postgres struct {
		DSN string `yaml:"dsn"`
	} `yaml:"postgres"`
	Redis struct {
		Addr string `yaml:"addr"`
	} `yaml:"redis"`
	NATS struct {
		URL string `yaml:"url"`
	} `yaml:"nats"`
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := loadConfig()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// --- Postgres + migrations ---
	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pool.Close()

	sqlDB := stdlib.OpenDBFromPool(pool)
	if err := runMigrations(sqlDB); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	// --- Safety plane (no-op, Stage 0) ---
	sp := safety.NewNoop()

	// --- Layer stubs ---
	_ = transport.New(transport.Config{Safety: sp})
	_ = bus.New(bus.Config{Safety: sp})
	_ = stream.New(stream.Config{Safety: sp})
	_ = convkitctx.New(convkitctx.Config{Safety: sp})
	tools.New(tools.Config{Safety: sp})
	orchestration.New(orchestration.Config{Safety: sp})

	// --- HTTP server ---
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		log.Info().Str("addr", addr).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Error().Err(err).Msg("graceful shutdown error")
	}
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func loadConfig() serverConfig {
	path := os.Getenv("CONVKIT_CONFIG")
	if path == "" {
		path = "config/server.yaml"
	}

	data, err := os.ReadFile(path) // #nosec G304 — path comes from env var, not user input
	if err != nil {
		log.Fatal().Err(err).Str("path", path).Msg("failed to read config file")
	}

	var cfg serverConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to parse config file")
	}
	return cfg
}

func runMigrations(db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("runMigrations: set dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("runMigrations: up: %w", err)
	}
	return nil
}
