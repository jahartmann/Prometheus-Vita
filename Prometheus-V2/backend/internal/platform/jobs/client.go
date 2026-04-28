package jobs

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus-v2/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// QueueDefault is the name of the default queue used by domain modules until
// they declare their own.
const QueueDefault = "default"

// Client is the River queue client parameterized on the pgx v5 transaction
// type used by the riverpgxv5 driver.
type Client = river.Client[pgx.Tx]

// NewWorkers returns an empty worker registry. Domain modules append their
// workers to it before NewClient is called.
func NewWorkers() *river.Workers {
	return river.NewWorkers()
}

// NewClient builds a River client backed by the shared pgx pool and starts it.
// The caller is responsible for invoking Stop on shutdown.
func NewClient(ctx context.Context, pool *db.Pool, workers *river.Workers) (*Client, error) {
	driver := riverpgxv5.New(pool.Pool)
	cli, err := river.NewClient(driver, &river.Config{
		Workers: workers,
		Queues: map[string]river.QueueConfig{
			QueueDefault: {MaxWorkers: 4},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("init river client: %w", err)
	}
	if err := cli.Start(ctx); err != nil {
		return nil, fmt.Errorf("start river client: %w", err)
	}
	return cli, nil
}

// MigrateUp applies all pending River schema migrations against the shared
// pgx pool. It is safe to invoke on every server start.
func MigrateUp(ctx context.Context, pool *db.Pool) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool.Pool), nil)
	if err != nil {
		return fmt.Errorf("init river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("river migrate up: %w", err)
	}
	return nil
}
