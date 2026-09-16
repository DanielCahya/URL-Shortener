package repository_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DanielCahya/url-shortener/internal/repository"
)

func TestMigrations_AnalyticsTablesExist(t *testing.T) {
	// Ensure DB is running
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping test, could not connect to database: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// 1. Run migrations manually just in case, though they might already be run by another test's init
	err = repository.RunMigrations(ctx, pool)
	require.NoError(t, err)

	// 2. Check outbox_events table exists
	var exists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'outbox_events')").Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "outbox_events table should exist")

	// 3. Check click_events table exists
	err = pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'click_events')").Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "click_events table should exist")

	// 4. Verify unique constraint on click_events(event_id)
	var constraintExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint 
			WHERE conname = 'uq_click_events_event_id'
		)`).Scan(&constraintExists)
	require.NoError(t, err)
	assert.True(t, constraintExists, "uq_click_events_event_id constraint should exist")

	// Clean up before test to ensure idempotency
	_, _ = pool.Exec(ctx, "DELETE FROM click_events WHERE id = 'test-click-id'")

	// 5. Test nullable fields work correctly by inserting a row with NULLs
	_, err = pool.Exec(ctx, "INSERT INTO urls (id, short_code, original_url) VALUES ('test-url-id', 'test', 'http://test.com') ON CONFLICT DO NOTHING")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO click_events (id, event_id, url_id, country, device, browser, operating_system, referrer) 
		VALUES ('test-click-id', 'test-event-id', 'test-url-id', NULL, NULL, NULL, NULL, NULL)
		ON CONFLICT (event_id) DO NOTHING
	`)
	require.NoError(t, err, "Should be able to insert click_event with NULL analytics fields")

	// Clean up
	_, _ = pool.Exec(ctx, "DELETE FROM click_events WHERE id = 'test-click-id'")
	_, _ = pool.Exec(ctx, "DELETE FROM urls WHERE id = 'test-url-id'")
}
