package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn             string
		maxOpenConns    int
		minConns        int
		maxConnIdleTime time.Duration
	}
}

type application struct {
	config config
	logger *slog.Logger
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	err := godotenv.Load()
	if err != nil {
		logger.Error("error loading .env file")
		os.Exit(1)
	}

	dsn := os.Getenv("DB_DSN")

	if dsn == "" {
		logger.Error("required DB_DSN env var missing")
		os.Exit(1)
	}

	var cfg config

	flag.StringVar(&cfg.db.dsn, "db-dsn", dsn, "PostgreSQL DSN")
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.minConns, "db-min-conns", 6, "PostgreSQL min connections")
	flag.DurationVar(&cfg.db.maxConnIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	flag.Parse()

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer db.Close()

	logger.Info("db connection established")
}

func openDB(cfg config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = int32(cfg.db.maxOpenConns)
	poolConfig.MinConns = int32(cfg.db.minConns) // use ~25% of MaxConns
	poolConfig.MaxConnIdleTime = cfg.db.maxConnIdleTime

	connectionPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	err = connectionPool.Ping(ctx)
	if err != nil {
		connectionPool.Close()
		return nil, err
	}

	return connectionPool, nil
}
