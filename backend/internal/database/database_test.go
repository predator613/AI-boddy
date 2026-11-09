package database

import (
	"context"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/migrations"
)

// Интеграционные тесты: нужна живая база в TEST_DATABASE_URL (в docker compose она задается сама).
func testURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	return url
}

func TestConnectAndMigrateIsIdempotent(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testURL(t), 10*time.Second)
	require.NoError(t, err)
	defer pool.Close()

	_, err = Migrate(ctx, pool, migrations.FS)
	require.NoError(t, err)

	// Второй запуск ничего не применяет.
	applied, err := Migrate(ctx, pool, migrations.FS)
	require.NoError(t, err)
	assert.Empty(t, applied)

	var users int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&users))
	assert.GreaterOrEqual(t, users, 4)
}

func TestMigrateBrokenFileIsRolledBack(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testURL(t), 10*time.Second)
	require.NoError(t, err)
	defer pool.Close()

	broken := fstest.MapFS{"999_broken.sql": {Data: []byte(`CREATE TABLE broken_tmp (id int); SELEC oops;`)}}
	_, err = Migrate(ctx, pool, broken)
	require.Error(t, err)

	var exists bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = '999_broken.sql')`).Scan(&exists))
	assert.False(t, exists)
	require.NoError(t, pool.QueryRow(ctx, `SELECT to_regclass('broken_tmp') IS NOT NULL`).Scan(&exists))
	assert.False(t, exists, "partial migration must be rolled back")
}

func TestConnectErrors(t *testing.T) {
	_, err := Connect(context.Background(), "::not a url::", time.Second)
	assert.Error(t, err)

	// Порт, на котором никто не слушает: ждем недолго и получаем ошибку.
	_, err = Connect(context.Background(), "postgres://u:p@127.0.0.1:1/db?connect_timeout=1", 600*time.Millisecond)
	assert.Error(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Connect(ctx, "postgres://u:p@127.0.0.1:1/db?connect_timeout=1", time.Minute)
	assert.Error(t, err)
}
