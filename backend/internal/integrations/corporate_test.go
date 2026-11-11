package integrations_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/integrations"
	"ai-buddy/backend/internal/mockservices"
)

func TestHTTPClientWithMockServices(t *testing.T) {
	store := mockservices.NewStore(time.Millisecond)
	srv := httptest.NewServer(mockservices.Handler(store))
	defer srv.Close()
	c := integrations.NewHTTP(srv.URL+"/", nil)
	ctx := context.Background()

	auto, err := c.Submit(ctx, integrations.Order{Kind: "access", Requester: "anna", Service: "wiki", Auto: true})
	require.NoError(t, err)
	assert.Equal(t, integrations.TicketGranted, auto.Status)
	assert.NotNil(t, auto.DecidedAt)

	leave, err := c.Submit(ctx, integrations.Order{Kind: "leave", Requester: "anna", LeaveDate: "2026-12-03"})
	require.NoError(t, err)
	assert.Equal(t, integrations.TicketWaiting, leave.Status)
	assert.Contains(t, leave.Ref, "HR-")
	time.Sleep(5 * time.Millisecond)
	got, err := c.Ticket(ctx, leave.Ref)
	require.NoError(t, err)
	assert.Equal(t, integrations.TicketApproved, got.Status, "согласовано после задержки")

	_, err = c.Ticket(ctx, "ACC-1")
	assert.ErrorIs(t, err, integrations.ErrUnknownTicket)
	_, err = c.Submit(ctx, integrations.Order{Kind: "vacation"})
	assert.Error(t, err)

	rej, err := store.Decide(auto.Ref, false, " ")
	require.NoError(t, err)
	assert.Equal(t, "Отклонено согласующим", rej.Reason)

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not json")) }))
	defer down.Close()
	_, err = integrations.NewHTTP(down.URL, nil).Ticket(ctx, "x")
	assert.Error(t, err)
	_, err = integrations.NewHTTP("http://127.0.0.1:1", nil).Submit(ctx, integrations.Order{Kind: "access"})
	assert.Error(t, err)
}
