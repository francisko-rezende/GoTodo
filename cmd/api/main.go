package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file")
		os.Exit(1)
	}

	dsn := os.Getenv("DB_DSN")

	if dsn == "" {
		log.Fatal("required DB_DSN env var missing")
		os.Exit(1)
	}

	db, err := openDB(dsn)
	if err != nil {
		log.Fatal("failed to open db connection pool")
		os.Exit(1)
	}

	defer db.Close()

	fmt.Println("db connection established")
}

func openDB(dsn string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 6 // use ~25% of MaxConns
	poolConfig.MaxConnIdleTime = 15 * time.Minute

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
