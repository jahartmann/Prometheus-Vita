package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/antigravity/prometheus-v2/internal/platform/db"
	"github.com/stretchr/testify/require"
)

func TestNew_PingsPostgres(t *testing.T) {
	dsn := os.Getenv("PROMETHEUS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PROMETHEUS_TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := db.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, pool.Ping(ctx))
}
