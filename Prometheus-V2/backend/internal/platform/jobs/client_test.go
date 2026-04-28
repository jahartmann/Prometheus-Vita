package jobs_test

import (
	"context"
	"os"
	"testing"

	"github.com/antigravity/prometheus-v2/internal/platform/db"
	"github.com/antigravity/prometheus-v2/internal/platform/jobs"
	"github.com/stretchr/testify/require"
)

func TestNewClient_RegistersWorkers(t *testing.T) {
	dsn := os.Getenv("PROMETHEUS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PROMETHEUS_TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := db.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, jobs.MigrateUp(ctx, pool))

	client, err := jobs.NewClient(ctx, pool, jobs.NewWorkers(), jobs.DefaultQueues())
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.Stop(ctx))
}
