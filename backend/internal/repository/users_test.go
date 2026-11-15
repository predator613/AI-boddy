package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/database"
	"ai-buddy/backend/internal/testfix"
	"ai-buddy/backend/migrations"
)

func newRepo(t *testing.T) *Users {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, url, 10*time.Second)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	_, err = database.Migrate(ctx, pool, migrations.FS)
	require.NoError(t, err)
	require.NoError(t, testfix.OtherTeam(ctx, pool))
	return NewUsers(pool)
}

func TestGetByLogin(t *testing.T) {
	repo := newRepo(t)
	u, err := repo.GetByLogin(context.Background(), "intern_maria")
	require.NoError(t, err)
	assert.Equal(t, "Мария", u.Name)
	assert.Equal(t, "intern", u.Role)
	assert.Equal(t, "Backend", u.TeamName)
	require.NotNil(t, u.MentorID)
	assert.NotEmpty(t, u.PasswordHash)

	igor, err := repo.GetByLogin(context.Background(), "mentor_igor")
	require.NoError(t, err)
	assert.Equal(t, igor.ID, *u.MentorID, "Мария закреплена за Игорем")
	assert.Nil(t, igor.MentorID)

	_, err = repo.GetByLogin(context.Background(), "nobody")
	assert.ErrorIs(t, err, ErrNotFound)

	// Согласующий DevOps (роль staff) в приложение не входит.
	_, err = repo.GetByLogin(context.Background(), "devops_kirill")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetByID(t *testing.T) {
	repo := newRepo(t)
	anna, err := repo.GetByLogin(context.Background(), "intern_maria")
	require.NoError(t, err)

	u, err := repo.GetByID(context.Background(), anna.ID)
	require.NoError(t, err)
	assert.Equal(t, "intern_maria", u.Login)

	_, err = repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListTestUsers(t *testing.T) {
	repo := newRepo(t)
	list, err := repo.ListTestUsers(context.Background())
	require.NoError(t, err)

	logins := map[string]string{}
	for _, u := range list {
		logins[u.Login] = u.TeamName
	}
	assert.Equal(t, "Backend", logins["intern_maria"])
	assert.Equal(t, "Backend", logins["mentor_igor"])
	assert.Equal(t, "Backend", logins["lead_artemiy"])
	assert.Equal(t, "Frontend", logins["intern_pavel"])
	assert.NotContains(t, logins, "devops_kirill", "роль staff не показывается в демо-входе")
	for _, u := range list {
		assert.Contains(t, []string{"intern", "mentor", "lead"}, u.Role)
	}
}

func TestQueryErrorsWhenPoolClosed(t *testing.T) {
	repo := newRepo(t)
	repo.db.Close()

	_, err := repo.GetByLogin(context.Background(), "intern_maria")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)

	_, err = repo.GetByID(context.Background(), "x")
	assert.Error(t, err)

	_, err = repo.ListTestUsers(context.Background())
	assert.Error(t, err)
}
