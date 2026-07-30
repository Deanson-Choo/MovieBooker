package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Init(ctx context.Context) error {
	// 1. Read direct connection string from environment
	connString := os.Getenv("POSTGRES_CONN_STR")
	if connString == "" {
		return fmt.Errorf("POSTGRES_CONN_STR is not set")
	}

	// 2. Parse pool config from DSN
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return err
	}

	// 3. Apply Best-Practice Production Settings
	config.MaxConns = 20                       // Absolute ceiling on active connections
	config.MinConns = 10                       // Keep cold connections alive to eliminate cold starts
	config.MaxConnIdleTime = 15 * time.Minute  // Terminate idle connections to free DB memory
	config.MaxConnLifetime = 1 * time.Hour     // Prevent memory leaks / stale connections
	config.HealthCheckPeriod = 5 * time.Second // Automatically prune dropped/broken sockets

	// 4. Establish the pool manager
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	// 5. Force a ping to guarantee physical network connectivity at start
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return err
	}

	Pool = pool
	return nil
}
